// Сметы в чате бота (v0.6) — «вся механика героя работает и из чата».
//
// Сценарий: мастер не всегда на объекте открывает Mini App — иногда быстрее
// черкнуть в чат. Поэтому /smeta даёт тот же конструктор смет: строки тем же
// парсером («штукатурка 45 м² 260»), цены из прайса, мульти-позиции, шаблоны
// работ, типовые помещения, голос, ссылка заказчику — и АКТ ИЗ СМЕТЫ:
// отметил номера строк → акт собрался → XLSX уехал. Одним кодом с мини-аппом:
// FSM кладёт строки сразу в БД (store.AddEstimateLine), никакого дубля данных.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"proakt/internal/domain"
	"proakt/internal/parse"
	"proakt/internal/rooms"
	"proakt/internal/store"
	"proakt/internal/tg"
	"proakt/internal/xlsx"
)

// --- вход: /smeta -------------------------------------------------------------

// showEstimates — список смет + создание новой.
func (b *Bot) showEstimates(ctx context.Context, chatID int64) {
	ests, err := b.st.ListEstimates(ctx, chatID, 0)
	if err != nil {
		b.text(ctx, chatID, "Не смог прочитать сметы 😕")
		return
	}
	if len(ests) == 0 {
		rows := tg.KB{{{Text: "➕ Новая смета", CallbackData: "estnew"}}}
		b.textKB(ctx, chatID, "Смет пока нет.\n\nСмета — план работ до ремонта: заказчик заранее видит\nвсе работы и скрытую подготовку (грунт, шпаклёвка, гидроизоляция) с фото.\nСоздать первую?", tg.Inline(rows))
		return
	}
	var sb strings.Builder
	sb.WriteString("🧾 Сметы:\n\n")
	rows := tg.KB{}
	for _, e := range ests {
		st := estStatusText(e.Status)
		sb.WriteString(fmt.Sprintf("• «%s» — %s · %s (%s)\n", e.Title, e.ObjectName, money(e.Total), st))
		rows = append(rows, []tg.KBButton{{Text: fmt.Sprintf("🧾 %s · %s", e.Title, money(e.Total)), CallbackData: fmt.Sprintf("estopen:%d", e.ID)}})
	}
	rows = append(rows, []tg.KBButton{{Text: "➕ Новая смета", CallbackData: "estnew"}})
	b.textKB(ctx, chatID, sb.String(), tg.Inline(rows))
}

func estStatusText(s string) string {
	switch s {
	case "sent":
		return "отправлена"
	case "approved":
		return "согласована"
	case "done":
		return "закрыта"
	}
	return "черновик"
}

// estMenu — меню открытой сметы (кнопки под списком строк).
func (b *Bot) estMenu(estID int64) tg.InlineKeyboardMarkup {
	return tg.Inline(tg.KB{
		{{Text: "➕ Шаблон", CallbackData: fmt.Sprintf("esttpl:%d", estID)},
			{Text: "🏠 Комната", CallbackData: fmt.Sprintf("estroom:%d", estID)}},
		{{Text: "📋 Акт из сметы", CallbackData: fmt.Sprintf("estact:%d", estID)},
			{Text: "📤 Заказчику", CallbackData: fmt.Sprintf("estshare:%d", estID)}},
		{{Text: "↩️ Убрать последнюю строку", CallbackData: fmt.Sprintf("estundo:%d", estID)}},
		{{Text: "🗑 Удалить смету", CallbackData: fmt.Sprintf("estdel:%d", estID)}},
		{{Text: BtnCancel, CallbackData: "cancel"}},
	})
}

// estnew → выбор объекта.
func (b *Bot) estNewStart(ctx context.Context, chatID int64) {
	objs, err := b.st.ListObjects(ctx, chatID)
	if err != nil || len(objs) == 0 {
		b.textKB(ctx, chatID, "Для сметы нужен объект — сначала создадим.", CancelMenu())
		b.beginObject(ctx, chatID, map[string]string{"from": "est"})
		return
	}
	rows := []tg.KBButton{}
	for _, o := range objs {
		rows = append(rows, tg.KBButton{Text: "🏠 " + o.Name, CallbackData: fmt.Sprintf("estobj:%d", o.ID)})
	}
	rows = append(rows,
		tg.KBButton{Text: BtnNewObj, CallbackData: "objnew"},
		tg.KBButton{Text: BtnCancel, CallbackData: "cancel"})
	b.textKB(ctx, chatID, "Смета по какому объекту?", tg.Inline(tg.KB{rows}))
}

