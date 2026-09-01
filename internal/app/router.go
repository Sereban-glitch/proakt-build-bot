// Package app — маршрутизация обновлений и сценарии (FSM).
package app

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"proakt/internal/ai"
	"proakt/internal/config"
	"proakt/internal/domain"
	"proakt/internal/parse"
	"proakt/internal/store"
	"proakt/internal/tg"
	"proakt/internal/xlsx"
)

// Состояния FSM.
const (
	stIdle         = "idle"     // главное меню
	stObjName      = "obj_name" // ждём название объекта
	stObjCustomer  = "obj_customer"
	stActObj       = "act_obj"      // выбор объекта для акта (inline)
	stActLines     = "act_lines"    // ввод позиций
	stPayAmount    = "pay_amount"   // ввод суммы оплаты
	stPhotoPick    = "photo_pick"   // выбор привязки фото (inline)
	stPhotoWait    = "photo_wait"   // ждём само фото
	stPriceAdd     = "price_add"    // ждём строку «наименование [ед.] цена»
	stPriceImport  = "price_import" // ждём файл прайса
	stPriceClear   = "price_clear"  // подтверждение очистки прайса
	stPriceFind    = "price_find"   // ждём слово для поиска по прайсу (v0.3.5)
	stPriceDel     = "price_del"    // ждём название позиции для удаления (v0.3.5)
	stPriceDelConf = "price_del_ok" // подтверждение удаления одной позиции (v0.3.5)
	stActDelConf   = "act_del_ok"   // подтверждение удаления акта (v0.3.6)
)

type Bot struct {
	cfg   *config.Config
	tg    *tg.Client
	st    *store.Store
	files string
	ai    *ai.Gateway // голос и разбор позиций (nil — голос выключен)
}

func New(cfg *config.Config, c *tg.Client, st *store.Store, gw *ai.Gateway) *Bot {
	return &Bot{cfg: cfg, tg: c, st: st, files: cfg.FilesDir, ai: gw}
}

// Handle — обработать одно обновление (с защитой от паники).
func (b *Bot) Handle(ctx context.Context, u tg.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ПАНИКА в обработчике: %v", r)
		}
	}()
	if u.CallbackQuery != nil {
		b.onCallback(ctx, *u.CallbackQuery)
		return
	}
	if u.Message != nil {
		b.onMessage(ctx, *u.Message)
	}
}

