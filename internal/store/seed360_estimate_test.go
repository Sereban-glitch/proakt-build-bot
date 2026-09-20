package store

import (
	"math"
	"testing"
)

func TestSeed360Estimate(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	chat := uniqueChat()

	obj, err := st.CreateObject(ctx, chat, "Парковый 2 · квартира", "Заказчик · данные скрыты")
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	estID, total, err := st.Seed360Estimate(ctx, chat, obj.ID)
	if err != nil {
		t.Fatalf("Seed360Estimate: %v", err)
	}
	if estID == 0 {
		t.Fatal("estID = 0, хочу > 0")
	}
	// Итог mock.js: 38 позиций, 190383.50 (допуск на копейки).
	if math.Abs(total-190383.50) > 0.01 {
		t.Fatalf("итог %v, хочу 190383.50", total)
	}

	lines, err := st.EstimateLines(ctx, estID)
	if err != nil {
		t.Fatalf("EstimateLines: %v", err)
	}
	if len(lines) != 38 {
		t.Fatalf("позиций %d, хочу 38", len(lines))
	}
	for i, l := range lines {
		want := l.Qty * l.Price
		if math.Abs(l.Sum-want) > 0.01 {
			t.Fatalf("строка %d (%s): sum %v != qty*price %v", i+1, l.Name, l.Sum, want)
		}
	}

	// Повтор — идемпотентно, без дублей.
	estID2, total2, err := st.Seed360Estimate(ctx, chat, obj.ID)
	if err != nil {
		t.Fatalf("Seed360Estimate повтор: %v", err)
	}
	if estID2 != estID {
		t.Fatalf("повтор создал дубль: %d vs %d", estID2, estID)
	}
	if math.Abs(total2-total) > 0.01 {
		t.Fatalf("повтор вернул другой итог: %v vs %v", total2, total)
	}
	ests, err := st.ListEstimates(ctx, chat, obj.ID)
	if err != nil {
		t.Fatalf("ListEstimates: %v", err)
	}
	if len(ests) != 1 {
		t.Fatalf("смет у объекта %d, хочу 1 (без дублей)", len(ests))
	}
}