// estobj:<id> → ввод названия.
func (b *Bot) estObjPicked(ctx context.Context, chatID int64, objID int64) {
	_ = b.st.SetState(ctx, chatID, stEstTitle, map[string]string{"object_id": fmt.Sprint(objID)})
	b.textKB(ctx, chatID, "Название сметы?\nНапример: «Кухня 9 м²» или «2-комнатная, коридор + ванная»", CancelMenu())
}

// stEstTitle: текст = название → создаём смету и открываем ввод строк.
func (b *Bot) estTitleFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	title := strings.TrimSpace(text)
	if len([]rune(title)) < 2 || len([]rune(title)) > 80 {
		b.text(ctx, chatID, "Название — от 2 до 80 символов. Попробуй ещё:")
		return
	}
	objID, _ := strconv.ParseInt(data["object_id"], 10, 64)
	e, err := b.st.CreateEstimate(ctx, chatID, objID, title, 1.0, "")
	if err != nil {
		b.text(ctx, chatID, "Не сохранил смету 😕 Попробуй ещё раз: /smeta")
		b.reset(ctx, chatID)
		return
	}
	b.estOpen(ctx, chatID, e.ID)
}

// estOpen — открытая смета: список строк + меню. Вход в режим ввода строк.
func (b *Bot) estOpen(ctx context.Context, chatID int64, estID int64) {
	if err := b.estRender(ctx, chatID, estID); err != nil {
		b.textKB(ctx, chatID, "Смета не найдена 😕 Список: /smeta", MainMenu())
		b.reset(ctx, chatID)
		return
	}
}

// estRender — показать строки и меню, перевести FSM в режим ввода.
func (b *Bot) estRender(ctx context.Context, chatID int64, estID int64) error {
	brief, err := b.st.GetEstimateBrief(ctx, chatID, estID)
	if err != nil {
		return err
	}
	lines, err := b.st.EstimateLines(ctx, estID)
	if err != nil {
		return err
	}
	_ = b.st.SetState(ctx, chatID, stEstLines, map[string]string{"est_id": fmt.Sprint(estID)})

	var sb strings.Builder
	fmt.Fprintf(&sb, "🧾 «%s» · %s · %s\n\n", brief.Title, brief.ObjectName, estStatusText(brief.Status))
	if len(lines) == 0 {
		sb.WriteString("Позиций пока нет. Вводи строкой — как в акте:\n")
		sb.WriteString("  штукатурка 45 м² 260\n")
		sb.WriteString("  шлифовка 45 м          ← цена из прайса\n\n")
		if b.ai != nil {
			sb.WriteString("Или надиктуй голосом 🎤\n\n")
		}
		sb.WriteString("➕ Шаблон — готовый техцикл, 🏠 Комната — целое помещение с нормами (потолок = площадь, стены = периметр × высота).")
	} else {
		for i, l := range lines {
			hidden := ""
			if l.Hidden {
				hidden = " ▪"
			}
			done := ""
			if l.Done {
				done = " ✓"
			}
			sb.WriteString(fmt.Sprintf("%d. %s%s%s\n", i+1, describeLine(draftFromEstLine(l)), hidden, done))
		}
		if brief.Coeff != 1 {
			fmt.Fprintf(&sb, "\nИтого: %s (коэффициент ×%.2f)\n", money(brief.Total), brief.Coeff)
		} else {
			fmt.Fprintf(&sb, "\nИтого: %s\n", money(brief.Total))
		}
		if brief.Hidden > 0 {
			fmt.Fprintf(&sb, "чистовые %s · скрытые %s (%d%%)\n", money(brief.Visible), money(brief.Hidden), brief.HiddenShare)
		}
		sb.WriteString("\nПиши следующую строку или голосом 🎤 · «готово» — свернуть")
	}
	b.textKB(ctx, chatID, sb.String(), b.estMenu(estID))
	return nil
}

