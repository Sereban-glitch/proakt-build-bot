// Прайс-лист (v0.3): каталог цен мастера, автоподстановка цен в акты,
// импорт из Excel/CSV, добавление строкой. Всё в рамках chat_id.
package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"proakt/internal/domain"
	"proakt/internal/parse"
	"proakt/internal/pricefile"
	"proakt/internal/tg"
)

// лимит файла прайса (XLSX из Google Таблиц ~десятки КБ — 5 МБ с запасом)
const priceFileMaxBytes = 5 << 20

// MatchCatalog — чистый поиск позиции в прайсе по наименованию:
// точное совпадение → прайс-наименование содержится в запросе → запрос в прайс-наименовании.
func MatchCatalog(items []domain.CatalogItem, query string) (domain.CatalogItem, bool) {
	q := parse.NormName(query)
	if q == "" {
		return domain.CatalogItem{}, false
	}
	for _, it := range items { // точное
		if it.Name == q {
			return it, true
		}
	}
	if len([]rune(q)) >= 4 { // вхождение (обе стороны)
		for _, it := range items {
			if len([]rune(it.Name)) >= 3 && (strings.Contains(it.Name, q) || strings.Contains(q, it.Name)) {
				return it, true
			}
		}
	}
	return domain.CatalogItem{}, false
}

// fillFromCatalog — проставить цены позициям без цены (голос/текст).
// Возвращает позиции с ценой и список имён, для которых цены не нашлось.
func fillFromCatalog(items []domain.CatalogItem, lines []domain.DraftLine) ([]domain.DraftLine, []string) {
	var out []domain.DraftLine
	var missing []string
	for _, l := range lines {
		if l.Price > 0 && l.Sum > 0 {
			out = append(out, l)
			continue
		}
		if l.Qty > 0 && l.Price == 0 {
			if it, ok := MatchCatalog(items, l.Name); ok {
				l.Price = it.Price
				l.Sum = l.Qty * it.Price
				if l.Unit == "" {
					l.Unit = it.Unit
				}
				out = append(out, l)
				continue
			}
		}
		missing = append(missing, l.Name)
	}
	return out, missing
}

// --- меню прайса --------------------------------------------------------------

// showPriceMenu — вход в раздел.
func (b *Bot) showPriceMenu(ctx context.Context, chatID int64) {
	n, _ := b.st.CatalogCount(ctx, chatID)
	b.textKB(ctx, chatID, fmt.Sprintf("💵 Прайс — твои цены.\nПозиций: %d\n\nЗаполни один раз — и в акте пиши без цены:\n«штукатурка 45 м²» — цену возьму отсюда.\n\nЦена изменилась? Напиши её заново тем же названием —\nпокажу «было → стало» (цены теперь не дрейфуют молча).", n), priceMenu())
}

func priceMenu() tg.InlineKeyboardMarkup {
	return tg.Inline(tg.KB{
		{tg.KBButton{Text: "📄 Список", CallbackData: "plist"}, tg.KBButton{Text: "🔍 Найти", CallbackData: "pfind"}},
		{tg.KBButton{Text: "➕ Добавить/обновить", CallbackData: "padd"}, tg.KBButton{Text: "🗑 Убрать позицию", CallbackData: "pdel"}},
		{tg.KBButton{Text: "📥 Импорт из файла", CallbackData: "pimport"}, tg.KBButton{Text: "🧹 Очистить всё", CallbackData: "pclear"}},
		// v0.6: Google Таблица — источник истины + обратная выгрузка
		{tg.KBButton{Text: "🌐 Google Таблица", CallbackData: "psheet"}, tg.KBButton{Text: "📤 Выгрузить CSV", CallbackData: "pexport"}},
	})
}

// showPriceList — список позиций (первые 100).
func (b *Bot) showPriceList(ctx context.Context, chatID int64) {
	items, err := b.st.ListCatalog(ctx, chatID)
	if err != nil {
		b.text(ctx, chatID, "Не смог прочитать прайс 😕")
		return
	}
	if len(items) == 0 {
		b.text(ctx, chatID, "Прайс пуст. ➕ Добавь цену или 📥 импортируй файл.")
		return
	}
	var sb strings.Builder
	sb.WriteString("💵 Прайс:\n\n")
	for i, it := range items {
		if i >= 100 {
			sb.WriteString(fmt.Sprintf("\n… и ещё %d позиций — быстрее найти через 🔍 Найти", len(items)-100))
			break
		}
		sb.WriteString(fmt.Sprintf("• %s — %s%s\n", it.Name, money(it.Price), unitSuffix(it.Unit)))
	}
	b.textKB(ctx, chatID, sb.String(), priceMenu())
}

func unitSuffix(u string) string {
	if u == "" {
		return ""
	}
	return "/" + u
}

