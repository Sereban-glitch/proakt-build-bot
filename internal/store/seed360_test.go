package store

import (
	"testing"

	"proakt/internal/domain"
)

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
	if rep.Total != 549526 {
		t.Fatalf("сумма %v, хочу 549526", rep.Total)
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
	if len(acts) != 12 {
		t.Fatalf("в базе актов %d, хочу 12", len(acts))
	}
	var total float64
	for _, a := range acts {
		total += a.Total
	}
	if total != 549526 {
		t.Fatalf("сумма в базе %v, хочу 549526", total)
	}
	// Акт №12 — детальные строки, сумма 64309.
	lines, err := st.ActLines(t.Context(), findAct(t, acts, 12))
	if err != nil {
		t.Fatalf("ActLines 12: %v", err)
	}
	var s12 float64
	for _, l := range lines {
		s12 += l.Sum
	}
	if s12 != 64309 {
		t.Fatalf("акт 12 сумма %v, хочу 64309", s12)
	}
}

func findAct(t *testing.T, acts []domain.ActBrief, no int) int64 {
	t.Helper()
	for _, a := range acts {
		if a.ActNo == no {
			return a.ID
		}
	}
	t.Fatalf("акт №%d не найден", no)
	return 0
}