// draftFromEstLine — строка сметы в формат describeLine (единый вид с актами).
func draftFromEstLine(l domain.EstimateLine) domain.DraftLine {
	return domain.DraftLine{Name: l.Name, Qty: l.Qty, Unit: l.Unit, Price: l.Price, Sum: l.Sum}
}

// estLineFromText — ввод строки сметы тем же парсером, что у акта (v0.3.6).
func (b *Bot) estLineFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	estID, _ := strconv.ParseInt(data["est_id"], 10, 64)
	if estID <= 0 {
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Смета потерялась — открой заново: /smeta", MainMenu())
		return
	}
	// сворачивание: «готово», «всё», «хватит»…
	if isFinishText(text) {
		b.estRender(ctx, chatID, estID)
		return
	}
	// ↩️ убрать последнюю — по тексту
	if isUndoText(text) {
		b.estUndo(ctx, chatID, estID)
		return
	}

	// мульти-позиции: «шлифовка 45 м грунтовка 45 м»
	if segs := parse.SplitPositions(text); len(segs) >= 2 {
		var added int
		var lastLine domain.EstimateLine
		var failed bool
		for _, seg := range segs {
			l, ok := b.estParseOne(ctx, chatID, estID, seg)
			if !ok {
				failed = true
				break
			}
			added++
			lastLine = l
		}
		if failed && added == 0 {
			b.text(ctx, chatID, multiFailText)
			return
		}
		if added > 0 {
			b.estReportAdded(ctx, chatID, estID, added, lastLine)
			return
		}
		return
	}

	// одиночная строка
	l, ok := b.estParseOne(ctx, chatID, estID, text)
	if !ok {
		return // estParseOne уже объяснил, что не так
	}
	b.estReportAdded(ctx, chatID, estID, 1, l)
}

// estParseOne — разобрать ОДНУ строку и записать в БД (прайс, переспрос).
func (b *Bot) estParseOne(ctx context.Context, chatID, estID int64, text string) (domain.EstimateLine, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return domain.EstimateLine{}, false
	}
	// «наименование количество [ед.]» без цены → прайс
	if name, qty, unit, ok := parse.ParseQty(text); ok {
		items, err := b.st.ListCatalog(ctx, chatID)
		if err == nil {
			if it, ok2 := MatchCatalog(items, name); ok2 {
				if unit == "" {
					unit = it.Unit
				}
				l, err := b.st.AddEstimateLine(ctx, chatID, estID, name, unit, qty, it.Price, guessHiddenName(name), "")
				if err != nil {
					b.text(ctx, chatID, "Не записал строку 😕 Попробуй ещё раз.")
					return domain.EstimateLine{}, false
				}
				_ = b.tg.SendChatAction(ctx, chatID)
				return l, true
			}
		}
		b.text(ctx, chatID, fmt.Sprintf("В прайсе нет «%s» 🤔\nВведи с ценой: %s %s 260 — или добавь цену: 💵 Прайс → ➕", name, name, unit))
		return domain.EstimateLine{}, false
	}
	// двусмысленная строка — переспрос (деньги святы)
	if parse.SuspiciousMulti(text) {
		b.text(ctx, chatID, suspiciousText)
		return domain.EstimateLine{}, false
	}
	line, ok := parse.ParsePosition(text)
	if !ok {
		b.text(ctx, chatID, "Не разобрал строку 🤔\nПримеры:\nштукатурка 45 м² 260\nдемонтаж 2000\nили без цены: штукатурка 45 м²")
		return domain.EstimateLine{}, false
	}
	l, err := b.st.AddEstimateLine(ctx, chatID, estID, line.Name, line.Unit, line.Qty, line.Price, guessHiddenName(line.Name), "")
	if err != nil {
		b.text(ctx, chatID, "Не записал строку 😕 Попробуй ещё раз.")
		return domain.EstimateLine{}, false
	}
	_ = b.tg.SendChatAction(ctx, chatID)
	return l, true
}