// beginPriceAdd — «➕ Добавить/обновить».
func (b *Bot) beginPriceAdd(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceAdd, map[string]string{})
	b.textKB(ctx, chatID, "Напиши позицию с ценой:\n\nштукатурка 260\nштукатурка м² 260\nдемонтаж стен 2000\n\nЕсли позиция уже есть — обновлю цену и покажу «было → стало».", CancelMenu())
}

// priceAddFromText — разбор строки добавления (v0.3.5: обновление видно явно).
func (b *Bot) priceAddFromText(ctx context.Context, chatID int64, text string) {
	item, ok := parse.ParseCatalogItem(text)
	if !ok {
		b.text(ctx, chatID, "Не разобрал 🤔 Примеры:\nштукатурка 260\nштукатурка м² 260")
		return
	}
	prev, existed, err := b.st.UpsertCatalogItem(ctx, chatID, item.Name, item.Unit, item.Price)
	if err != nil {
		log.Printf("прайс: upsert чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не записал в прайс 😕 Попробуй ещё раз.")
		return
	}
	n, _ := b.st.CatalogCount(ctx, chatID)
	var msg string
	switch {
	case existed && prev != item.Price:
		msg = fmt.Sprintf("🔄 Обновил: было %s → стало %s\n%s — %s%s\nПозиций: %d\n\nАкты, уже созданные раньше, не трогаю — новая цена пойдёт в следующие.",
			money(prev), money(item.Price), item.Name, money(item.Price), unitSuffix(item.Unit), n)
	case existed:
		msg = fmt.Sprintf("✅ %s — %s%s (цена та же)\nПозиций: %d", item.Name, money(item.Price), unitSuffix(item.Unit), n)
	default:
		msg = fmt.Sprintf("✅ Прайс: %s — %s%s\nПозиций: %d", item.Name, money(item.Price), unitSuffix(item.Unit), n)
	}
	b.textKB(ctx, chatID, msg, priceMenu())
}

// beginPriceImport — «📥 Импорт из файла».
func (b *Bot) beginPriceImport(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceImport, map[string]string{})
	b.textKB(ctx, chatID, "Пришли файл Excel (.xlsx) или CSV — разберу и добавлю цены.\n\nВыгрузка из Google Таблиц: Файл → Скачать → Microsoft Excel (.xlsx).", CancelMenu())
}

// onDocumentMessage — пришёл файл.
func (b *Bot) onDocumentMessage(ctx context.Context, m tg.Message) {
	chatID := m.Chat.ID
	state, _, _ := b.st.State(ctx, chatID)
	if state != stPriceImport {
		b.text(ctx, chatID, "Чтобы загрузить прайс файлом: 💵 Прайс → 📥 Импорт из файла.")
		return
	}
	d := m.Document
	if d == nil {
		return
	}
	if d.FileSize > priceFileMaxBytes {
		b.text(ctx, chatID, "Файл великоват (лимит 5 МБ) 🙃")
		return
	}
	_ = b.tg.SendChatAction(ctx, chatID)
	data, err := b.tg.DownloadFileBytes(ctx, d.FileID, priceFileMaxBytes)
	if err != nil {
		log.Printf("прайс: скачивание чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не смог скачать файл 😕 Попробуй ещё раз.")
		return
	}
	items, err := pricefile.Parse(data, d.FileName)
	if err != nil {
		log.Printf("прайс: разбор чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не разобрал файл 😕 "+err.Error())
		return
	}
	n, err := b.st.BulkUpsertCatalog(ctx, chatID, items)
	if err != nil {
		log.Printf("прайс: импорт чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не записал прайс 😕")
		return
	}
	b.reset(ctx, chatID)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Импортировал позиций: %d из файла (всего в прайсе: %d)\nСовпадения с существующими — обновлены.\n\nПервые:", n, n))
	for i, it := range items {
		if i >= 5 {
			sb.WriteString("\n…")
			break
		}
		sb.WriteString(fmt.Sprintf("\n• %s — %s%s", it.Name, money(it.Price), unitSuffix(it.Unit)))
	}
	sb.WriteString("\n\nТеперь в акте можно писать «штукатурка 45 м²» — без цены.")
	b.textKB(ctx, chatID, sb.String(), priceMenu())
}

// beginPriceClear — «🗑 Очистить» с подтверждением.
func (b *Bot) beginPriceClear(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceClear, map[string]string{})
	b.textKB(ctx, chatID, "Удалить весь прайс? Действие необратимо.", tg.Inline(tg.KB{
		{tg.KBButton{Text: "🗑 Да, удалить", CallbackData: "pclyes"},
			tg.KBButton{Text: "⏹ Нет, оставить", CallbackData: "pclno"}},
	}))
}

