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
	b.textKB(ctx, chatID, fmt.Sprintf("💵 Прайс — твои цены.\nПозиций: %d\n\nЗаполни его один раз — и дальше пиши позицию без цены:\n«штукатурка 45 м²» — цену возьму отсюда.", n), priceMenu())
}

func priceMenu() tg.InlineKeyboardMarkup {
	return tg.Inline(tg.KB{
		{tg.KBButton{Text: "📄 Список", CallbackData: "plist"}},
		{tg.KBButton{Text: "➕ Добавить цену", CallbackData: "padd"}},
		{tg.KBButton{Text: "📥 Импорт из файла", CallbackData: "pimport"}},
		{tg.KBButton{Text: "🗑 Очистить", CallbackData: "pclear"}},
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
			sb.WriteString(fmt.Sprintf("\n… и ещё %d позиций", len(items)-100))
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

// beginPriceAdd — «➕ Добавить цену».
func (b *Bot) beginPriceAdd(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceAdd, map[string]string{})
	b.textKB(ctx, chatID, "Напиши позицию с ценой:\n\nштукатурка 260\nштукатурка м² 260\nдемонтаж стен 2000", CancelMenu())
}

// priceAddFromText — разбор строки добавления.
func (b *Bot) priceAddFromText(ctx context.Context, chatID int64, text string) {
	item, ok := parse.ParseCatalogItem(text)
	if !ok {
		b.text(ctx, chatID, "Не разобрал 🤔 Примеры:\nштукатурка 260\nштукатурка м² 260")
		return
	}
	if err := b.st.UpsertCatalogItem(ctx, chatID, item.Name, item.Unit, item.Price); err != nil {
		log.Printf("прайс: upsert чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не записал в прайс 😕 Попробуй ещё раз.")
		return
	}
	n, _ := b.st.CatalogCount(ctx, chatID)
	b.textKB(ctx, chatID, fmt.Sprintf("✅ Прайс: %s — %s%s\nПозиций: %d", item.Name, money(item.Price), unitSuffix(item.Unit), n), priceMenu())
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
	sb.WriteString(fmt.Sprintf("✅ Импортировал позиций: %d (всего в прайсе: %d)\n\nПервые:", n, n))
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