// chatOf — чат обновления.
func chatOf(u tg.Update) int64 {
	if u.Message != nil {
		return u.Message.Chat.ID
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message != nil {
		return u.CallbackQuery.Message.Chat.ID
	}
	return 0
}

func (b *Bot) text(ctx context.Context, chatID int64, s string) {
	if err := b.tg.SendMessage(ctx, chatID, s, nil); err != nil {
		log.Printf("sendMessage чат %d: %v", chatID, err)
	}
}

func (b *Bot) textKB(ctx context.Context, chatID int64, s string, kb any) {
	if err := b.tg.SendMessage(ctx, chatID, s, kb); err != nil {
		log.Printf("sendMessage чат %d: %v", chatID, err)
	}
}

// --- сообщения ---------------------------------------------------------------

func (b *Bot) onMessage(ctx context.Context, m tg.Message) {
	chatID := m.Chat.ID
	if m.From == nil {
		return
	}
	// регистрируем пользователя (первый = админ)
	if _, err := b.st.EnsureUser(ctx, chatID, m.From.ID, m.From.FirstName, m.From.Username); err != nil {
		log.Printf("EnsureUser: %v", err)
	}

	// фото — глобальный вход (если не в середине другого сценария)
	if len(m.Photo) > 0 {
		b.onPhotoMessage(ctx, m)
		return
	}

	// голос — ввод позиций акта (v0.2)
	if m.Voice != nil {
		b.onVoiceMessage(ctx, m)
		return
	}

	// файл — импорт прайса (v0.3)
	if m.Document != nil {
		b.onDocumentMessage(ctx, m)
		return
	}

	text := strings.TrimSpace(m.Text)
	if text == "" {
		return
	}

	state, data, _ := b.st.State(ctx, chatID)

	// команды работают из любого состояния
	switch {
	case text == "/start":
		if b.guardDraft(ctx, chatID) {
			return
		}
		b.reset(ctx, chatID)
		// v0.3.8: сначала постоянное меню (крупные кнопки внизу), затем
		// приветствие с inline-кнопкой «📖 Инструкция по шагам» — новичок
		// в один тап получает весь маршрут: добавить, найти, удалить.
		b.textKB(ctx, chatID, "Главное меню — кнопки внизу экрана 👇", MainMenu())
		b.textKB(ctx, chatID, b.welcome(), InstrButton())
		return
	case text == "/help" || strings.Contains(text, "помощь") || text == BtnInstrMenu ||
		strings.EqualFold(text, "инструкция"):
		b.textKB(ctx, chatID, b.help(), MainMenu())
		return
	case text == "/cancel" || text == BtnCancel || isCancelText(text):
		if b.guardDraft(ctx, chatID) {
			return
		}
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Отменил. Возвращаюсь в меню.", MainMenu())
		return
	case text == "/new" || text == BtnNewAct:
		if b.guardDraft(ctx, chatID) {
			return
		}
		b.startNewAct(ctx, chatID)
		return
	case text == "/draft":
		b.showDraft(ctx, chatID)
		return
	case text == "/debts" || text == BtnDebts:
		b.showDebts(ctx, chatID)
		return
	case text == "/acts":
		b.showActs(ctx, chatID)
		return
	case text == "/objects" || text == BtnObjs:
		b.showObjects(ctx, chatID)
		return
	case text == "/photo" || text == BtnPhoto:
		if b.guardDraft(ctx, chatID) {
			return
		}
		b.startPhotoWait(ctx, chatID)
		return
	case text == "/report" || text == BtnReport:
		b.showReport(ctx, chatID)
		return
	case text == "/price" || text == BtnPrice:
		b.showPriceMenu(ctx, chatID)
		return
	case strings.HasPrefix(text, "/price "):
		b.priceFindFromText(ctx, chatID, strings.TrimSpace(strings.TrimPrefix(text, "/price")))
		return
	case text == BtnMore:
		b.textKB(ctx, chatID, "Ещё:", MoreMenu())
		return
	case text == BtnBack:
		b.textKB(ctx, chatID, "Главное меню", MainMenu())
		return
	}

	// остальное — по состоянию
	switch state {
	case stObjName:
		b.createObjectFromText(ctx, chatID, data, text)
	case stObjCustomer:
		b.setObjectCustomer(ctx, chatID, data, text)
	case stActLines:
		// v0.3.5: контроль черновика — убрать последнюю / показать всё
		if text == BtnUndo || isUndoText(text) {
			b.undoLastDraft(ctx, chatID, data)
			return
		}
		if text == BtnDraft || isDraftText(text) {
			b.showDraft(ctx, chatID)
			return
		}
		b.actLineFromText(ctx, chatID, data, text)
	case stPayAmount:
		b.paymentFromText(ctx, chatID, data, text)
	case stPhotoWait:
		b.text(ctx, chatID, "Жду фото 📷 — просто пришли снимок (можно с подписью).")
	case stPriceAdd:
		b.priceAddFromText(ctx, chatID, text)
	case stPriceImport:
		b.text(ctx, chatID, "Жду файл прайса — Excel (.xlsx) или CSV. Или ⏹ Отмена.")
	case stPriceClear:
		b.text(ctx, chatID, "Жду подтверждения кнопкой выше ⬆️")
	case stPriceFind:
		b.priceFindFromText(ctx, chatID, text)
	case stPriceDel:
		b.priceDelFromText(ctx, chatID, text)
	case stPriceDelConf:
		b.text(ctx, chatID, "Жду подтверждения кнопкой выше ⬆️ (или ⏹ Отмена)")
	case stActDelConf:
		b.text(ctx, chatID, "Жду подтверждения кнопкой выше ⬆️ (или ⏹ Отмена)")
	default:
		b.textKB(ctx, chatID, notUnderstood, MainMenu())
	}
}

// --- callback-кнопки ---------------------------------------------------------

func (b *Bot) onCallback(ctx context.Context, cq tg.CallbackQuery) {
	chatID := chatOf(tg.Update{CallbackQuery: &cq})
	if chatID == 0 {
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "")
		return
	}
	if _, err := b.st.EnsureUser(ctx, chatID, cq.From.ID, cq.From.FirstName, cq.From.Username); err != nil {
		log.Printf("EnsureUser: %v", err)
	}
	data := cq.Data
	_, stateData, _ := b.st.State(ctx, chatID)

	switch {
	case data == "cancel":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Отмена")
		if b.guardDraft(ctx, chatID) {
			return
		}
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Отменил. Главное меню.", MainMenu())

	case data == "dripyes":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Выбросил")
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "🗑 Черновик выброшен. Главное меню.", MainMenu())

	case data == "dripcno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Продолжаем")
		b.textKB(ctx, chatID, "👷 Продолжай вводить позиции — или ✅ Завершить акт.", ActMenu())

	case data == "vok":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Добавляю")
		b.applyVoiceLines(ctx, chatID, stateData)

	case data == "vox":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Отбросил")
		b.discardVoiceLines(ctx, chatID, stateData)

	case data == "plist":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Список")
		b.showPriceList(ctx, chatID)

	case data == "pfind":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Поиск")
		b.beginPriceFind(ctx, chatID)

	case data == "padd":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Добавить")
		b.beginPriceAdd(ctx, chatID)

	case data == "pdel":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Убрать позицию")
		b.beginPriceDel(ctx, chatID)

	case data == "pdelyes":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Убираю")
		b.priceDelConfirm(ctx, chatID, stateData["del_name"])

	case data == "pdelno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оставил")
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Оставил позицию в прайсе.", priceMenu())

	case data == "pimport":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Импорт")
		b.beginPriceImport(ctx, chatID)

	case data == "pclear":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Очистить")
		b.beginPriceClear(ctx, chatID)

	case data == "pclyes":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Удаляю")
		if err := b.st.ClearCatalog(ctx, chatID); err != nil {
			log.Printf("прайс: очистка чат %d: %v", chatID, err)
		}
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "🗑 Прайс очищен.", MainMenu())

	case data == "pclno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оставил")
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Оставил прайс как есть.", MainMenu())

	case data == "objnew":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Новый объект")
		b.beginObject(ctx, chatID, map[string]string{"from": ""})

	case strings.HasPrefix(data, "actobj:"):
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Поехали")
		objID, _ := strconv.ParseInt(strings.TrimPrefix(data, "actobj:"), 10, 64)
		if objID > 0 {
			b.beginActLines(ctx, chatID, objID)
		}

	case strings.HasPrefix(data, "pay:"):
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оплата")
		actID, _ := strconv.ParseInt(strings.TrimPrefix(data, "pay:"), 10, 64)
		if actID > 0 {
			b.beginPayment(ctx, chatID, map[string]string{"act_id": fmt.Sprint(actID)})
		}

	case strings.HasPrefix(data, "pha:"):
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "К акту")
		actID, _ := strconv.ParseInt(strings.TrimPrefix(data, "pha:"), 10, 64)
		b.attachPhoto(ctx, chatID, stateData, &actID, 0)

	case strings.HasPrefix(data, "pho:"):
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "К объекту")
		objID, _ := strconv.ParseInt(strings.TrimPrefix(data, "pho:"), 10, 64)
		b.attachPhoto(ctx, chatID, stateData, nil, objID)

	case strings.HasPrefix(data, "phv:"): // v0.3.7: показать фото объекта
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Показываю")
		objID, _ := strconv.ParseInt(strings.TrimPrefix(data, "phv:"), 10, 64)
		if objID > 0 {
			b.showPhotos(ctx, chatID, objID, 0)
		}

	case strings.HasPrefix(data, "pav:"): // v0.3.7: показать фото акта
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Показываю")
		actID, _ := strconv.ParseInt(strings.TrimPrefix(data, "pav:"), 10, 64)
		if actID > 0 {
			if brief, err := b.st.GetAct(ctx, actID); err == nil {
				b.showPhotos(ctx, chatID, brief.ObjectID, actID)
			} else {
				b.text(ctx, chatID, "Акт не найден 😕 Список: /acts")
			}
		}

	case data == "instr": // v0.3.8: инструкция по шагам — кнопка из приветствия
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Открываю")
		b.textKB(ctx, chatID, b.instruction(), MainMenu())

	case strings.HasPrefix(data, "phdel:"): // v0.3.8: удалить неудачное фото
		b.confirmDeletePhoto(ctx, chatID, cq.ID, cbID(data, "phdel:"))

	case strings.HasPrefix(data, "phdelyes:"):
		b.deletePhotoDo(ctx, chatID, cq.ID, cbID(data, "phdelyes:"))

	case data == "phdelno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оставил")
		b.text(ctx, chatID, "Фото на месте 📷 Ещё: /objects")

	case strings.HasPrefix(data, "plist:"): // v0.3.8: оплаты акта списком
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оплаты")
		b.showPayments(ctx, chatID, cbID(data, "plist:"))

	case strings.HasPrefix(data, "paydel:"): // v0.3.8: удалить ошибочную оплату
		b.confirmDeletePayment(ctx, chatID, cq.ID, cbID(data, "paydel:"))

	case strings.HasPrefix(data, "paydelyes:"):
		b.deletePaymentDo(ctx, chatID, cq.ID, cbID(data, "paydelyes:"))

	case data == "paydelno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оставил")
		b.text(ctx, chatID, "Оплата на месте. Акты: /acts")

	case data == "objpick": // v0.3.8: скрыть лишний объект
		b.pickObjectToHide(ctx, chatID, cq.ID)

	case strings.HasPrefix(data, "objdel:"):
		b.confirmHideObject(ctx, chatID, cq.ID, cbID(data, "objdel:"))

	case strings.HasPrefix(data, "objdelyes:"):
		b.hideObjectDo(ctx, chatID, cq.ID, cbID(data, "objdelyes:"))

	case data == "objdelno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оставил")
		b.text(ctx, chatID, "Объект на месте 🏠 Список: /objects")

	case data == "finyes": // v0.3.6: завершили акт с позициями без цены осознанно
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Завершаю")
		state, data2, _ := b.st.State(ctx, chatID)
		if state != stActLines {
			b.text(ctx, chatID, "Черновик уже не активен — начни заново: /new")
			return
		}
		lines := parseDraft(data2["lines"])
		if len(lines) == 0 {
			b.text(ctx, chatID, "Позиций нет — вводи заново: /new")
			return
		}
		b.finishActDo(ctx, chatID, data2, lines)

	case data == "finno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Продолжаем")
		b.textKB(ctx, chatID, "Продолжай ввод 👀 Черновик покажет всё, ↩️ уберёт последнюю.", ActMenu())

	case strings.HasPrefix(data, "adel:"): // v0.3.6: удаление ошибочного акта
		actID, _ := strconv.ParseInt(strings.TrimPrefix(data, "adel:"), 10, 64)
		b.confirmDeleteAct(ctx, chatID, cq.ID, actID)

	case data == "adelyes":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Удаляю")
		b.deleteActDo(ctx, chatID)

	case data == "adelno":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Оставил")
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Акт на месте 📋 Список: /acts", MainMenu())

	default:
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "")
	}
}

