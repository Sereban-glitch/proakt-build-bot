// Package parse — разбор строки позиции акта из текста.
// Поддерживает форматы:
//
//	"штукатурка 45 260"      → наименование, количество, цена за единицу
//	"штукатурка 45 м² 260"   → то же + единица после количества
//	"демонтаж стен 2000"     → наименование + готовая сумма
package parse

import (
	"strconv"
	"strings"

	"proakt/internal/domain"
)

// isNumber — похоже ли поле на число (запятая — десятичный разделитель; допускаем знак).
func isNumber(s string) bool {
	s = strings.ReplaceAll(s, ",", ".")
	if s != "" && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	dots, digits := 0, 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '.':
			dots++
			if dots > 1 {
				return false
			}
		default:
			return false
		}
	}
	return digits > 0
}

func toFloat(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return v, err == nil
}

// ParsePosition разбирает строку в позицию черновика.
func ParsePosition(s string) (domain.DraftLine, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return domain.DraftLine{}, false
	}

	var numIdx []int
	for i, f := range fields {
		if isNumber(f) {
			numIdx = append(numIdx, i)
		}
	}

	var line domain.DraftLine
	switch {
	case len(numIdx) >= 2:
		qtyIdx, priceIdx := numIdx[0], numIdx[len(numIdx)-1]
		qty, ok1 := toFloat(fields[qtyIdx])
		price, ok2 := toFloat(fields[priceIdx])
		if !ok1 || !ok2 || qty <= 0 || price <= 0 {
			return domain.DraftLine{}, false
		}
		name := strings.Join(fields[:qtyIdx], " ")
		if name == "" {
			return domain.DraftLine{}, false
		}
		// единица — поле сразу после количества (короткое и не число)
		unit := ""
		if qtyIdx+1 < priceIdx {
			cand := fields[qtyIdx+1]
			if !isNumber(cand) && len([]rune(cand)) <= 8 {
				unit = cand
			}
		}
		line = domain.DraftLine{Name: name, Qty: qty, Unit: unit, Price: price, Sum: qty * price}

	case len(numIdx) == 1:
		idx := numIdx[0]
		sum, ok := toFloat(fields[idx])
		if !ok || sum <= 0 {
			return domain.DraftLine{}, false
		}
		var parts []string
		for i, f := range fields {
			if i != idx {
				parts = append(parts, f)
			}
		}
		name := strings.Join(parts, " ")
		if name == "" {
			return domain.DraftLine{}, false
		}
		line = domain.DraftLine{Name: name, Qty: 1, Price: sum, Sum: sum}

	default:
		return domain.DraftLine{}, false
	}

	line.Name = strings.Trim(line.Name, " ,.-")
	if line.Name == "" {
		return domain.DraftLine{}, false
	}
	return line, true
}
