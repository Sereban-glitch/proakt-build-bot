package parse

import "testing"

func TestParsePosition(t *testing.T) {
	cases := []struct {
		in    string
		ok    bool
		name  string
		qty   float64
		unit  string
		price float64
		sum   float64
	}{
		{"штукатурка 45 260", true, "штукатурка", 45, "", 260, 11700},
		{"штукатурка 45 м² 260", true, "штукатурка", 45, "м²", 260, 11700},
		{"демонтаж стен 2000", true, "демонтаж стен", 1, "", 2000, 2000},
		{"стяжка 3,5 400", true, "стяжка", 3.5, "", 400, 1400},
		{"електрика 12 м.п. 85", true, "електрика", 12, "м.п.", 85, 1020},
		{"45 260", false, "", 0, "", 0, 0},                                  // нет названия
		{"штукатурка", false, "", 0, "", 0, 0},                              // нет чисел
		{"", false, "", 0, "", 0, 0},                                        // пусто
		{"демонтаж -5 100", false, "", 0, "", 0, 0},                         // отрицательное количество
		{"малярні роботи 2 2 300", true, "малярні роботи", 2, "", 300, 600}, // qty=2, unit пропущен (число), price=300
	}
	for _, c := range cases {
		got, ok := ParsePosition(c.in)
		if ok != c.ok {
			t.Errorf("%q: ok=%v, хочу %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if got.Name != c.name || got.Qty != c.qty || got.Unit != c.unit || got.Price != c.price || got.Sum != c.sum {
			t.Errorf("%q: got %+v, хочу name=%q qty=%v unit=%q price=%v sum=%v",
				c.in, got, c.name, c.qty, c.unit, c.price, c.sum)
		}
	}
}

func TestParsePositionSumOnly(t *testing.T) {
	got, ok := ParsePosition("грунтовка стін 850,50")
	if !ok {
		t.Fatal("должен разобраться")
	}
	if got.Sum != 850.50 || got.Qty != 1 {
		t.Errorf("got %+v", got)
	}
}
