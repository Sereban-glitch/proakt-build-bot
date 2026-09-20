package app

import (
	"fmt"
	"strings"

	"proakt/internal/tg"
)

// Кнопки главного меню (постоянная клавиатура — крупные мишени).
const (
	BtnNewAct = "📋 Новый акт"
	BtnSmeta  = "🧾 Сметы"
	BtnPhoto  = "📷 Фото"
	BtnDebts  = "💰 Долги"
	BtnPrice  = "💵 Прайс"
	BtnObjs   = "🏠 Объекты"
	BtnMore   = "⚙️ Ещё"
	BtnReport = "📊 Отчёт"
	BtnBack   = "🔙 Главное"
	BtnFinish = "✅ Завершить акт"
	BtnEstFin = "✅ Готово"
	BtnUndo   = "↩️ Убрать последнюю"
	BtnDraft  = "👀 Черновик"
	BtnCancel = "⏹ Отмена"
	BtnNewObj = "➕ Новый объект"

	// v0.3.8: инструкция — одна кнопка в приветствии («владелец не строитель:
	// всё должно быть понятно строителю с первого касания»).
	BtnInstr     = "📖 Инструкция по шагам"
	BtnInstrMenu = "📖 Инструкция"
)

// MainMenu — постоянное меню.
func MainMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnNewAct}, tg.KBButton{Text: BtnPhoto}},
		{tg.KBButton{Text: BtnSmeta}, tg.KBButton{Text: BtnDebts}},
		{tg.KBButton{Text: BtnPrice}, tg.KBButton{Text: BtnObjs}},
		{tg.KBButton{Text: BtnMore}},
	}, "Диктуй или нажми…")
}

// MoreMenu — раздел «Ещё».
func MoreMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnReport}, tg.KBButton{Text: BtnInstrMenu}},
		{tg.KBButton{Text: BtnBack}},
	}, "Ещё…")
}

// ActMenu — контекстное меню во время ввода позиций (v0.3.5: контроль ошибок).
func ActMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnFinish}},
		{tg.KBButton{Text: BtnUndo}, tg.KBButton{Text: BtnDraft}},
		{tg.KBButton{Text: BtnCancel}},
	}, "Вводи позицию строкой или голосом 🎤 Ошибся — ↩️")
}

// EstInputMenu — меню ввода строк сметы (v0.6): свернуть/убрать последнюю.
func EstInputMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnEstFin}},
		{tg.KBButton{Text: BtnUndo}},
		{tg.KBButton{Text: BtnCancel}},
	}, "Строкой или голосом 🎤 Ошибся — ↩️")
}

// CancelMenu — меню с одной кнопкой отмены.
func CancelMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{{tg.KBButton{Text: BtnCancel}}}, "…")
}

// Commands — меню команд Telegram.
func Commands() []tg.BotCommand {
	return []tg.BotCommand{
		{Command: "start", Description: "запуск и инструкция"},
		{Command: "new", Description: "новый акт"},
		{Command: "smeta", Description: "сметы: конструктор, акт из сметы"},
		{Command: "draft", Description: "текущий черновик акта"},
		{Command: "photo", Description: "фото скрытых работ"},
		{Command: "debts", Description: "долги и оплаты"},
		{Command: "price", Description: "прайс-лист, импорт, Google Таблица"},
		{Command: "sync", Description: "синхронизировать прайс с таблицей"},
		{Command: "acts", Description: "список актов"},
		{Command: "objects", Description: "объекты"},
		{Command: "report", Description: "сводка за всё время"},
		{Command: "backup", Description: "выгрузка смет и актов файлом"},
		{Command: "cancel", Description: "отменить действие"},
		{Command: "help", Description: "инструкция по шагам"},
	}
}

// InstrButton — inline-кнопка инструкции (вешается на приветствие, v0.3.8).
func InstrButton() tg.InlineKeyboardMarkup {
	return tg.Inline(tg.KB{
		{tg.KBButton{Text: BtnInstr, CallbackData: "instr"}},
	})
}

