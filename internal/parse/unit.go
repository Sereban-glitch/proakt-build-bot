// Единицы измерения и разбор позиций прайса (v0.3).
package parse

import (
	"strings"

	"proakt/internal/domain"
)

// unitAliases — синонимы единиц к каноническому виду.
var unitAliases = map[string]string{
	"м2": "м²", "м²": "м²", "кв.м": "м²", "кв.м.": "м²", "кв": "м²", "кв.": "м²",
	"квадрат": "м²", "квадрата": "м²", "квадратов": "м²", "квм": "м²",
	"м.п": "м.п.", "м.п.": "м.п.", "мп": "м.п.", "мп.": "м.п.", "м/п": "м.п.",
	"пог.м": "м.п.", "пог.м.": "м.п.", "погонный": "м.п.", "погонные": "м.п.",
	"шт": "шт", "шт.": "шт", "штука": "шт", "штуки": "шт", "штук": "шт",
	"кг": "кг", "кг.": "кг", "л": "л", "литр": "л", "литров": "л",
	"м3": "м³", "м³": "м³", "куб": "м³", "куба": "м³", "кубов": "м³",
	"уп": "уп", "уп.": "уп", "упак": "уп", "упаковка": "уп",
	"м": "м", "м.": "м", "метр": "м", "метра": "м", "метров": "м",
}

// NormalizeUnit приводит написание единицы к канону; "" — не похоже на единицу.
func NormalizeUnit(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.Trim(strings.TrimSpace(s), ".,;:")))
	if len([]rune(s)) > 12 {
		return ""
	}
	return unitAliases[s]
}

// NormName — нормализация наименования прайса: нижний регистр, ё→е, без лишних пробелов.
func NormName(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " ")))
	s = strings.ReplaceAll(s, "ё", "е")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

// ParseQty разбирает «наименование количество единица» БЕЗ цены —
// цену возьмём из прайса. Единица обязательна (отличает количество от суммы):
//
//	"штукатурка 45 м²" → ("штукатурка", 45, "м²")
//	"штукатурка м² 45" → то же (единица до числа тоже понимаем)
//	"демонтаж 2000"    → false (это сумма, не количество)
func ParseQty(s string) (name string, qty float64, unit string, ok bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
	fields := strings.Fields(s)
	var numIdx []int
	for i, f := range fields {
		if isNumber(f) {
			numIdx = append(numIdx, i)
		}
	}
	if len(numIdx) != 1 {
		return "", 0, "", false
	}
	idx := numIdx[0]
	v, valid := toFloat(fields[idx])
	if !valid || v <= 0 {
		return "", 0, "", false
	}
	if idx == 0 {
		return "", 0, "", false // числа перед наименованием не бывает
	}

	// единица после числа: «штукатурка 45 м²»
	if idx+1 < len(fields) && !isNumber(fields[idx+1]) {
		if u := NormalizeUnit(fields[idx+1]); u != "" {
			name := strings.Join(fields[:idx], " ")
			name = strings.Trim(name, " ,.-")
			if len([]rune(name)) >= 2 {
				return name, v, u, true
			}
		}
	}
	// единица перед числом: «штукатурка м² 45»
	if idx >= 2 && !isNumber(fields[idx-1]) {
		if u := NormalizeUnit(fields[idx-1]); u != "" {
			name := strings.Join(fields[:idx-1], " ")
			name = strings.Trim(name, " ,.-")
			if len([]rune(name)) >= 2 {
				return name, v, u, true
			}
		}
	}
	return "", 0, "", false
}

// ParseCatalogItem разбирает строку добавления в прайс: «наименование [ед.] цена».
//
//	"штукатурка 260"        → (штукатурка, "", 260)
//	"штукатурка м² 260"     → (штукатурка, м², 260)
//	"демонтаж стен 2000"    → (демонтаж стен, "", 2000)
func ParseCatalogItem(s string) (domain.CatalogItem, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return domain.CatalogItem{}, false
	}
	// цена — последнее число в строке
	priceIdx := -1
	var price float64
	for i := len(fields) - 1; i >= 0; i-- {
		if isNumber(fields[i]) {
			v, ok := toFloat(fields[i])
			if !ok || v <= 0 {
				return domain.CatalogItem{}, false
			}
			price, priceIdx = v, i
			break
		}
	}
	if priceIdx <= 0 {
		return domain.CatalogItem{}, false
	}
	// единица — короткий не-числовой токен прямо перед ценой
	unit := ""
	nameEnd := priceIdx
	if priceIdx >= 2 && !isNumber(fields[priceIdx-1]) && len([]rune(fields[priceIdx-1])) <= 8 {
		if u := NormalizeUnit(fields[priceIdx-1]); u != "" {
			unit = u
			nameEnd = priceIdx - 1
		}
	}
	name := strings.Trim(strings.Join(fields[:nameEnd], " "), " ,.-")
	if len([]rune(name)) < 2 || len([]rune(name)) > 120 {
		return domain.CatalogItem{}, false
	}
	return domain.CatalogItem{Name: NormName(name), Unit: unit, Price: price}, true
}
