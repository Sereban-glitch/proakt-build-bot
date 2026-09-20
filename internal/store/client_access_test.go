package store

import "testing"

func TestClientAccessGrantRevoke(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	o, err := st.CreateObject(ctx, uniqueChat(), "Тест-объект 360", "Тест")
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	const tg int64 = 777001
	if err := st.GrantClient(ctx, o.ID, tg); err != nil {
		t.Fatalf("grant: %v", err)
	}
	ok, err := st.CanClientSee(ctx, tg, o.ID)
	if err != nil {
		t.Fatalf("can: %v", err)
	}
	if !ok {
		t.Fatalf("должен видеть после grant")
	}
	objs, err := st.ClientObjects(ctx, tg)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(objs) != 1 || objs[0].ID != o.ID {
		t.Fatalf("в списке %d объектов, хочу 1 свой", len(objs))
	}
	ids, err := st.ClientList(ctx, o.ID)
	if err != nil {
		t.Fatalf("ClientList: %v", err)
	}
	if len(ids) != 1 || ids[0].UserID != tg || !ids[0].ShowFinance {
		t.Fatalf("привязка %v, хочу [{tg %d, финансы видны}]", ids, tg)
	}
	if err := st.SetClientFinance(ctx, o.ID, tg, false); err != nil {
		t.Fatalf("SetClientFinance: %v", err)
	}
	vis, err := st.ClientFinanceVisible(ctx, tg, o.ID)
	if err != nil {
		t.Fatalf("ClientFinanceVisible: %v", err)
	}
	if vis {
		t.Fatalf("финансы должны быть скрыты")
	}
	if err := st.RevokeClient(ctx, o.ID, tg); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	ok, _ = st.CanClientSee(ctx, tg, o.ID)
	if ok {
		t.Fatalf("не должен видеть после revoke")
	}
}