// reset — сброс FSM в idle.
func (b *Bot) reset(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stIdle, map[string]string{})
}

// --- контроль черновика (v0.3.5, «линза строителя») -----------------------------

// guardDraft — защита от потери черновика: если идёт ввод позиций и они уже есть,
// вместо молчаливого сброса спрашиваем подтверждение. true = спросили, действие отложено.
// Строитель надиктовал 20 минут голосом — /start не должен выбрасывать это в никуда.
func (b *Bot) guardDraft(ctx context.Context, chatID int64) bool {
	state, data, _ := b.st.State(ctx, chatID)
	if state != stActLines {
		return false
	}
	lines := parseDraft(data["lines"])
	if len(lines) == 0 {
		return false
	}
	b.textKB(ctx, chatID, fmt.Sprintf(
		"⚠️ В черновике акта %d позиций на %s — ввод ещё не завершён.\n\nТочно выбросить?",
		len(lines), money(sumDraft(lines))), tg.Inline(tg.KB{
		{tg.KBButton{Text: "🗑 Выбросить черновик", CallbackData: "dripyes"}},
		{tg.KBButton{Text: "👷 Продолжить ввод", CallbackData: "dripcno"}},
	}))
	return true
}

// undoLastDraft — «↩️ Убрать последнюю»: ошибка при вводе — не начинать акт заново.
func (b *Bot) undoLastDraft(ctx context.Context, chatID int64, data map[string]string) {
	rest, removed, ok := removeLastDraft(data)
	if !ok {
		b.text(ctx, chatID, "Черновик пуст — нечего убирать. Введи позицию строкой или голосом 🎤")
		return
	}
	data["lines"] = linesJSON(rest)
	_ = b.st.SetState(ctx, chatID, stActLines, data)
	var sb strings.Builder
	sb.WriteString("↩️ Убрал:\n" + describeLine(removed))
	if len(rest) > 0 {
		sb.WriteString(fmt.Sprintf("\n—\nОсталось: %d позиций · %s", len(rest), money(sumDraft(rest))))
	} else {
		sb.WriteString("\n—\nЧерновик пуст — вводи заново.")
	}
	b.text(ctx, chatID, sb.String())
}

