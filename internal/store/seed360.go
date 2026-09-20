// Сид «ПрорАКТ 360» — настоящие данные Виталия (объект Парковый 2).
//
// Источник строк: заметки Виталия (Google Drive), прайс — смета Парковый 2.
// Повторный запуск не дублирует: объект ищем по имени, акты — по
// (object_id, act_no), прайс — через BulkUpsert, оплаты — только если их нет.
package store

import (
	"context"

	"proakt/internal/domain"
)

// Seed360Report — что залили.
type Seed360Report struct {
	ObjectID   int64
	Acts       int
	Total      float64
	EstimateID int64
	EstTotal   float64
}

// toDraft — строки заметок в черновик акта.
func toDraft(src []domain.DraftLine) []domain.DraftLine { return src }

// realLines — настоящие строки акта №actNo (seed360_real_gen.go).
func realLines(actNo int) []domain.DraftLine {
	var out []domain.DraftLine
	for _, l := range seedRealLines[actNo] {
		out = append(out, domain.DraftLine{Name: l.Name, Qty: l.Qty, Unit: l.Unit, Price: l.Price, Sum: l.Sum})
	}
	return out
}

// Seed360 заливает стартовые объекты Виталия под chatID мастера:
// Парковый 2 (12 актов + смета + прайс), замеры, фасад (2 акта).
func (s *Store) Seed360(ctx context.Context, chatID int64) (Seed360Report, error) {
	var rep Seed360Report

	parkID, err := s.ensureSeedObject(ctx, chatID, "Парковый 2 · квартира", "Заказчик · данные скрыты")
	if err != nil {
		return rep, err
	}
	rep.ObjectID = parkID

	for actNo := 1; actNo <= 12; actNo++ {
		lines := toDraft(realLines(actNo))
		if len(lines) == 0 {
			continue
		}
		if _, err := s.createActFixedNo(ctx, parkID, actNo, lines); err != nil {
			return rep, err
		}
		for _, l := range lines {
			rep.Total += l.Sum
		}
		rep.Acts++
	}

	if _, err := s.BulkUpsertCatalog(ctx, chatID, seedCatalog); err != nil {
		return rep, err
	}
	var catNames []string
	for _, c := range seedCatalog {
		catNames = append(catNames, c.Name)
	}
	if err := s.MarkStarterCatalog(ctx, chatID, catNames); err != nil {
		return rep, err
	}

	if _, err := s.ensureSeedObject(ctx, chatID, "Квартира · новый расчёт", "Заказчик (демо)"); err != nil {
		return rep, err
	}
	fasadID, err := s.ensureSeedObject(ctx, chatID, "Частный дом · фасад", "Заказчик (демо)")
	if err != nil {
		return rep, err
	}
	fasadActs := map[int][]domain.DraftLine{
		1: {{Name: "Подготовка фасада", Qty: 140, Unit: "м²", Price: 300, Sum: 42000}},
		2: {{Name: "Покраска фасада", Qty: 150, Unit: "м²", Price: 350, Sum: 52500}},
	}
	for actNo, lines := range fasadActs {
		if _, err := s.createActFixedNo(ctx, fasadID, actNo, lines); err != nil {
			return rep, err
		}
	}
	fasadPhotos := []struct {
		actNo   int
		caption string
	}{
		{1, "Фасад, подготовка · демо"},
		{2, "Фасад, готово · демо"},
	}
	for _, ph := range fasadPhotos {
		var actID int64
		err := s.pool.QueryRow(ctx, `SELECT id FROM acts WHERE object_id=$1 AND act_no=$2`, fasadID, ph.actNo).Scan(&actID)
		if err != nil {
			return rep, err
		}
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO photos(act_id, object_id, file_id, file_path, caption)
			 SELECT $1, $2, '', '', $3 WHERE NOT EXISTS
			 (SELECT 1 FROM photos WHERE object_id=$2 AND caption=$3)`,
			actID, fasadID, ph.caption); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

// ensureSeedObject — объект по имени + метка стартового.
func (s *Store) ensureSeedObject(ctx context.Context, chatID int64, name, customer string) (int64, error) {
	objs, err := s.ListObjects(ctx, chatID)
	if err != nil {
		return 0, err
	}
	for _, o := range objs {
		if o.Name == name {
			if err := s.MarkStarterObject(ctx, chatID, o.ID); err != nil {
				return 0, err
			}
			return o.ID, nil
		}
	}
	o, err := s.CreateObject(ctx, chatID, name, customer)
	if err != nil {
		return 0, err
	}
	if err := s.MarkStarterObject(ctx, chatID, o.ID); err != nil {
		return 0, err
	}
	return o.ID, nil
}

// Seed360Upgrade меняет строки-заглушки («Работы по акту · архив Виталия»)
// на настоящие из заметок. Трогает только акты-заглушки, живые данные целы.
func (s *Store) Seed360Upgrade(ctx context.Context, chatID int64) (int, error) {
	acts, err := s.ListActs(ctx, chatID, 100)
	if err != nil {
		return 0, err
	}
	fixed := 0
	for _, a := range acts {
		lines, err := s.ActLines(ctx, a.ID)
		if err != nil {
			return fixed, err
		}
		if len(lines) != 1 || lines[0].Name != "Работы по акту · архив Виталия" {
			continue
		}
		real := realLines(a.ActNo)
		if len(real) == 0 {
			continue
		}
		if err := s.ReplaceActLines(ctx, a.ID, real); err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}

// Seed360DemoPayments — демо-оплаты для презентации прогресса (помечены,
// удаляются через интерфейс). Акты 1–11 — полностью, №12 — частично 19783.
// Акты, где оплаты уже есть, не трогаем.
func (s *Store) Seed360DemoPayments(ctx context.Context, chatID int64) (int, error) {
	acts, err := s.ListActs(ctx, chatID, 100)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, a := range acts {
		if a.ObjectID == 0 {
			continue
		}
		got, err := s.Payments(ctx, a.ID)
		if err != nil {
			return added, err
		}
		if len(got) > 0 {
			continue
		}
		amount := a.Total
		if a.ActNo == 12 {
			amount = 19783
		}
		if amount <= 0 {
			continue
		}
		if err := s.CreatePayment(ctx, a.ID, amount, "демо для презентации"); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// ReplaceActLines — заменить все строки акта (одна транзакция).
func (s *Store) ReplaceActLines(ctx context.Context, actID int64, lines []domain.DraftLine) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM act_lines WHERE act_id=$1`, actID); err != nil {
		return err
	}
	for i, l := range lines {
		if _, err := tx.Exec(ctx,
			`INSERT INTO act_lines(act_id, pos, name, qty, unit, price, sum) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			actID, i+1, l.Name, l.Qty, l.Unit, l.Price, l.Sum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// createActFixedNo — вставка акта с фиксированным номером, идемпотентно.
func (s *Store) createActFixedNo(ctx context.Context, objectID int64, actNo int, lines []domain.DraftLine) (int64, error) {
	var actID int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO acts(object_id, act_no) VALUES($1,$2)
		 ON CONFLICT (object_id, act_no) DO NOTHING
		 RETURNING id`, objectID, actNo).Scan(&actID)
	if err != nil {
		// Конфликт — акт уже есть, забираем его id без вставки строк.
		err2 := s.pool.QueryRow(ctx,
			`SELECT id FROM acts WHERE object_id=$1 AND act_no=$2`, objectID, actNo).Scan(&actID)
		if err2 != nil {
			return 0, err2
		}
		return actID, nil
	}
	for i, l := range lines {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO act_lines(act_id, pos, name, qty, unit, price, sum) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			actID, i+1, l.Name, l.Qty, l.Unit, l.Price, l.Sum); err != nil {
			return actID, err
		}
	}
	return actID, nil
}
