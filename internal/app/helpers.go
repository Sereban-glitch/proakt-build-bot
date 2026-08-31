package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"proakt/internal/domain"
)

// --- черновик акта в state_data ------------------------------------------------

func parseDraft(js string) []domain.DraftLine {
	if js == "" {
		return nil
	}
	var lines []domain.DraftLine
	_ = json.Unmarshal([]byte(js), &lines)
	return lines
}

func linesJSON(lines []domain.DraftLine) string {
	raw, _ := json.Marshal(lines)
	return string(raw)
}

func appendDraft(data map[string]string, line domain.DraftLine) []domain.DraftLine {
	lines := parseDraft(data["lines"])
	lines = append(lines, line)
	return lines
}

// removeLastDraft — убрать последнюю позицию черновика (v0.3.5, «линза строителя»:
// ошибся количеством на объекте — не начинай акт заново, просто убери строку).
// Возвращает оставшиеся позиции и убранную (ok=false — черновик пуст).
func removeLastDraft(data map[string]string) (rest []domain.DraftLine, removed domain.DraftLine, ok bool) {
	lines := parseDraft(data["lines"])
	if len(lines) == 0 {
		return nil, domain.DraftLine{}, false
	}
	last := lines[len(lines)-1]
	return lines[:len(lines)-1], last, true
}

func sumDraft(lines []domain.DraftLine) float64 {
	var sum float64
	for _, l := range lines {
		sum += l.Sum
	}
	return sum
}

func draftToLines(lines []domain.DraftLine) []domain.ActLine {
	out := make([]domain.ActLine, 0, len(lines))
	for i, l := range lines {
		out = append(out, domain.ActLine{Pos: i + 1, Name: l.Name, Qty: l.Qty, Unit: l.Unit, Price: l.Price, Sum: l.Sum})
	}
	return out
}

// --- форматирование -------------------------------------------------------------

// describeLine — человекочитаемое описание позиции.
func describeLine(l domain.DraftLine) string {
	if l.Qty != 1 {
		if l.Unit != "" {
			return fmt.Sprintf("%s — %s %s × %s = %s", l.Name, num(l.Qty), l.Unit, money(l.Price), money(l.Sum))
		}
		return fmt.Sprintf("%s — %s × %s = %s", l.Name, num(l.Qty), money(l.Price), money(l.Sum))
	}
	return fmt.Sprintf("%s — %s", l.Name, money(l.Sum))
}

// money — «12 300 грн» с пробелами-тысячами.
func money(v float64) string {
	neg := v < 0
	iv := int64(v*100 + 0.5)
	if v < 0 {
		iv = int64(v*100 - 0.5)
	}
	whole := iv / 100
	frac := iv % 100
	s := strconv.FormatInt(whole, 10)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, " ")
	if frac != 0 {
		out += "," + fmt.Sprintf("%02d", abs64(frac))
	}
	if neg {
		out = "-" + out
	}
	return out + " грн"
}

func abs64(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}

func num(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// isFinishText — «завершить акт», набранный обычным текстом (без кнопки):
// «завершить», «готово», «всё», «конец», «хватит» и т.п.
func isFinishText(s string) bool {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " ")))
	s = strings.Trim(s, " !.,…✅✔️")
	switch s {
	case "завершить", "завершить акт", "завершать", "закончить", "закончил", "заканчивай",
		"готово", "все", "всё", "конец", "хватит", "больше нет", "fin", "finish", "done", "ок", "ok":
		return true
	}
	return false
}

// isCancelText — «отмена», набранная обычным текстом.
func isCancelText(s string) bool {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " ")))
	s = strings.Trim(s, " !.,…⏹")
	switch s {
	case "отмена", "отменить", "отмени", "стоп", "cancel":
		return true
	}
	return false
}

// isUndoText — «убрать последнюю», набранная обычным текстом (v0.3.5).
func isUndoText(s string) bool {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " ")))
	s = strings.Trim(s, " !.,…↩️")
	switch s {
	case "убрать", "убрать последнюю", "убрать последнюю позицию", "убрать позицию",
		"отменить последнюю", "удалить последнюю", "удали последнюю",
		"не так", "ошибся", "ошибка", "undo":
		return true
	}
	return false
}

// isDraftText — «покажи черновик», набранный обычным текстом (v0.3.5).
func isDraftText(s string) bool {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " ")))
	s = strings.Trim(s, " !.,…👀")
	switch s {
	case "черновик", "покажи черновик", "покажи акт", "что в акте", "что уже есть",
		"список позиций", "показать позиции", "draft":
		return true
	}
	return false
}

// parseAmount — сумма из текста («5000», «5000,50», «заплатил 5 тыс» → 5000).
func parseAmount(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
	fields := strings.Fields(s)
	var last float64
	found := false
	for _, f := range fields {
		clean := strings.ReplaceAll(strings.ReplaceAll(f, ",", "."), "грн", "")
		clean = strings.Trim(clean, "₴$;:,.!?")
		if v, err := strconv.ParseFloat(clean, 64); err == nil && v > 0 {
			last = v
			found = true
		}
	}
	if !found {
		return 0
	}
	return last
}
