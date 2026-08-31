package parse

import "testing"

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"грунтовка 45м²", "грунтовка|45|м²"},
		{"грунтовка 45м", "грунтовка|45|м"},
		{"грунтовка 45 м2", "грунтовка|45|м2"},
		{"демонтаж 2000 грн", "демонтаж|2000"},
		{"демонтаж 2000грн", "демонтаж|2000"},
		{"штукатурка 12,5кг", "штукатурка|12,5|кг"},
		{"45метров", "45|метров"},
		{"штукатурка 45 260", "штукатурка|45|260"},
		{"демонтаж 2000", "демонтаж|2000"},
		{"45", "45"},
		{"м²", "м²"},
	}
	for _, c := range cases {
		got := Tokenize(c.in)
		if join(got) != c.want {
			t.Errorf("Tokenize(%q) = %q, want %q", c.in, join(got), c.want)
		}
	}
}

func TestParsePositionGlued(t *testing.T) {
	// «грунтовка стен 45м 260» — слитая единица и цена
	l, ok := ParsePosition("грунтовка стен 45м 260")
	if !ok || l.Name != "грунтовка стен" || l.Qty != 45 || l.Unit != "м" || l.Price != 260 || l.Sum != 11700 {
		t.Errorf("glued unit+price: got %+v ok=%v", l, ok)
	}
	// «демонтаж 2000 грн» — валюта не в наименовании
	l, ok = ParsePosition("демонтаж перегородок 2000 грн")
	if !ok || l.Name != "демонтаж перегородок" || l.Sum != 2000 {
		t.Errorf("currency: got %+v ok=%v", l, ok)
	}
	// единица канонизируется: «м2» → «м²»
	l, ok = ParsePosition("штукатурка 45 м2 260")
	if !ok || l.Unit != "м²" {
		t.Errorf("unit canon: got %+v ok=%v", l, ok)
	}
}

func TestParseQtyGlued(t *testing.T) {
	name, qty, unit, ok := ParseQty("шлифовка стен штукатурки 45м")
	if !ok || name != "шлифовка стен штукатурки" || qty != 45 || unit != "м" {
		t.Errorf("glued qty: got %q %v %q ok=%v", name, qty, unit, ok)
	}
}

func TestSplitPositions(t *testing.T) {
	// две позиции одной строкой (реальный ввод мастера)
	segs := SplitPositions("Шлифовка стен штукатурки перед шпаклевкой 45м грунтовка стен перед шпаклевкой 45м")
	if len(segs) != 2 {
		t.Fatalf("two positions: got %d segs %q", len(segs), segs)
	}
	if _, _, _, ok := ParseQty(segs[0]); !ok {
		t.Errorf("seg1 не разбирается: %q", segs[0])
	}
	if _, _, _, ok := ParseQty(segs[1]); !ok {
		t.Errorf("seg2 не разбирается: %q", segs[1])
	}

	// одна позиция с ценой — НЕ режется
	if segs := SplitPositions("штукатурка 45 м² 260"); segs != nil {
		t.Errorf("qty+unit+price must not split: %q", segs)
	}
	if segs := SplitPositions("штукатурка 45 260"); segs != nil {
		t.Errorf("qty+price must not split: %q", segs)
	}
	if segs := SplitPositions("демонтаж 2000"); segs != nil {
		t.Errorf("sum-only must not split: %q", segs)
	}
	if segs := SplitPositions("штукатурка 45 м²"); segs != nil {
		t.Errorf("qty+unit single must not split: %q", segs)
	}

	// две позиции с ценами
	segs = SplitPositions("шпаклёвка 200 м² 260 шлифовка 45 м² 50")
	if len(segs) != 2 {
		t.Fatalf("two priced positions: got %q", segs)
	}
	if _, ok := ParsePosition(segs[0]); !ok {
		t.Errorf("seg1 не разбирается: %q", segs[0])
	}
	if _, ok := ParsePosition(segs[1]); !ok {
		t.Errorf("seg2 не разбирается: %q", segs[1])
	}

	// «и» не тянется в наименование второй позиции
	segs = SplitPositions("штукатурка 45 м² и грунтовка 30 м²")
	if len(segs) != 2 || segs[1] != "грунтовка 30 м²" {
		t.Errorf("и-связка: got %q", segs)
	}
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += "|"
		}
		out += s
	}
	return out
}
