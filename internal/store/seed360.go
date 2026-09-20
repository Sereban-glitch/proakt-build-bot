// Сид «ПрорАКТ 360» — калиброванные данные Виталия (объект Парковый 2).
//
// Источник: internal/webapp/static/js/services/mock.js (12 актов, смета).
// Повторный запуск не дублирует: объект ищем по имени, акты — по
// (object_id, act_no) через ON CONFLICT DO NOTHING, прайс — через Upsert.
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

// seedActSums — 12 сумм из mock.js, порядок = номера актов 1..12.
var seedActSums = []float64{38980, 42560, 34296, 40720, 39470, 41024, 46026, 48340, 48127, 50023, 55651, 64309}

// seedAct12Lines — детальные строки акта №12 из mock.js linesByAct[112], 1-в-1.
var seedAct12Lines = []domain.DraftLine{
	{Name: "занос материалов", Qty: 23, Unit: "шт", Price: 25, Sum: 575},
	{Name: "укрывка плёнкой перед малярными работами", Qty: 140, Unit: "м²", Price: 30, Sum: 4200},
	{Name: "защита примыканий плёнкой и лентой", Qty: 51, Unit: "м.п", Price: 30, Sum: 1530},
	{Name: "покраска стен валиком", Qty: 104.9, Unit: "м²", Price: 120, Sum: 12588},
	{Name: "покраска откосов и участков стен", Qty: 115.8, Unit: "м.п", Price: 120, Sum: 13896},
	{Name: "покраска потолка безвоздушным методом", Qty: 90.4, Unit: "м²", Price: 160, Sum: 14464},
	{Name: "покраска откосов потолка безвоздушно", Qty: 106.6, Unit: "м.п", Price: 160, Sum: 17056},
}

// Seed360 заливает калиброванный объект Виталия под chatID мастера.
func (s *Store) Seed360(ctx context.Context, chatID int64) (Seed360Report, error) {
	var rep Seed360Report

	objs, err := s.ListObjects(ctx, chatID)
	if err != nil {
		return rep, err
	}
	var objID int64
	for _, o := range objs {
		if o.Name == "Парковый 2 · квартира" {
			objID = o.ID
		}
	}
	if objID == 0 {
		o, err := s.CreateObject(ctx, chatID, "Парковый 2 · квартира", "Заказчик · данные скрыты")
		if err != nil {
			return rep, err
		}
		objID = o.ID
	}
	rep.ObjectID = objID

	for i, sum := range seedActSums {
		actNo := i + 1
		var lines []domain.DraftLine
		if actNo == 12 {
			lines = seedAct12Lines
		} else {
			lines = []domain.DraftLine{{Name: "Работы по акту · архив Виталия", Qty: 1, Unit: "компл", Price: sum, Sum: sum}}
		}
		if _, err := s.createActFixedNo(ctx, objID, actNo, lines); err != nil {
			return rep, err
		}
		rep.Total += sum
	}
	rep.Acts = len(seedActSums)
	return rep, nil
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