// showDraft — «👀 Черновик» (/draft): все позиции акта в работе + промежуточная сумма.
// Это и есть «предварительный акт» строителя: сколько уже набежало по объекту.
func (b *Bot) showDraft(ctx context.Context, chatID int64) {
	state, data, _ := b.st.State(ctx, chatID)
	lines := parseDraft(data["lines"])
	if state != stActLines {
		b.textKB(ctx, chatID, "Черновика нет. Начать акт: 📋 Новый акт", MainMenu())
		return
	}
	if len(lines) == 0 {
		// ввод уже начат (объект выбран) — не сбиваем с толку «черновика нет»
		b.textKB(ctx, chatID, "В акте пока нет позиций — вводи первую строкой или голосом 🎤", ActMenu())
		return
	}
	objName := "объект"
	if id, err := strconv.ParseInt(data["object_id"], 10, 64); err == nil {
		if o, err := b.st.GetObject(ctx, id); err == nil {
			objName = o.Name
		}
	}
	// чанками по 40 позиций — лимит Telegram 4096 символов на сообщение
	const chunk = 40
	for start := 0; start < len(lines); start += chunk {
		end := start + chunk
		if end > len(lines) {
			end = len(lines)
		}
		var sb strings.Builder
		if start == 0 {
			sb.WriteString(fmt.Sprintf("👀 Черновик акта — «%s»:\n\n", objName))
		}
		for i := start; i < end; i++ {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, describeLine(lines[i])))
		}
		if end == len(lines) {
			sb.WriteString(fmt.Sprintf("\nПозиций: %d · сумма: %s\n✅ Завершить акт — пришлю Excel · ↩️ Убрать последнюю — если ошибся",
				len(lines), money(sumDraft(lines))))
			b.textKB(ctx, chatID, sb.String(), ActMenu())
		} else {
			b.text(ctx, chatID, sb.String())
		}
	}
}

