// Tokenize и SplitPositions (v0.3.6):
//  1. «слитые» единицы — строитель пишет «45м» без пробела → режем на «45» «м»;
//  2. слова-валюта («грн», «₴») выбрасываем — они не часть наименования;
//  3. несколько позиций в одной строке — «шлифовка 45 м грунтовка 45 м»
//     делим на отдельные позиции по границе «единица → обычное слово».
package parse

import (
	"strings"
)

// currencyWords — валюта в строке позиции: в наименование не попадает.
var currencyWords = map[string]bool{
	"грн": true, "грн.": true, "гривна": true, "гривну": true, "гривні": true,
	"гривен": true, "гривны": true, "uah": true, "₴": true,
}

func isCurrency(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	return currencyWords[low] || currencyWords[strings.Trim(low, ".,")]
}

// Tokenize — слова строки с разрезанием слитых «число+единица» и без валюты.
//
//	"грунтовка 45м²"    → ["грунтовка", "45", "м²"]
//	"демонтаж 2000 грн" → ["демонтаж", "2000"]
func Tokenize(s string) []string {
	var out []string
	for _, f := range strings.Fields(strings.ReplaceAll(s, "\u00a0", " ")) {
		if isCurrency(f) {
			continue
		}
		num, rest, ok := splitNumSuffix(f)
		if !ok {
			out = append(out, f)
			continue
		}
		if rest == "" { // «2000грн» → «2000»
			out = append(out, num)
			continue
		}
		if NormalizeUnit(rest) != "" { // «45м» → «45» «м»
			out = append(out, num, rest)
			continue
		}
		out = append(out, f)
	}
	return out
}

// splitNumSuffix — слово вида «число+хвост»: «45м» → ("45","м",true).
// Хвост пустой, если это валюта («2000грн»). ok=false — не число+суффикс.
func splitNumSuffix(f string) (num, rest string, ok bool) {
	r := []rune(f)
	best := -1
	for i := 1; i < len(r); i++ {
		if !isNumber(string(r[:i])) {
			break // префикс перестал быть числом — дальше нет смысла
		}
		tail := string(r[i:])
		if NormalizeUnit(tail) != "" || isCurrency(tail) {
			best = i
		}
	}
	if best < 1 {
		return "", "", false
	}
	tail := string(r[best:])
	if isCurrency(tail) {
		return string(r[:best]), "", true
	}
	return string(r[:best]), tail, true
}

// SplitPositions — делит строку на несколько позиций.
// Граница ставится:
//  1. сразу после ЕДИНИЦЫ, если дальше обычное слово (не число и не единица);
//  2. сразу после ЦЕНЫ (число после количества/единицы), если дальше обычное слово.
//
// Вернёт nil, если строка — одна позиция («штукатурка 45 м² 260» не режется:
// после «м²» идёт цена, после цены — конец).
//
//	"шлифовка стен 45 м грунтовка стен 45 м" → ["шлифовка стен 45 м", "грунтовка стен 45 м"]
//	"штукатурка 45 м² 260"                   → nil
func SplitPositions(s string) []string {
	toks := Tokenize(s)
	if len(toks) < 4 {
		return nil
	}
	var segs []string
	var cur []string
	for i, t := range toks {
		cur = append(cur, t)
		if i == len(toks)-1 {
			break
		}
		next := toks[i+1]
		if isNumber(next) || NormalizeUnit(next) != "" {
			continue // дальше ещё часть этой же позиции (цена/единица)
		}
		boundary := NormalizeUnit(t) != ""      // после единицы → новая позиция
		if !boundary && isNumber(t) && i > 0 && // после цены (число за количеством/единицей)
			(isNumber(toks[i-1]) || NormalizeUnit(toks[i-1]) != "") {
			boundary = true
		}
		if boundary {
			segs = append(segs, strings.Join(cur, " "))
			cur = nil
		}
	}
	if len(cur) > 0 {
		segs = append(segs, strings.Join(cur, " "))
	}
	if len(segs) < 2 {
		return nil
	}
	// хвостовые связки не тянут в наименование: «… 45 м и грунтовка …»
	for i := range segs {
		seg := strings.TrimSpace(segs[i])
		for _, lead := range []string{"и ", "і ", "+ ", "плюс "} {
			seg = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(seg), lead))
		}
		segs[i] = seg
	}
	return segs
}

// SuspiciousMulti — похоже на несколько позиций, которые мы НЕ смогли разрезать:
// чисел ≥ 2 и между крайними есть обычное слово (не единица). Разбирать такую
// строку как одну позицию опасно (числа попадут в количество/цены наугад) —
// лучше переспросить, чем молча записать неверные деньги.
func SuspiciousMulti(s string) bool {
	toks := Tokenize(s)
	first, last := -1, -1
	for i, t := range toks {
		if isNumber(t) {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 || last == first {
		return false
	}
	for i := first + 1; i < last; i++ {
		if !isNumber(toks[i]) && NormalizeUnit(toks[i]) == "" {
			return true
		}
	}
	return false
}
