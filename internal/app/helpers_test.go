// Тесты v0.3.5: контроль черновика (undo/показ) и распознавание фраз строителя.
package app

import (
	"testing"

	"proakt/internal/domain"
)

func draftLines() []domain.DraftLine {
	return []domain.DraftLine{
		{Name: "штукатурка", Qty: 45, Unit: "м²", Price: 260, Sum: 11700},
		{Name: "демонтаж стен", Price: 2000, Sum: 2000},
		{Name: "поклейка стеклохолста", Qty: 30, Unit: "м²", Price: 45, Sum: 1350},
	}
}

func TestRemoveLastDraft(t *testing.T) {
	data := map[string]string{"lines": linesJSON(draftLines())}
	rest, removed, ok := removeLastDraft(data)
	if !ok {
		t.Fatal("ok=false, хочу true")
	}
	if removed.Name != "поклейка стеклохолста" || removed.Sum != 1350 {
		t.Errorf("убрана не та позиция: %+v", removed)
	}
	if len(rest) != 2 {
		t.Fatalf("осталось %d, хочу 2", len(rest))
	}
	if sumDraft(rest) != 13700 {
		t.Errorf("сумма после undo = %v, хочу 13700", sumDraft(rest))
	}
}

func TestRemoveLastDraftEmpty(t *testing.T) {
	for _, data := range []map[string]string{nil, {}, {"lines": ""}, {"lines": "[]"}} {
		if _, _, ok := removeLastDraft(data); ok {
			t.Fatalf("пустой черновик %v: ok=true, хочу false", data)
		}
	}
}

func TestUndoDraftRoundtrip(t *testing.T) {
	// undo → сериализация → следующий undo убирает предпоследнюю
	data := map[string]string{"lines": linesJSON(draftLines())}
	rest1, _, ok := removeLastDraft(data)
	if !ok {
		t.Fatal("первый undo не сработал")
	}
	data["lines"] = linesJSON(rest1)
	rest2, removed2, ok := removeLastDraft(data)
	if !ok || removed2.Name != "демонтаж стен" || len(rest2) != 1 {
		t.Fatalf("второй undo: %+v ok=%v", removed2, ok)
	}
}

func TestIsUndoText(t *testing.T) {
	yes := []string{"убрать", "Убрать последнюю", "убрать последнюю позицию", "не так",
		"Ошибся!", "отменить последнюю", "undo", "удали последнюю"}
	for _, s := range yes {
		if !isUndoText(s) {
			t.Errorf("isUndoText(%q) = false, хочу true", s)
		}
	}
	no := []string{"штукатурка 45 м2 260", "завершить", "отмена", "демонтаж 2000", ""}
	for _, s := range no {
		if isUndoText(s) {
			t.Errorf("isUndoText(%q) = true, хочу false", s)
		}
	}
}

func TestIsDraftText(t *testing.T) {
	yes := []string{"черновик", "Покажи черновик", "что в акте", "список позиций", "draft"}
	for _, s := range yes {
		if !isDraftText(s) {
			t.Errorf("isDraftText(%q) = false, хочу true", s)
		}
	}
	no := []string{"завершить", "штукатурка 45", "отмена", ""}
	for _, s := range no {
		if isDraftText(s) {
			t.Errorf("isDraftText(%q) = true, хочу false", s)
		}
	}
}

func TestIsUndoDraftDoNotCollide(t *testing.T) {
	// фразы строителя не должны перехватывать друг у друга позиции/завершение
	for _, s := range []string{"завершить", "готово", "всё"} {
		if isUndoText(s) || isDraftText(s) {
			t.Errorf("%q перехвачен undo/draft, а это завершение", s)
		}
	}
}