// --- поиск по прайсу (v0.3.5) ---------------------------------------------------

// searchCatalog — позиции прайса, содержащие запрос (обе стороны, регистр
// неважен). Чистая функция — легко тестировать.
func searchCatalog(items []domain.CatalogItem, query string) []domain.CatalogItem {
	q := parse.NormName(query)
	if len([]rune(q)) < 2 {
		return nil
	}
	var out []domain.CatalogItem
	for _, it := range items {
		if it.Name == q || strings.Contains(it.Name, q) || strings.Contains(q, it.Name) {
			out = append(out, it)
		}
	}
	return out
}

// beginPriceFind — «🔍 Найти».
func (b *Bot) beginPriceFind(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceFind, map[string]string{})
	b.textKB(ctx, chatID, "Что ищем? Напиши слово — покажу совпадения с ценами:\n\nштукатурка\nМожно искать несколько раз подряд, ⏹ Отмена — выход.", CancelMenu())
}

// priceFindFromText — поиск по подстроке (работает и как /price слово).
func (b *Bot) priceFindFromText(ctx context.Context, chatID int64, text string) {
	items, err := b.st.ListCatalog(ctx, chatID)
	if err != nil {
		log.Printf("прайс: чтение чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не смог прочитать прайс 😕")
		return
	}
	if len(items) == 0 {
		b.text(ctx, chatID, "Прайс пуст — искать нечего 🙂 Добавь: ➕ Добавить/обновить")
		return
	}
	found := searchCatalog(items, text)
	if len(found) == 0 {
		b.text(ctx, chatID, fmt.Sprintf("«%s» в прайсе нет 🤔\nПопробуй другое слово или глянь 📄 Список.", parse.NormName(text)))
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 Нашёл %d:\n\n", len(found)))
	for i, it := range found {
		if i >= 30 {
			sb.WriteString(fmt.Sprintf("\n… и ещё %d — уточни запрос", len(found)-30))
			break
		}
		sb.WriteString(fmt.Sprintf("• %s — %s%s\n", it.Name, money(it.Price), unitSuffix(it.Unit)))
	}
	b.textKB(ctx, chatID, sb.String(), priceMenu())
}

// --- удаление одной позиции (v0.3.5) --------------------------------------------

// beginPriceDel — «🗑 Убрать позицию».
func (b *Bot) beginPriceDel(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceDel, map[string]string{})
	b.textKB(ctx, chatID, "Какую позицию убрать из прайса? Напиши название:\n\nштукатурка", CancelMenu())
}

// priceDelFromText — ищем позицию (точно → вхождение) и просим подтверждение.
func (b *Bot) priceDelFromText(ctx context.Context, chatID int64, text string) {
	items, err := b.st.ListCatalog(ctx, chatID)
	if err != nil {
		log.Printf("прайс: чтение чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не смог прочитать прайс 😕")
		return
	}
	if it, ok := MatchCatalog(items, text); ok {
		_ = b.st.SetState(ctx, chatID, stPriceDelConf, map[string]string{"del_name": it.Name})
		b.textKB(ctx, chatID, fmt.Sprintf("Убрать из прайса?\n\n• %s — %s%s\n\nСозданные акты не трогаю — только будущие позиции без цены.",
			it.Name, money(it.Price), unitSuffix(it.Unit)), tg.Inline(tg.KB{
			{tg.KBButton{Text: "🗑 Да, убрать", CallbackData: "pdelyes"},
				tg.KBButton{Text: "⏹ Нет, оставить", CallbackData: "pdelno"}},
		}))
		return
	}
	b.text(ctx, chatID, fmt.Sprintf("В прайсе нет «%s» 🤔\nТочное название можно глянуть: 🔍 Найти или 📄 Список.", parse.NormName(text)))
}

// priceDelConfirm — исполнение удаления по кнопке «Да, убрать».
func (b *Bot) priceDelConfirm(ctx context.Context, chatID int64, name string) {
	if name == "" {
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Не понял, какую позицию 🤔 Начни заново: 💵 Прайс → 🗑 Убрать позицию.", priceMenu())
		return
	}
	it, ok, err := b.st.DeleteCatalogItem(ctx, chatID, name)
	if err != nil {
		log.Printf("прайс: удаление чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не удалось удалить 😕 Попробуй ещё раз.")
		return
	}
	b.reset(ctx, chatID)
	if !ok {
		b.textKB(ctx, chatID, "Такой позиции в прайсе уже нет.", priceMenu())
		return
	}
	n, _ := b.st.CatalogCount(ctx, chatID)
	b.textKB(ctx, chatID, fmt.Sprintf("🗑 Убрал из прайса: %s (было %s%s)\nОсталось позиций: %d",
		it.Name, money(it.Price), unitSuffix(it.Unit), n), priceMenu())
}