// --- объекты -----------------------------------------------------------------

func (b *Bot) beginObject(ctx context.Context, chatID int64, data map[string]string) {
	_ = b.st.SetState(ctx, chatID, stObjName, data)
	b.textKB(ctx, chatID, "Как назовём объект?\nНапример: ЖК Сонячний, кв. 45", CancelMenu())
}

func (b *Bot) createObjectFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	name := strings.TrimSpace(text)
	if len([]rune(name)) < 2 || len([]rune(name)) > 60 {
		b.text(ctx, chatID, "Название должно быть от 2 до 60 символов. Попробуй ещё:")
		return
	}
	data["name"] = name
	_ = b.st.SetState(ctx, chatID, stObjCustomer, data)
	b.text(ctx, chatID, "Заказчик? (имя или организация — или «-», чтобы пропустить)")
}

func (b *Bot) setObjectCustomer(ctx context.Context, chatID int64, data map[string]string, text string) {
	customer := strings.TrimSpace(text)
	if customer == "-" || strings.EqualFold(customer, "нет") {
		customer = ""
	}
	obj, err := b.st.CreateObject(ctx, chatID, data["name"], customer)
	if err != nil {
		b.text(ctx, chatID, "Не получилось сохранить объект 😕 Попробуй ещё раз: /objects")
		b.reset(ctx, chatID)
		return
	}
	b.reset(ctx, chatID)
	b.textKB(ctx, chatID, fmt.Sprintf("🏠 Объект «%s» создан%s\n\nМожно создавать акт: 📋 Новый акт",
		obj.Name, custSuffix(obj.Customer)), MainMenu())
}

func custSuffix(c string) string {
	if c == "" {
		return ""
	}
	return " (заказчик: " + c + ")"
}

func (b *Bot) showObjects(ctx context.Context, chatID int64) {
	objs, err := b.st.ListObjects(ctx, chatID)
	if err != nil {
		b.text(ctx, chatID, "Не смог прочитать объекты 😕")
		return
	}
	if len(objs) == 0 {
		b.beginObject(ctx, chatID, map[string]string{})
		return
	}
	var sb strings.Builder
	sb.WriteString("🏠 Твои объекты:\n\n")
	rows := tg.KB{}
	for _, o := range objs {
		sb.WriteString(fmt.Sprintf("• %s%s\n", o.Name, custSuffix(o.Customer)))
		if o.Acts > 0 {
			line := fmt.Sprintf("   актов %d · выполнено %s", o.Acts, money(o.Total))
			if o.Debt() > 0.009 {
				line += fmt.Sprintf(" · долг %s", money(o.Debt()))
			} else if o.Paid > 0.009 {
				line += " · оплачено ✅"
			}
			if o.Photos > 0 {
				line += fmt.Sprintf(" · 📷 %d", o.Photos)
			}
			sb.WriteString(line + "\n")
		} else {
			sb.WriteString("   актов ещё нет\n")
		}
		if o.Photos > 0 { // v0.3.7: скрытые работы можно показать в любой момент
			rows = append(rows, []tg.KBButton{{
				Text:         photoBtnText(o.Name, o.Photos),
				CallbackData: fmt.Sprintf("phv:%d", o.ID),
			}})
		}
	}
	sb.WriteString("\n(это предварительные итоги — акты появляются по ходу работ)")
	rows = append(rows, []tg.KBButton{{Text: BtnNewObj, CallbackData: "objnew"}})
	rows = append(rows, []tg.KBButton{{Text: "🗑 Убрать объект", CallbackData: "objpick"}}) // v0.3.8
	b.textKB(ctx, chatID, sb.String(), tg.Inline(rows))
}

// photoBtnText — подпись кнопки показа фото (лимит Telegram — 64 байта).
func photoBtnText(name string, n int) string {
	const budget = 44 // 64 − «📷 »(5) − « — »(5) − «NN фото»(≤9)
	label := name
	for len(label) > budget { // бьём по рунам, чтобы не разрезать UTF-8
		label = string([]rune(label)[:len([]rune(label))-1])
	}
	return fmt.Sprintf("📷 %s — %d фото", label, n)
}

// --- акты --------------------------------------------------------------------

