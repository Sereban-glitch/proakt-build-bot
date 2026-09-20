// Сид сметы «ПрорАКТ 360» — малярные работы Парковый 2.
//
// Источник: internal/webapp/static/js/services/mock.js, estLines[3]
// (38 позиций, итог 190383.50). Цифры — демо-наполнение, суммы строк
// считаются как qty*price. Повторный запуск идемпотентен: смету ищем
// по title у этого объекта, дублей не создаём.
package store

import (
	"context"
)

// seed360EstimateTitle — title сметы, по нему идемпотентность.
const seed360EstimateTitle = "Малярные работы · этап под покраску"

const seed360EstimateNote = "Источник: файл «Парковый 2». 38 позиций с известным объёмом · 190 383,50 ₴. Неоценённая укрывка в итог не включена."

// seed360EstimateLines — в seed360_estimate_lines_gen.go (38 позиций из xlsx).

// Seed360Estimate заливает демо-смету Парковый 2 под объект.
// Идемпотентно: если смета с таким title уже есть у объекта —
// возвращает её id и итог без дублей.
func (s *Store) Seed360Estimate(ctx context.Context, chatID, objectID int64) (estID int64, total float64, err error) {
	ests, err := s.ListEstimates(ctx, chatID, objectID)
	if err != nil {
		return 0, 0, err
	}
	for _, b := range ests {
		if b.Title == seed360EstimateTitle {
			return b.ID, b.Total, nil
		}
	}

	e, err := s.CreateEstimate(ctx, chatID, objectID, seed360EstimateTitle, 1, seed360EstimateNote)
	if err != nil {
		return 0, 0, err
	}
	// Статус как в mock.js (approved) — смета согласована, в работе 2 позиции.
	if st := "approved"; true {
		if _, uerr := s.UpdateEstimate(ctx, chatID, e.ID, nil, &st, nil, nil); uerr != nil {
			return 0, 0, uerr
		}
	}

	if _, err := s.AddEstimateLinesBulk(ctx, chatID, e.ID, seed360EstimateLines); err != nil {
		return 0, 0, err
	}
	for _, l := range seed360EstimateLines {
		total += l.Qty * l.Price
	}
	return e.ID, total, nil
}

// Seed360EstimateProgress помечает готовые строки (все кроме 2 финальных).
// Идемпотентно, живые правки Виталика (снятие done) не трогает повторно:
// ставит done только там, где note пустой.
func (s *Store) Seed360EstimateProgress(ctx context.Context, chatID, objectID int64) (int64, error) {
	ests, err := s.ListEstimates(ctx, chatID, objectID)
	if err != nil {
		return 0, err
	}
	for _, b := range ests {
		if b.Title != seed360EstimateTitle {
			continue
		}
		tag, err := s.pool.Exec(ctx,
			`UPDATE estimate_lines SET done=true WHERE est_id=$1 AND (note IS NULL OR note='') AND done=false`, b.ID)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}
	return 0, nil
}