// guessHiddenName — словарь скрытых работ в боте (единый с фронтом и rooms).
func guessHiddenName(name string) bool {
	n := strings.ToLower(name)
	for _, k := range []string{"грунт", "шпакл", "шлиф", "армир", "штроб", "укрыв", "заделк", "стык", "гипсокартон", "сетк", "уголок", "профил", "изоляц", "оттяжк", "обеспыл"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// estReportAdded — отчёт после добавления строк (сразу с суммой сметы).
func (b *Bot) estReportAdded(ctx context.Context, chatID, estID int64, added int, last domain.EstimateLine) {
	brief, err := b.st.GetEstimateBrief(ctx, chatID, estID)
	if err != nil {
		b.estRender(ctx, chatID, estID)
		return
	}
	var sb strings.Builder
	if added == 1 {
		sb.WriteString("✅ " + describeLine(draftFromEstLine(last)) + "\n")
	} else {
		sb.WriteString(fmt.Sprintf("✅ Добавил позиций: %d\n", added))
	}
	fmt.Fprintf(&sb, "—\nСмета: %s · позиций %d", money(brief.Total), brief.Lines)
	if brief.Hidden > 0 {
		fmt.Fprintf(&sb, " · скрытые %s", money(brief.Hidden))
	}
	if brief.Coeff != 1 {
		fmt.Fprintf(&sb, " · коэффициент ×%.2f", brief.Coeff)
	}
	b.text(ctx, chatID, sb.String())
}

// estUndo — убрать последнюю строку прямо из БД.
func (b *Bot) estUndo(ctx context.Context, chatID, estID int64) {
	lines, err := b.st.EstimateLines(ctx, estID)
	if err != nil || len(lines) == 0 {
		b.text(ctx, chatID, "В смете нет позиций — убирать нечего.")
		return
	}
	last := lines[len(lines)-1]
	removed, err := b.st.DeleteEstimateLine(ctx, chatID, last.ID)
	if err != nil {
		b.text(ctx, chatID, "Не получилось убрать 😕 Попробуй ещё раз.")
		return
	}
	brief, _ := b.st.GetEstimateBrief(ctx, chatID, estID)
	b.text(ctx, chatID, fmt.Sprintf("↩️ Убрал: %s\n—\nСмета: %s · позиций %d",
		describeLine(draftFromEstLine(removed)), money(brief.Total), brief.Lines))
}

// --- шаблоны работ в смету ------------------------------------------------------

// estTplList — список шаблонов чата для вставки.
func (b *Bot) estTplList(ctx context.Context, chatID, estID int64) {
	tpls, err := b.st.ListTemplates(ctx, chatID)
	if err != nil || len(tpls) == 0 {
		b.textKB(ctx, chatID, "Шаблонов ещё нет — собери первый в 💼 Мини-апп → «Шаблоны»\n(или пришли список работ — соберу вручную).", CancelMenu())
		return
	}
	rows := tg.KB{}
	for _, t := range tpls {
		rows = append(rows, []tg.KBButton{{Text: fmt.Sprintf("🧰 %s (%d)", t.Name, len(t.Lines)), CallbackData: fmt.Sprintf("esttpl2:%d:%d", estID, t.ID)}})
	}
	rows = append(rows, []tg.KBButton{{Text: BtnCancel, CallbackData: "cancel"}})
	b.textKB(ctx, chatID, "Какой шаблон вставить?", tg.Inline(rows))
}

// estTplApply — вставка строк шаблона в смету (цены — из шаблона).
func (b *Bot) estTplApply(ctx context.Context, chatID, estID, tplID int64) {
	tpls, _ := b.st.ListTemplates(ctx, chatID)
	var tpl *domain.Template
	for i := range tpls {
		if tpls[i].ID == tplID {
			tpl = &tpls[i]
			break
		}
	}
	if tpl == nil {
		b.text(ctx, chatID, "Шаблон не нашёлся 😕")
		return
	}
	lines := make([]domain.EstimateLine, 0, len(tpl.Lines))
	for _, tl := range tpl.Lines {
		lines = append(lines, domain.EstimateLine{Name: tl.Name, Qty: tl.Qty, Unit: tl.Unit, Price: tl.Price, Hidden: tl.Hidden, Note: tpl.Name})
	}
	n, err := b.st.AddEstimateLinesBulk(ctx, chatID, estID, lines)
	if err != nil {
		b.text(ctx, chatID, "Не вставил шаблон 😕 Попробуй ещё раз.")
		return
	}
	b.estRender(ctx, chatID, estID)
	b.text(ctx, chatID, fmt.Sprintf("⚡ Шаблон «%s» — вставлено позиций: %d", tpl.Name, n))
}

// --- типовые помещения с нормами -------------------------------------------------

// estRoomList — выбор пресета помещения.
func (b *Bot) estRoomList(ctx context.Context, chatID, estID int64) {
	rows := tg.KB{}
	for _, p := range rooms.Presets {
		rows = append(rows, []tg.KBButton{{
			Text:         fmt.Sprintf("🏠 %s — %s", p.Name, p.Hint),
			CallbackData: fmt.Sprintf("estroom2:%d:%s", estID, p.Key),
		}})
	}
	rows = append(rows, []tg.KBButton{{Text: BtnCancel, CallbackData: "cancel"}})
	b.textKB(ctx, chatID, "Какое помещение? Дальше спросу площадь и высоту —\nобъёмы посчитаю сам: потолок = площадь, стены = периметр × высота.", tg.Inline(rows))
}

// estRoomWait — просим размеры: «9 2.7» (площадь высота) или «12» (высота 2.7 по умолчанию).
func (b *Bot) estRoomWait(ctx context.Context, chatID, estID int64, preset string) {
	_ = b.st.SetState(ctx, chatID, stEstRoomWait, map[string]string{
		"est_id": fmt.Sprint(estID), "preset": preset,
	})
	p, _ := rooms.PresetByKey(preset)
	b.textKB(ctx, chatID, fmt.Sprintf("🏠 %s. Введи размеры:\n\n9 2.7      ← площадь и высота (м)\n9          ← высота 2.7 по умолчанию\n\nПериметр посчитаю сам (≈4·√площади);\nукажешь точный — добавь третьим числом.", p.Name), CancelMenu())
}

// estRoomFromText — размеры → расчёт норм → вставка блока в смету.
func (b *Bot) estRoomFromText(ctx context.Context, chatID int64, data map[string]string, text string) {
	estID, _ := strconv.ParseInt(data["est_id"], 10, 64)
	presetKey := data["preset"]
	fields := strings.Fields(strings.ReplaceAll(text, ",", "."))
	var vals []float64
	for _, f := range fields {
		if v, err := strconv.ParseFloat(f, 64); err == nil && v > 0 && v < 10000 {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		b.text(ctx, chatID, "Не понял размеры 🤔 Пример: 9 2.7 (площадь, высота)")
		return
	}
	area := vals[0]
	height := 2.7
	if len(vals) >= 2 {
		height = vals[1]
	}
	perimeter := 0.0
	if len(vals) >= 3 {
		perimeter = vals[2]
	}
	p, ok := rooms.PresetByKey(presetKey)
	if !ok || estID <= 0 {
		b.text(ctx, chatID, "Помещение потерялось — начни заново: 🏠 Комната")
		return
	}
	built := rooms.Build(p, area, height, perimeter)
	for i := range built {
		// цена из прайса, если позиция там есть — иначе 0 (мастер заполнит)
		if it, ok := b.matchCatalogQuiet(ctx, chatID, built[i].Name); ok {
			built[i].Price = it.Price
			built[i].Sum = built[i].Qty * it.Price
		} else {
			built[i].Sum = 0
		}
	}
	n, err := b.st.AddEstimateLinesBulk(ctx, chatID, estID, built)
	if err != nil {
		b.text(ctx, chatID, "Не вставил помещение 😕 Попробуй ещё раз.")
		return
	}
	noPrice := 0
	for _, l := range built {
		if l.Price == 0 {
			noPrice++
		}
	}
	b.estRender(ctx, chatID, estID)
	note := ""
	if noPrice > 0 {
		note = fmt.Sprintf("\n⚠ Без цены: %d позиций (нет в прайсе) — впиши цены в мини-аппе или диктуй с ценой.", noPrice)
	}
	b.text(ctx, chatID, fmt.Sprintf("🏠 «%s» — %s: вставлено позиций: %d (площадь %s м², высота %s м)%s",
		p.Name, rooms.EstimateName(p.Key, area), n, num(area), num(height), note))
}

// matchCatalogQuiet — тихий поиск в прайсе (ошибки = «не нашли»).
func (b *Bot) matchCatalogQuiet(ctx context.Context, chatID int64, name string) (domain.CatalogItem, bool) {
	items, err := b.st.ListCatalog(ctx, chatID)
	if err != nil {
		return domain.CatalogItem{}, false
	}
	return MatchCatalog(items, name)
}

// --- акт из сметы (еженедельная рутина за 10 минут) -------------------------------

// estActStart — показать строки сметы с номерами и попросить отметить сделанные.
func (b *Bot) estActStart(ctx context.Context, chatID, estID int64) {
	brief, err := b.st.GetEstimateBrief(ctx, chatID, estID)
	if err != nil {
		b.text(ctx, chatID, "Смета не найдена 😕 Список: /smeta")
		return
	}
	lines, err := b.st.EstimateLines(ctx, estID)
	if err != nil {
		b.text(ctx, chatID, "Не прочитал строки 😕")
		return
	}
	var open []domain.EstimateLine
	for _, l := range lines {
		if !l.Done {
			open = append(open, l)
		}
	}
	if len(open) == 0 {
		b.textKB(ctx, chatID, "Все строки сметы уже закрыты актами ✓\nНовые работы — добавляй в смету и отмечай здесь.", MainMenu())
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "📋 Акт за неделю — «%s» (%s):\n\n", brief.Title, brief.ObjectName)
	for i, l := range open {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, describeLine(draftFromEstLine(l)))
	}
	sb.WriteString("\nКакие строки сделаны за эту неделю?\nНапиши номера: 1,3,5 — или слово «все».")
	ids := make([]string, len(open))
	for i, l := range open {
		ids[i] = fmt.Sprint(l.ID)
	}
	_ = b.st.SetState(ctx, chatID, stEstPick, map[string]string{
		"est_id": fmt.Sprint(estID), "ids": strings.Join(ids, ","),
	})
	b.text(ctx, chatID, sb.String())
}

// estActPick — мастер прислал номера («1,3,5» / «все») → акт + XLSX.
func (b *Bot) estActPick(ctx context.Context, chatID int64, data map[string]string, text string) {
	estID, _ := strconv.ParseInt(data["est_id"], 10, 64)
	if estID <= 0 {
		b.reset(ctx, chatID)
		b.textKB(ctx, chatID, "Смета потерялась — /smeta", MainMenu())
		return
	}
	t := strings.ToLower(strings.TrimSpace(text))
	var ids []int64
	if isAllText(t) {
		for _, s := range strings.Split(data["ids"], ",") {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				ids = append(ids, v)
			}
		}
	} else {
		for _, s := range strings.FieldsFunc(t, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '.' || r == '-' || r == '+'
		}) {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 1 {
				continue
			}
			// номер в списке открытых строк → id строки
			parts := strings.Split(data["ids"], ",")
			if n > len(parts) {
				continue
			}
			if v, err := strconv.ParseInt(parts[n-1], 10, 64); err == nil {
				ids = append(ids, v)
			}
		}
	}
	if len(ids) == 0 {
		b.text(ctx, chatID, "Не понял номера 🤔 Напиши через запятую: 1,3,5 — или «все».")
		return
	}
	b.reset(ctx, chatID)

	brief, n, err := b.st.ActFromEstimate(ctx, chatID, estID, ids)
	if err != nil {
		if err == store.ErrNotFound {
			b.textKB(ctx, chatID, "Закрывать нечего: строки без суммы — сначала впиши цены.", MainMenu())
			return
		}
		b.text(ctx, chatID, "Не собрал акт 😕 Попробуй ещё раз: /smeta → 📋 Акт из сметы")
		return
	}
	_ = n

	// XLSX в формате таблицы мастера — тем же генератором, что для обычных актов
	obj, err := b.st.GetObject(ctx, brief.ObjectID)
	if err != nil {
		obj = domain.Object{ID: brief.ObjectID, Name: brief.ObjectName, Customer: brief.Customer}
	}
	actLines, _ := b.st.ActLines(ctx, brief.ID)
	draft := make([]domain.DraftLine, len(actLines))
	for i, l := range actLines {
		draft[i] = domain.DraftLine{Name: l.Name, Qty: l.Qty, Unit: l.Unit, Price: l.Price, Sum: l.Sum}
	}
	_ = b.tg.SendChatAction(ctx, chatID)
	path, xerr := xlsx.GenerateAct(
		b.files+"/acts", brief, obj, actLines, b.cfg.Executor, time.Now())
	if xerr != nil {
		b.textKB(ctx, chatID, fmt.Sprintf("📋 Акт №%d по «%s» записан: %s (%d строк из сметы).\nExcel не собрался 😕 — но данные сохранены.",
			brief.ActNo, brief.ObjectName, money(brief.Total), len(actLines)), MainMenu())
		return
	}
	caption := fmt.Sprintf("Акт №%d · %s · %s (из сметы)", brief.ActNo, brief.ObjectName, money(brief.Total))
	if err := b.tg.SendDocumentFile(ctx, chatID, path, caption); err != nil {
		b.text(ctx, chatID, "Файл не отправился, но акт записан 📋")
	}
	b.textKB(ctx, chatID, fmt.Sprintf("📋 Акт №%d собран из сметы: %s.\nЗакрытые строки отмечены ✓ — в следующий акт не попадут.\nОплату — в 💰 Долги.", brief.ActNo, money(brief.Total)), MainMenu())
}