func (b *Bot) startNewAct(ctx context.Context, chatID int64) {
	objs, err := b.st.ListObjects(ctx, chatID)
	if err != nil || len(objs) == 0 {
		b.textKB(ctx, chatID, "Сначала создадим объект — куда записывать работы.", CancelMenu())
		b.beginObject(ctx, chatID, map[string]string{"from": "act"})
		return
	}
	rows := []tg.KBButton{}
	for _, o := range objs {
		rows = append(rows, tg.KBButton{Text: "🏠 " + o.Name, CallbackData: fmt.Sprintf("actobj:%d", o.ID)})
	}
	rows = append(rows,
		tg.KBButton{Text: BtnNewObj, CallbackData: "objnew"},
		tg.KBButton{Text: BtnCancel, CallbackData: "cancel"})
	b.textKB(ctx, chatID, "По какому объекту делаем акт?", tg.Inline(tg.KB{rows}))
}

func (b *Bot) beginActLines(ctx context.Context, chatID int64, objectID int64) {
	_ = b.st.SetState(ctx, chatID, stActLines, map[string]string{"object_id": fmt.Sprint(objectID), "lines": "[]"})
	obj, err := b.st.GetObject(ctx, objectID)
	name := "объект"
	if err == nil {
		name = obj.Name
	}
	how := "Вводи позиции построчно:\nштукатурка 45 м² 260\nили: демонтаж 2000"
	if n, err := b.st.CatalogCount(ctx, chatID); err == nil && n > 0 {
		how += "\nили без цены: штукатурка 45 м² — возьму из прайса 💵"
	} else {
		how += "\n\n💡 Прайс пуст: позиция без цены станет 0 грн.\nЗагрузи свои цены: 💵 Прайс → 📥 Импорт из файла."
	}
	if b.ai != nil {
		how += "\n\nИли надиктуй голосом 🎤 — одной фразой или несколько подряд."
	}
	how += "\nОшибся — ↩️ Убрать последнюю. Посмотреть всё — 👀 Черновик."
	how += "\n\nКогда закончишь — жми ✅ Завершить акт (или напиши «завершить»)."
	b.textKB(ctx, chatID, fmt.Sprintf("📋 Акт для «%s».\n\n%s", name, how), ActMenu())
}

func (b *Bot) actLineFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	// защита от случайного нажатия текста-кнопки и синонимы обычным текстом
	if text == BtnFinish || isFinishText(text) {
		b.finishAct(ctx, chatID, data)
		return
	}
	// v0.3.6: несколько позиций в одной строке — «шлифовка 45 м грунтовка 45 м»
	if lines, fromCat, isMulti := b.parseMultiPosition(ctx, chatID, text); isMulti {
		if len(lines) > 0 {
			b.addActLines(ctx, chatID, data, lines, fromCat)
			return
		}
		b.text(ctx, chatID, multiFailText)
		return
	}
	// v0.3: «наименование количество [ед.]» без цены — цену ищем в прайсе
	if name, qty, unit, ok := parse.ParseQty(text); ok {
		if line, ok2 := b.lineFromCatalog(ctx, chatID, name, qty, unit); ok2 {
			b.addActLine(ctx, chatID, data, line, "💡 цена из прайса")
			return
		}
		b.text(ctx, chatID, fmt.Sprintf("В прайсе нет «%s» 🤔\nВведи с ценой: %s %s 260\nили добавь в прайс: 💵 Прайс → ➕ Добавить",
			name, name, unit))
		return
	}
	line, ok := parse.ParsePosition(text)
	if !ok {
		b.text(ctx, chatID, "Не разобрал строку 🤔\nПримеры:\nштукатурка 45 м² 260\nдемонтаж 2000\n\nИли без цены — тогда она возьмётся из прайса:\nштукатурка 45 м²")
		return
	}
	// v0.3.6: «двусмысленные» строки (несколько чисел с словами между ними)
	// не разбираем молча — количество/цена наугад = неверные деньги в акте
	if parse.SuspiciousMulti(text) {
		b.text(ctx, chatID, suspiciousText)
		return
	}
	b.addActLine(ctx, chatID, data, line, "")
}

// lineFromCatalog — позиция с ценой из прайса.
func (b *Bot) lineFromCatalog(ctx context.Context, chatID int64, name string, qty float64, unit string) (domain.DraftLine, bool) {
	items, err := b.st.ListCatalog(ctx, chatID)
	if err != nil {
		log.Printf("прайс: чтение чат %d: %v", chatID, err)
		return domain.DraftLine{}, false
	}
	it, ok := MatchCatalog(items, name)
	if !ok {
		return domain.DraftLine{}, false
	}
	if unit == "" {
		unit = it.Unit
	}
	return domain.DraftLine{Name: name, Qty: qty, Unit: unit, Price: it.Price, Sum: qty * it.Price}, true
}