// welcome — приветствие (v0.7: развёрнутое, с преимуществами и пошаговым
// планом — «владелец не разработчик, должен понять с первого касания»).
func (b *Bot) welcome() string {
	voice := ""
	if b.ai != nil {
		voice = "3️⃣ Создай смету — голосом 🎤 или из шаблона, бот сам соберёт\n"
	} else {
		voice = "3️⃣ Создай смету — из шаблона или вручную, бот сам посчитает\n"
	}
	return fmt.Sprintf(`Привет! Я ПрорАКТ 360 — твой цифровой прораб на телефоне 🧰

💪 ЧЕМ ЭТО ЛУЧШЕ ТАБЛИЦ:
• Акт прямо с телефона — без тетрадок и вечернего переноса в Excel
• Бот считает сам — суммы, объёмы, остатки по оплатам
• Прайс внутри — не искать цены, пиши название, цена подставится
• Заказчик открывает акт по ссылке и соглашается — без переписки
• История по объектам — всё на виду, ничего не теряется

Пять шагов — и готово:
1️⃣ Загрузи прайс (💵 Прайс) — один раз, дальше работает
2️⃣ Создай объект (🏠) — адрес и заказчик
%s4️⃣ Отправь заказчику (📤) — он откроет, подтвердит, заплатит
5️⃣ Готовый акт (✅) — сразу в Excel

Нажми кнопку внизу 👇`, voice)
}

// help — справка = пошаговая инструкция (v0.3.8: один источник правды).
func (b *Bot) help() string {
	return b.instruction()
}

// instruction — пошаговая инструкция на языке строителя (v0.3.8).
// Требование владельца: «навигация должна быть интуитивно понятной, пошагово,
// добавить и удалить — всё в обе стороны». Поэтому: 7 шагов работы +
// раздел «ОШИБСЯ? УДАЛИТЬ МОЖНО ВСЁ» + масштабируемость (форматы/таблицы).
func (b *Bot) instruction() string {
	voice := ""
	if b.ai != nil {
		voice = "\n  Можно и голосом 🎤 — как говоришь: «штукатурка сорок пять по двести шестьдесят»."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `📖 ИНСТРУКЦИЯ — 7 ШАГОВ

ШАГ 1️⃣ — ЦЕНЫ (один раз в начале)
Перешли файл прайса прямо в чат → 💵 Прайс → 📥 Импорт из файла.
Дальше в акте достаточно написать «шлифовка 45 м» — цена подставится сама.
Цена изменилась — 💵 Прайс → ➕ Добавить: «шлифовка 200».

ШАГ 2️⃣ — ОБЪЕКТ (квартира или дом)
🏠 Объекты → ➕ Новый объект → название («ЖК Сонячний, кв. 45») и заказчик.

ШАГ 3️⃣ — АКТ (по ходу работ)
📋 Новый акт → выбери объект → и пиши как говоришь:
  штукатурка 45 м² 260    → 45 × 260 = 11 700 грн
  шлифовка 45 м           → цену возьмёт из прайса
  демонтаж стен 2000      → готовая сумма%s
Ошибся — ↩️ Убрать последнюю. Проверить всё — 👀 Черновик.

ШАГ 4️⃣ — ФАЙЛ АКТА
✅ Завершить акт → пришлю готовый Excel (укр.) — пересылай заказчику сразу.

ШАГ 4½ — СМЕТЫ (/smeta) — план работ ДО ремонта
🧾 Сметы → новая → вводи строки (или диктуй 🎤) → «📤 Заказчику».
🏠 Комната — «Кухня 9 м²» вставит весь техцикл сама (потолок = площадь,
стены = периметр × высота). 📋 Акт из сметы — отметил сделанное → готов Excel.

ШАГ 5️⃣ — ДЕНЬГИ ПРИШЛИ
💰 Долги → «💵 Оплата · акт №…» → сумма (например 10000). Остаток покажу сам.

ШАГ 6️⃣ — СКРЫТЫЕ РАБОТЫ (фото ДО закрытия)
📷 Фото (или просто пришли снимок) → подпись («электрика до штукатурки»)
→ выбери акт или объект.
Показать заказчику: 🏠 Объекты → кнопка «📷 …фото».

ШАГ 7️⃣ — КОНТРОЛЬ ДЕНЕГ
📊 Отчёт — сколько сделано, оплачено и где деньги «мимо кармана»:
не оплаченные акты и акты с нулём видны сразу.

🗑 ОШИБСЯ? УДАЛИТЬ МОЖНО ВСЁ:
• позицию в акте → ↩️ Убрать последнюю
• весь акт → /acts → 🗑
• неудачное фото → 🏠 Объекты → 📷 фото → кнопка 🗑 под снимком
• ошибочную оплату → /acts → «💰 оплаты №…» → 🗑
• позицию прайса → 💵 Прайс → 🗑 Убрать позицию
• лишний объект → 🏠 Объекты → 🗑 Убрать объект (акты и деньги останутся)

🛠 НУЖНО БОЛЬШЕ?
Другой формат акта (PDF/CSV), свои колонки, дополнительные таблицы —
всё это добавляется и настраивается. Скажи, что нужно — сделаем.

Команды: /new /draft /price /photo /debts /acts /objects /report /cancel`, voice)
	return sb.String()
}

const notUnderstood = "Не понял 🤔 Жми кнопки внизу или 📖 /help"