// isAllText — «все», «всё», «весь», «all».
func isAllText(s string) bool {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, " ", " ")))
	s = strings.Trim(s, " !.,…✅")
	switch s {
	case "все", "всё", "весь", "всех", "все строки", "all":
		return true
	}
	return false
}

// newEstShareToken — криптостойкий токен ссылки заказчику (16 байт = 32 hex).
func newEstShareToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// estShareLink — ссылка заказчику прямо из чата.
func (b *Bot) estShareLink(ctx context.Context, chatID, estID int64) {
	token, err := b.st.SetShareToken(ctx, chatID, estID, newEstShareToken())
	if err != nil {
		b.text(ctx, chatID, "Не создал ссылку 😕 Попробуй ещё раз.")
		return
	}
	brief, _ := b.st.GetEstimateBrief(ctx, chatID, estID)
	url := token
	if b.cfg.WebappURL != "" {
		url = strings.TrimRight(b.cfg.WebappURL, "/") + "/s/" + token
	}
	b.text(ctx, chatID, fmt.Sprintf("🔗 Ссылка заказчику для «%s»:\n%s\n\nСтраница покажет чистовые работы, скрытую подготовку с пояснением\n(«почему так дорого») и фото этапов. Заказчик может принять смету\nи оставить комментарий — придёт сюда в чат.", brief.Title, url))
}

