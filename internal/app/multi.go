// Несколько позиций в одной строке (v0.3.6):
// «шлифовка стен 45 м грунтовка стен 45 м» — реальный ввод мастера.
package app

import (
	"context"
	"fmt"
	"strings"

	"proakt/internal/domain"
	"proakt/internal/parse"
)

// parseMultiPosition — разбор строки с несколькими позициями.
// Возвращает (позиции, true) если строка распознана как несколько И все части
// разобрались; (nil, true) — несколько, но не разобрались; (nil, false) —
// строка не выглядит как несколько позиций.
// Деньги не угадываем: каждая часть должна разбираться целиком.
func (b *Bot) parseMultiPosition(ctx context.Context, chatID int64, text string) ([]domain.DraftLine, int, bool) {
	segs := parse.SplitPositions(text)
	if len(segs) < 2 {
		return nil, 0, false
	}
	var lines []domain.DraftLine
	fromCatalog := 0
	for _, seg := range segs {
		if name, qty, unit, ok := parse.ParseQty(seg); ok {
			line, ok2 := b.lineFromCatalog(ctx, chatID, name, qty, unit)
			if !ok2 {
				return nil, 0, true // без цены, а в прайсе нет — не молчим
			}
			lines = append(lines, line)
			fromCatalog++
			continue
		}
		line, ok := parse.ParsePosition(seg)
		if !ok {
			return nil, 0, true
		}
		lines = append(lines, line)
	}
	return lines, fromCatalog, true
}

const multiFailText = "Похоже, тут несколько позиций, но разобрать не смог 🤔\n" +
	"Введи по одной в строке:\n" +
	"шлифовка стен 45 м²\n" +
	"грунтовка стен 45 м²\n" +
	"(или сразу с ценой: шлифовка 45 м² 50)"

// suspiciousText — «двусмысленная» строка: чисел ≥ 2, между ними обычные слова.
// Разбирать её как ОДНУ позицию опасно (количество/цена наугад) — переспрашиваем.
const suspiciousText = "Кажется, в одной строке несколько позиций 🤔\n" +
	"Введи каждую отдельно — одну строку = одна позиция:\n" +
	"шлифовка стен 45 м² 50\n" +
	"грунтовка стен 45 м² 25"

// addActLines — добавить сразу несколько позиций и отчитаться списком.
func (b *Bot) addActLines(ctx context.Context, chatID int64, data map[string]string, newLines []domain.DraftLine, fromCatalog int) {
	lines := append(parseDraft(data["lines"]), newLines...)
	if len(lines) > 200 {
		b.text(ctx, chatID, "Лимит 200 позиций в акте — завершаем 🙃")
		return
	}
	data["lines"] = linesJSON(lines)
	_ = b.st.SetState(ctx, chatID, stActLines, data)
	_ = b.tg.SendChatAction(ctx, chatID)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Добавил позиций: %d\n", len(newLines)))
	for _, l := range newLines {
		sb.WriteString("• " + describeLine(l) + "\n")
	}
	sb.WriteString(fmt.Sprintf("—\nВсего в акте: %d · сумма: %s", len(lines), money(sumDraft(lines))))
	if fromCatalog > 0 {
		sb.WriteString("\n💡 цена из прайса")
	}
	b.text(ctx, chatID, sb.String())
}