// addActLine — добавить позицию в черновик и отчитаться.
func (b *Bot) addActLine(ctx context.Context, chatID int64, data map[string]string, line domain.DraftLine, note string) {
	lines := appendDraft(data, line)
	if len(lines) > 200 {
		b.text(ctx, chatID, "Лимит 200 позиций в акте — завершаем 🙃")
		return
	}
	data["lines"] = linesJSON(lines)
	_ = b.st.SetState(ctx, chatID, stActLines, data)
	_ = b.tg.SendChatAction(ctx, chatID)
	reply := fmt.Sprintf("✅ %s\n—\nПозиций: %d · сумма: %s",
		describeLine(line), len(lines), money(sumDraft(lines)))
	if note != "" {
		reply += "\n" + note
	}
	b.text(ctx, chatID, reply)
}

func (b *Bot) finishAct(ctx context.Context, chatID int64, data map[string]string) {
	lines := parseDraft(data["lines"])
	if len(lines) == 0 {
		b.text(ctx, chatID, "Позиций пока нет — вводи хотя бы одну строкой.")
		return
	}
	// v0.3.6: позиции без цены — акт выйдет на 0 грн; не молчим, а спрашиваем
	if n, names := zeroPriceLines(lines); n > 0 {
		sh := names
		if len(sh) > 3 {
			sh = sh[:3]
		}
		b.textKB(ctx, chatID, fmt.Sprintf(
			"⚠️ В акте %d из %d позиций без цены (в акте станет 0 грн):\n• %s\n\nКак быть?",
			n, len(lines), strings.Join(sh, "\n• ")), tg.Inline(tg.KB{
			{tg.KBButton{Text: "✅ Завершить как есть", CallbackData: "finyes"}},
			{tg.KBButton{Text: "✍️ Продолжить ввод", CallbackData: "finno"}},
		}))
		return
	}
	b.finishActDo(ctx, chatID, data, lines)
}

// finishActDo — само завершение: объект, запись, XLSX, сводка.
func (b *Bot) finishActDo(ctx context.Context, chatID int64, data map[string]string, lines []domain.DraftLine) {
	objID, _ := strconv.ParseInt(data["object_id"], 10, 64)
	obj, err := b.st.GetObject(ctx, objID)
	if err != nil {
		b.text(ctx, chatID, "Объект не найден 😕 начни заново: /new")
		b.reset(ctx, chatID)
		return
	}
	act, err := b.st.CreateAct(ctx, obj.ID, lines)
	if err != nil {
		b.text(ctx, chatID, "Не сохранил акт 😕 попробуй ещё раз /new")
		b.reset(ctx, chatID)
		return
	}
	b.reset(ctx, chatID)

	// сводка + XLSX
	total := sumDraft(lines)
	_ = b.tg.SendChatAction(ctx, chatID)
	brief := domain.ActBrief{ID: act.ID, ActNo: act.ActNo, ObjectID: obj.ID, ObjectName: obj.Name, Customer: obj.Customer, Total: total, Paid: 0}
	path, err := xlsx.GenerateAct(
		b.files+"/acts", brief, obj, draftToLines(lines), b.cfg.Executor, time.Now())
	if err != nil {
		log.Printf("xlsx: %v", err)
		b.textKB(ctx, chatID, fmt.Sprintf("📋 Акт №%d по «%s» записан.\nСумма: %s\nExcel-файл не собрался 😕 — но данные сохранены.",
			act.ActNo, obj.Name, money(total)), MainMenu())
		return
	}
	caption := fmt.Sprintf("Акт №%d · %s · %s", act.ActNo, obj.Name, money(total))
	if err := b.tg.SendDocumentFile(ctx, chatID, path, caption); err != nil {
		log.Printf("sendDocument: %v", err)
		b.text(ctx, chatID, "Файл не отправился, но акт записан 📋")
	}
	b.textKB(ctx, chatID, fmt.Sprintf(
		"📋 Акт №%d по «%s» готов: %s\nОплату отметить можно в 💰 Долги.",
		act.ActNo, obj.Name, money(total)), MainMenu())
}

// --- долги и оплаты ------------------------------------------------------------

func (b *Bot) showDebts(ctx context.Context, chatID int64) {
	acts, err := b.st.ListActs(ctx, chatID, 50)
	if err != nil {
		b.text(ctx, chatID, "Не смог прочитать акты 😕")
		return
	}
	var withDebt []domain.ActBrief
	for _, a := range acts {
		if a.Balance() > 0.009 {
			withDebt = append(withDebt, a)
		}
	}
	if len(withDebt) == 0 {
		b.textKB(ctx, chatID, "💪 Долгов нет — всё оплачено!", MainMenu())
		return
	}
	var sb strings.Builder
	sb.WriteString("💰 Должники:\n\n")
	rows := []tg.KBButton{}
	for _, a := range withDebt {
		sb.WriteString(fmt.Sprintf("• Акт №%d (%s): %s\n", a.ActNo, a.ObjectName, money(a.Balance())))
		rows = append(rows, tg.KBButton{
			Text:         fmt.Sprintf("💵 Оплата · акт №%d", a.ActNo),
			CallbackData: fmt.Sprintf("pay:%d", a.ID),
		})
	}
	sb.WriteString("\nНажми кнопку, чтобы отметить оплату.")
	b.textKB(ctx, chatID, sb.String(), tg.Inline(tg.KB{rows}))
}

