// Тесты подбора цены из прайса (MatchCatalog, fillFromCatalog) — v0.3.
package app

import (
	"testing"

	"proakt/internal/domain"
)

func catalog() []domain.CatalogItem {
	return []domain.CatalogItem{
		{Name: "штукатурка", Unit: "м²", Price: 260},
		{Name: "штукатурка откосов", Unit: "м²", Price: 300},
		{Name: "демонтаж стен", Price: 2000},
		{Name: "поклейка стеклохолста", Unit: "м²", Price: 45},
	}
}

func TestMatchCatalog(t *testing.T) {
	cases := []struct {
		query string
		want  string
		found bool
	}{
		{"штукатурка", "штукатурка", true},
		{"Штукатурка", "штукатурка", true},          // регистр неважен
		{"штукатурка стен", "штукатурка", true},     // каталог внутри запроса
		{"поклейка", "поклейка стеклохолста", true}, // запрос внутри каталога
		{"демонтаж стен", "демонтаж стен", true},
		{"электрика", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		it, ok := MatchCatalog(catalog(), c.query)
		if ok != c.found || (ok && it.Name != c.want) {
			t.Errorf("MatchCatalog(%q) = (%q,%v), хочу (%q,%v)", c.query, it.Name, ok, c.want, c.found)
		}
	}
}

func TestFillFromCatalog(t *testing.T) {
	lines := []domain.DraftLine{
		{Name: "Штукатурка стен", Qty: 45, Unit: "м²", Price: 260, Sum: 11700}, // с ценой — не трогаем
		{Name: "штукатурка", Qty: 20, Unit: "м²"},                              // без цены
		{Name: "демонтаж стен", Qty: 1, Price: 0, Sum: 2000},                   // сумма есть
		{Name: "электрика", Qty: 3, Unit: "точка"},                             // нет в прайсе
	}
	out, missing := fillFromCatalog(catalog(), lines)
	if len(out) != 3 {
		t.Fatalf("заполненных = %d, хочу 3: %+v", len(out), out)
	}
	if len(missing) != 1 || missing[0] != "электрика" {
		t.Fatalf("missing = %v", missing)
	}
	if out[1].Price != 260 || out[1].Sum != 5200 {
		t.Errorf("подстановка: %+v", out[1])
	}
}

func TestFillFromCatalogEmptyUnit(t *testing.T) {
	lines := []domain.DraftLine{{Name: "поклейка стеклохолста", Qty: 10}}
	out, missing := fillFromCatalog(catalog(), lines)
	if len(missing) != 0 || len(out) != 1 {
		t.Fatalf("out=%v missing=%v", out, missing)
	}
	if out[0].Unit != "м²" || out[0].Sum != 450 {
		t.Errorf("юнит из прайса: %+v", out[0])
	}
}
