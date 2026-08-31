// Тесты единиц, ParseQty и ParseCatalogItem (v0.3).
package parse

import "testing"

func TestNormalizeUnit(t *testing.T) {
	cases := map[string]string{
		"м2": "м²", "м²": "м²", "кв.м": "м²", "КВ": "м²", "квадратов": "м²",
		"м.п.": "м.п.", "мп": "м.п.", "м/п": "м.п.", "пог.м": "м.п.",
		"шт": "шт", "Шт.": "шт", "кг": "кг", "л": "л", "м3": "м³", "уп": "уп",
		"м": "м", "метров": "м",
		"стены": "", "штукатурка": "", "": "", "оченьдлинноеслово": "",
	}
	for in, want := range cases {
		if got := NormalizeUnit(in); got != want {
			t.Errorf("NormalizeUnit(%q) = %q, хочу %q", in, got, want)
		}
	}
}

func TestNormName(t *testing.T) {
	if NormName("  Штукатурка   стен ё ") != "штукатурка стен е" {
		t.Fatalf("NormName: %q", NormName("  Штукатурка   стен ё "))
	}
}

func TestParseQty(t *testing.T) {
	cases := []struct {
		in   string
		name string
		qty  float64
		unit string
		ok   bool
	}{
		{"штукатурка 45 м²", "штукатурка", 45, "м²", true},
		{"штукатурка 45 м2", "штукатурка", 45, "м²", true},
		{"Шпаклёвка 30,5 кв.м", "Шпаклёвка", 30.5, "м²", true},
		{"плинтус 20 м.п.", "плинтус", 20, "м.п.", true},
		{"штукатурка м² 45", "штукатурка", 45, "м²", true},
		{"грунтовка 3 шт", "грунтовка", 3, "шт", true},
		// не количество:
		{"демонтаж 2000", "", 0, "", false},     // одно число без единицы — сумма
		{"штукатурка 45 260", "", 0, "", false}, // два числа — полная позиция
		{"штукатурка", "", 0, "", false},        // без чисел
		{"45", "", 0, "", false},                // число без наименования
		{"стены мешок 5", "", 0, "", false},     // единицы нет
	}
	for _, c := range cases {
		name, qty, unit, ok := ParseQty(c.in)
		if ok != c.ok || (ok && (name != c.name || qty != c.qty || unit != c.unit)) {
			t.Errorf("ParseQty(%q) = (%q,%v,%q,%v), хочу (%q,%v,%q,%v)",
				c.in, name, qty, unit, ok, c.name, c.qty, c.unit, c.ok)
		}
	}
}

func TestParseCatalogItem(t *testing.T) {
	cases := []struct {
		in    string
		name  string
		unit  string
		price float64
		ok    bool
	}{
		{"штукатурка 260", "штукатурка", "", 260, true},
		{"Штукатурка стен 260", "штукатурка стен", "", 260, true},
		{"штукатурка м² 260", "штукатурка", "м²", 260, true},
		{"штукатурка м2 260", "штукатурка", "м²", 260, true},
		{"демонтаж стен 2000", "демонтаж стен", "", 2000, true},
		{"плинтус м.п. 90,5", "плинтус", "м.п.", 90.5, true},
		// ошибки:
		{"штукатурка", "", "", 0, false},    // без цены
		{"260", "", "", 0, false},           // без наименования
		{"штукатурка -5", "", "", 0, false}, // отрицательная цена
		{"", "", "", 0, false},
	}
	for _, c := range cases {
		it, ok := ParseCatalogItem(c.in)
		if ok != c.ok || (ok && (it.Name != c.name || it.Unit != c.unit || it.Price != c.price)) {
			t.Errorf("ParseCatalogItem(%q) = (%q,%q,%v,%v), хочу (%q,%q,%v,%v)",
				c.in, it.Name, it.Unit, it.Price, ok, c.name, c.unit, c.price, c.ok)
		}
	}
}
