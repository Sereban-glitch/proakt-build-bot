package store

import (
	"testing"
)

// Контрольные суммы — из заметок Виталия (seed360_real_gen.go):
// почти все акты бьются в копейку, №2 и №7 — как в источнике.
func TestSeed360ControlSums(t *testing.T) {
	st := openTestStore(t)
	chat := uniqueChat()
	rep, err := st.Seed360(t.Context(), chat)
	if err != nil {
		t.Fatalf("Seed360: %v", err)
	}
	if rep.Acts != 12 {
		t.Fatalf("актов %d, хочу 12", rep.Acts)
	}
	var wantTotal float64
	for actNo := 1; actNo <= 12; actNo++ {
		for _, l := range seedRealLines[actNo] {
			wantTotal += l.Sum
		}
	}
	if rep.Total != wantTotal {
		t.Fatalf("сумма %v, хочу %v", rep.Total, wantTotal)
	}
	cat, err := st.ListCatalog(t.Context(), chat)
	if err != nil {
		t.Fatalf("ListCatalog: %v", err)
	}
	if len(cat) != len(seedCatalog) {
		t.Fatalf("прайс %d позиций, хочу %d", len(cat), len(seedCatalog))
	}
	// Повтор — не дублирует.
	rep2, err := st.Seed360(t.Context(), chat)
	if err != nil {
		t.Fatalf("Seed360 повтор: %v", err)
	}
	if rep2.ObjectID != rep.ObjectID {
		t.Fatalf("повтор создал новый объект: %d vs %d", rep2.ObjectID, rep.ObjectID)
	}
	acts, err := st.ListActs(t.Context(), chat, 50)
	if err != nil {
		t.Fatalf("ListActs: %v", err)
	}
	if len(acts) != 14 {
		t.Fatalf("в базе актов %d, хочу 14 (12 Парковый + 2 фасад)", len(acts))
	}
	// Апгрейд заглушек + демо-оплаты.
	if _, err := st.Seed360Upgrade(t.Context(), chat); err != nil {
		t.Fatalf("Seed360Upgrade: %v", err)
	}
	added, err := st.Seed360DemoPayments(t.Context(), chat)
	if err != nil {
		t.Fatalf("Seed360DemoPayments: %v", err)
	}
	if added != 14 {
		t.Fatalf("демо-оплат %d, хочу 14", added)
	}
	added2, err := st.Seed360DemoPayments(t.Context(), chat)
	if err != nil {
		t.Fatalf("Seed360DemoPayments повтор: %v", err)
	}
	if added2 != 0 {
		t.Fatalf("повтор добавил %d оплат, хочу 0", added2)
	}
}

func TestSeed360RealLinesMatchNotes(t *testing.T) {
	// Суммы из заметок Виталия (контроль парсера/генератора).
	want := map[int]float64{
		1: 38980, 2: 42563, 3: 34296, 4: 40720, 5: 39470, 6: 41024,
		7: 45886, 8: 48340, 9: 48127, 10: 50023, 11: 55651, 12: 64309,
	}
	for actNo, w := range want {
		var got float64
		for _, l := range seedRealLines[actNo] {
			got += l.Sum
		}
		if got != w {
			t.Errorf("акт %d: сумма %v, хочу %v", actNo, got, w)
		}
	}
}