// estDeleteConfirm — подтверждение удаления сметы из чата.
func (b *Bot) estDeleteConfirm(ctx context.Context, chatID, estID int64) {
	brief, err := b.st.GetEstimateBrief(ctx, chatID, estID)
	if err != nil {
		b.text(ctx, chatID, "Смета не найдена 😕")
		return
	}
	_ = b.st.SetState(ctx, chatID, stEstDelConf, map[string]string{"est_id": fmt.Sprint(estID)})
	b.textKB(ctx, chatID, fmt.Sprintf("🗑 Удалить смету «%s» (%s, %d позиций)?\nАкты и оплаты объекта останутся. Вернуть будет нельзя.",
		brief.Title, money(brief.Total), brief.Lines), tg.Inline(tg.KB{
		{{Text: "🗑 Удалить смету", CallbackData: "estdelyes"}, {Text: "❌ Оставить", CallbackData: "estdelno"}},
	}))
}

// estDeleteDo — исполнение удаления.
func (b *Bot) estDeleteDo(ctx context.Context, chatID int64) {
	_, data, _ := b.st.State(ctx, chatID)
	estID, _ := strconv.ParseInt(data["est_id"], 10, 64)
	b.reset(ctx, chatID)
	brief, err := b.st.DeleteEstimate(ctx, chatID, estID)
	if err != nil {
		b.textKB(ctx, chatID, "Смета уже не найдена — может, удалена раньше.", MainMenu())
		return
	}
	b.textKB(ctx, chatID, fmt.Sprintf("🗑 Смета «%s» удалена.", brief.Title), MainMenu())
}
