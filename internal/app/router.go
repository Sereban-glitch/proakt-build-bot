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
	stIdle        = "idle"     // главное меню
	stObjName     = "obj_name" // ждём название объекта
	stObjCustomer = "obj_customer"
	stActObj      = "act_obj"      // выбор объекта для акта (inline)
	stActLines    = "act_lines"    // ввод позиций
	stPayAmount   = "pay_amount"   // ввод суммы оплаты
	stPhotoPick   = "photo_pick"   // выбор привязки фото (inline)
	stPhotoWait   = "photo_wait"   // ждём само фото
	stPriceAdd    = "price_add"    // ждём строку «наименование [ед.] цена»
	stPriceImport = "price_import" // ждём файл прайса
	stPriceClear  = "price_clear"  // подтверждение очистки прайса
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
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, b.welcome(), MainMenu())
		return
	case text == "/help" || strings.Contains(text, "помощь"):
		b.textKB(ctx, chatID, b.help(), MainMenu())
		return
	case text == "/cancel" || text == BtnCancel || isCancelText(text):
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Отменил. Возвращаюсь в меню.", MainMenu())
		return
	case text == "/new" || text == BtnNewAct:
		b.startNewAct(ctx, chatID)
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
		b.startPhotoWait(ctx, chatID)
		return
	case text == "/report" || text == BtnReport:
		b.showReport(ctx, chatID)
		return
	case text == "/price" || text == BtnPrice:
		b.showPriceMenu(ctx, chatID)
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
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Отменил. Главное меню.", MainMenu())

	case data == "vok":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Добавляю")
		b.applyVoiceLines(ctx, chatID, stateData)

	case data == "vox":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Отбросил")
		b.discardVoiceLines(ctx, chatID, stateData)

	case data == "plist":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Список")
		b.showPriceList(ctx, chatID)

	case data == "padd":
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "Добавить")
		b.beginPriceAdd(ctx, chatID)

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

	default:
		_ = b.tg.AnswerCallbackQuery(ctx, cq.ID, "")
	}
}

// reset — сброс FSM в idle.
func (b *Bot) reset(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stIdle, map[string]string{})
}

// --- объекты -----------------------------------------------------------------

func (b *Bot) beginObject(ctx context.Context, chatID int64, data map[string]string) {
	_ = b.st.SetState(ctx, chatID, stObjName, data)
	b.textKB(ctx, chatID, "Как назовём объект?\nНапример: ЖК «Сонячний», кв. 45", CancelMenu())
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
	for _, o := range objs {
		sb.WriteString(fmt.Sprintf("• %s%s\n", o.Name, custSuffix(o.Customer)))
	}
	b.textKB(ctx, chatID, sb.String(), tg.Inline(tg.KB{{tg.KBButton{Text: BtnNewObj, CallbackData: "objnew"}}}))
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
	}
	if b.ai != nil {
		how += "\n\nИли надиктуй голосом 🎤 — одной фразой или несколько подряд."
	}
	how += "\n\nКогда закончишь — жми ✅ Завершить акт (или напиши «завершить»)."
	b.textKB(ctx, chatID, fmt.Sprintf("📋 Акт для «%s».\n\n%s", name, how), ActMenu())
}

func (b *Bot) actLineFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	// защита от случайного нажатия текста-кнопки и синонимы обычным текстом
	if text == BtnFinish || isFinishText(text) {
		b.finishAct(ctx, chatID, data)
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

// --- акты (список) --------------------------------------------------------------

func (b *Bot) showActs(ctx context.Context, chatID int64) {
	acts, err := b.st.ListActs(ctx, chatID, 20)
	if err != nil || len(acts) == 0 {
		b.textKB(ctx, chatID, "Актов пока нет. Создай первый: 📋 Новый акт", MainMenu())
		return
	}
	var sb strings.Builder
	sb.WriteString("📋 Последние акты:\n\n")
	for _, a := range acts {
		status := "✅ оплачен"
		if a.Balance() > 0.009 {
			status = "долг " + money(a.Balance())
		}
		sb.WriteString(fmt.Sprintf("• №%d (%s) — %s · %s\n", a.ActNo, a.ObjectName, money(a.Total), status))
	}
	b.textKB(ctx, chatID, sb.String(), MainMenu())
}

// --- отчёт -----------------------------------------------------------------------

func (b *Bot) showReport(ctx context.Context, chatID int64) {
	stats, err := b.st.Stats(ctx, chatID)
	if err != nil {
		b.text(ctx, chatID, "Не собралась статистика 😕")
		return
	}
	priceN, _ := b.st.CatalogCount(ctx, chatID)
	b.textKB(ctx, chatID, fmt.Sprintf(
		"📊 Сводка:\n\n🏠 Объектов: %d\n📋 Актов: %d\n📷 Фото: %d\n💵 Прайс: %d позиций\n\nВыполнено: %s\nОплачено: %s\nДолг: %s",
		stats.Objects, stats.Acts, stats.Photos, priceN,
		money(stats.Total), money(stats.Paid), money(stats.Total-stats.Paid)), MainMenu())
}