func (b *Bot) beginPayment(ctx context.Context, chatID int64, data map[string]string) {
	_ = b.st.SetState(ctx, chatID, stPayAmount, data)
	actID, _ := strconv.ParseInt(data["act_id"], 10, 64)
	brief, err := b.st.GetAct(ctx, actID)
	if err != nil {
		b.text(ctx, chatID, "Акт не найден 😕")
		b.reset(ctx, chatID)
		return
	}
	b.textKB(ctx, chatID, fmt.Sprintf("Акт №%d · %s\nОплачено: %s из %s\n\nСколько получил? (числом, например 10000)",
		brief.ActNo, brief.ObjectName, money(brief.Paid), money(brief.Total)), CancelMenu())
}

func (b *Bot) paymentFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	val := parseAmount(text)
	if val <= 0 {
		b.text(ctx, chatID, "Не понял сумму. Введи числом, например: 5000")
		return
	}
	actID, _ := strconv.ParseInt(data["act_id"], 10, 64)
	if err := b.st.CreatePayment(ctx, actID, val, ""); err != nil {
		b.text(ctx, chatID, "Не записал оплату 😕")
		b.reset(ctx, chatID)
		return
	}
	brief, err := b.st.GetAct(ctx, actID)
	b.reset(ctx, chatID)
	if err != nil {
		b.textKB(ctx, chatID, fmt.Sprintf("💵 Оплата %s записана.", money(val)), MainMenu())
		return
	}
	bal := brief.Balance()
	if bal > 0.009 {
		b.textKB(ctx, chatID, fmt.Sprintf("💵 Оплата %s записана.\nАкт №%d: остаток %s",
			money(val), brief.ActNo, money(bal)), MainMenu())
	} else {
		b.textKB(ctx, chatID, fmt.Sprintf("💵 Оплата %s записана.\nАкт №%d закрыт полностью 🎉",
			money(val), brief.ActNo), MainMenu())
	}
}

// --- акты (список и удаление) — см. acts.go ----------------------------------------

// --- отчёт -----------------------------------------------------------------------

// showReport — сводка + «упущенная выгода» (v0.3.7, инсайт из видео:
// мастер должен СРАЗУ видеть, какие деньги проходят мимо кармана).
func (b *Bot) showReport(ctx context.Context, chatID int64) {
	stats, err := b.st.Stats(ctx, chatID)
	if err != nil {
		b.text(ctx, chatID, "Не собралась статистика 😕")
		return
	}
	priceN, _ := b.st.CatalogCount(ctx, chatID)
	b.textKB(ctx, chatID, reportText(stats, priceN), MainMenu())
}

// reportText — текст сводки (вынесен для тестов).
func reportText(st store.Stats, priceN int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "📊 Сводка:\n\n🏠 Объектов: %d\n📋 Актов: %d\n📷 Фото: %d\n💵 Прайс: %d позиций\n\nВыполнено: %s\nОплачено: %s\n",
		st.Objects, st.Acts, st.Photos, priceN, money(st.Total), money(st.Paid))

	debt := st.Total - st.Paid
	switch {
	case debt > 0.009 || st.ZeroActs > 0:
		sb.WriteString("\n⚠️ Упущенная выгода — деньги мимо кармана:\n")
		if debt > 0.009 {
			actsWord := "актов"
			if st.UnpaidActs%10 == 1 && st.UnpaidActs%100 != 11 {
				actsWord = "акт"
			} else if st.UnpaidActs%10 >= 2 && st.UnpaidActs%10 <= 4 && (st.UnpaidActs%100 < 10 || st.UnpaidActs%100 >= 20) {
				actsWord = "акта"
			}
			fmt.Fprintf(&sb, "• не оплачено %s (%d %s) — напомни заказчику: 💰 Долги\n",
				money(debt), st.UnpaidActs, actsWord)
		}
		if st.ZeroActs > 0 {
			zeroWord := "акты с суммой 0"
			if st.ZeroActs == 1 {
				zeroWord = "акт с суммой 0"
			}
			fmt.Fprintf(&sb, "• %d %s — работы сделаны, деньги не выставлены: оцени по прайсу\n",
				st.ZeroActs, zeroWord)
		}
	default:
		sb.WriteString("\n✅ Долгов нет, все акты с ценой — деньги под контролем!")
	}
	return sb.String()
}
