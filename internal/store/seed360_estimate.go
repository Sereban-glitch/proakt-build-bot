// Сид сметы «ПрорАКТ 360» — малярные работы Парковый 2.
//
// Источник: internal/webapp/static/js/services/mock.js, estLines[3]
// (38 позиций, итог 190383.50). Цифры — демо-наполнение, суммы строк
// считаются как qty*price. Повторный запуск идемпотентен: смету ищем
// по title у этого объекта, дублей не создаём.
package store

import (
	"context"

	"proakt/internal/domain"
)

// seed360EstimateTitle — title сметы, по нему идемпотентность.
const seed360EstimateTitle = "Малярные работы · этап под покраску"

const seed360EstimateNote = "Источник: файл «Парковый 2». 38 позиций с известным объёмом · 190 383,50 ₴. Неоценённая укрывка в итог не включена."

// seed360EstimateLines — 1-в-1 из mock.js estLines[3] (pos 1..38).
var seed360EstimateLines = []domain.EstimateLine{
	{Name: "укрывка окон гофрокартоном", Qty: 12, Unit: "м.п", Price: 25, Hidden: true},
	{Name: "заделка штроб", Qty: 47, Unit: "м.п", Price: 60, Hidden: true},
	{Name: "поклейка пенополистирола на верхний откос балкона", Qty: 1, Unit: "шт", Price: 100, Hidden: true},
	{Name: "грунтовка откосов перед штукатуркой", Qty: 18.5, Unit: "м.п", Price: 25, Hidden: true},
	{Name: "установка перфорированного пластикового уголка", Qty: 9.8, Unit: "м.п", Price: 80, Hidden: true},
	{Name: "армировка откосов стекловолоконной сеткой", Qty: 18.5, Unit: "м.п", Price: 80, Hidden: true},
	{Name: "отпуск дверных проёмов", Qty: 5, Unit: "шт", Price: 200, Hidden: true},
	{Name: "шлифовка стен штукатурки перед шпаклёвкой", Qty: 146.4, Unit: "м²", Price: 50, Hidden: true},
	{Name: "шлифовка откосов перед шпаклёвкой", Qty: 97.7, Unit: "м.п", Price: 50, Hidden: true},
	{Name: "грунтовка стен перед шпаклёвкой", Qty: 146.4, Unit: "м²", Price: 25, Hidden: true},
	{Name: "грунтовка откосов перед шпаклёвкой", Qty: 97.7, Unit: "м.п", Price: 25, Hidden: true},
	{Name: "шпаклёвка стен под стеклохолст", Qty: 140.5, Unit: "м²", Price: 140, Hidden: true},
	{Name: "шпаклёвка откосов под стеклохолст", Qty: 97.7, Unit: "м.п", Price: 140, Hidden: true},
	{Name: "шлифовка стен под стеклохолст", Qty: 140.5, Unit: "м²", Price: 50, Hidden: true},
	{Name: "шлифовка откосов под стеклохолст", Qty: 97.7, Unit: "м.п", Price: 50, Hidden: true},
	{Name: "грунтовка стен перед стеклохолстом", Qty: 75, Unit: "м²", Price: 25, Hidden: true},
	{Name: "грунтовка откосов перед стеклохолстом", Qty: 79.7, Unit: "м.п", Price: 25, Hidden: true},
	{Name: "поклейка стеклохолста на стены", Qty: 75, Unit: "м²", Price: 120, Hidden: true},
	{Name: "поклейка стеклохолста на откосы", Qty: 79.7, Unit: "м.п", Price: 120, Hidden: true},
	{Name: "шпаклёвка стен под покраску по стеклохолсту", Qty: 75, Unit: "м²", Price: 200, Hidden: true},
	{Name: "шпаклёвка откосов под покраску", Qty: 79.7, Unit: "м.п", Price: 200, Hidden: true},
	{Name: "шлифовка стен под покраску", Qty: 75, Unit: "м²", Price: 50, Hidden: true},
	{Name: "шлифовка откосов под покраску", Qty: 79.7, Unit: "м.п", Price: 50, Hidden: true},
	{Name: "грунтовка гипсовых панелей перед монтажом", Qty: 5.9, Unit: "м²", Price: 25, Hidden: true},
	{Name: "поклейка гипсовых панелей", Qty: 5.9, Unit: "м²", Price: 700, Hidden: false},
	{Name: "блок для розеток на гипсовых панелях", Qty: 1, Unit: "шт", Price: 500, Hidden: true},
	{Name: "шпаклёвка и шлифовка стыков гипсовых панелей", Qty: 5.9, Unit: "м²", Price: 500, Hidden: true},
	{Name: "грунтовка гипсовых панелей под покраску", Qty: 5.9, Unit: "м²", Price: 75, Hidden: true},
	{Name: "грунтовка стен под покраску", Qty: 129.3, Unit: "м²", Price: 25, Hidden: true},
	{Name: "грунтовка откосов под покраску", Qty: 81.7, Unit: "м.п", Price: 25, Hidden: true},
	{Name: "нанесение грунт-краски на гипсовые панели", Qty: 5.9, Unit: "м²", Price: 180, Hidden: true},
	{Name: "нанесение грунт-краски на стены", Qty: 129.3, Unit: "м²", Price: 60, Hidden: true},
	{Name: "нанесение грунт-краски на откосы", Qty: 81.7, Unit: "м.п", Price: 60, Hidden: true},
	{Name: "акрил на углы и разделение цветов", Qty: 16.2, Unit: "м.п", Price: 170, Hidden: true},
	{Name: "покраска гипсовых панелей безвоздушно", Qty: 5.9, Unit: "м²", Price: 360, Hidden: false},
	{Name: "покраска стен безвоздушным методом", Qty: 129.3, Unit: "м²", Price: 120, Hidden: false, Note: "В работе"},
	{Name: "покраска откосов безвоздушным методом", Qty: 81.7, Unit: "м.п", Price: 120, Hidden: false, Note: "Следующий этап"},
	{Name: "армировка примыкания балконного остекления", Qty: 5.6, Unit: "м.п", Price: 250, Hidden: true},
}

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
