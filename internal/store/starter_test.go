package store

import (
	"testing"
)

func TestStarterDeleteOnlyMarked(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	chat := uniqueChat()

	rep, err := st.Seed360(ctx, chat)
	if err != nil {
		t.Fatalf("Seed360: %v", err)
	}
	// Свои данные Виталика: объект + позиция прайса.
	mine, err := st.CreateObject(ctx, chat, "Свой объект", "Свои")
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if _, _, err := st.UpsertCatalogItem(ctx, chat, "своя шпаклёвка", "м²", 999); err != nil {
		t.Fatalf("UpsertCatalogItem: %v", err)
	}
	has, err := st.StarterHasData(ctx, chat)
	if err != nil || !has {
		t.Fatalf("StarterHasData: %v %v", has, err)
	}
	del, err := st.DeleteStarterData(ctx, chat)
	if err != nil {
		t.Fatalf("DeleteStarterData: %v", err)
	}
	if del.Objects != 3 {
		t.Fatalf("объектов удалено %d, хочу 3", del.Objects)
	}
	if del.Prices != len(seedCatalog) {
		t.Fatalf("цен удалено %d, хочу %d", del.Prices, len(seedCatalog))
	}
	// Своё цело.
	if _, err := st.GetObject(ctx, mine.ID); err != nil {
		t.Fatalf("свой объект пропал: %v", err)
	}
	cat, _ := st.ListCatalog(ctx, chat)
	found := false
	for _, c := range cat {
		if c.Name == "своя шпаклёвка" {
			found = true
		}
	}
	if !found {
		t.Fatalf("своя цена пропала")
	}
	// Парковый удалён, сид-объект rep.ObjectID тоже.
	if _, err := st.GetObject(ctx, rep.ObjectID); err == nil {
		t.Fatalf("стартовый объект остался")
	}
	has, _ = st.StarterHasData(ctx, chat)
	if has {
		t.Fatalf("метки остались после очистки")
	}
	// Повтор безопасен.
	del2, err := st.DeleteStarterData(ctx, chat)
	if err != nil {
		t.Fatalf("DeleteStarterData повтор: %v", err)
	}
	if del2.Objects != 0 || del2.Prices != 0 {
		t.Fatalf("повтор удалил %v, хочу нули", del2)
	}
}

func TestDeleteObject(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	chat := uniqueChat()
	o, _ := st.CreateObject(ctx, chat, "На удаление", "")
	name, err := st.DeleteObject(ctx, chat, o.ID)
	if err != nil || name != "На удаление" {
		t.Fatalf("DeleteObject: %v %q", name, err)
	}
	if _, err := st.GetObject(ctx, o.ID); err == nil {
		t.Fatalf("объект остался")
	}
	o2, _ := st.CreateObject(ctx, chat, "Чужой", "")
	if _, err := st.DeleteObject(ctx, chat+1, o2.ID); err == nil {
		t.Fatalf("чужой чат удалил объект")
	}
}
