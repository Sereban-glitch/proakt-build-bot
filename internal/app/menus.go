package app

import "proakt/internal/tg"

// Кнопки главного меню (постоянная клавиатура — крупные мишени).
const (
	BtnNewAct = "📋 Новый акт"
	BtnPhoto  = "📷 Фото"
	BtnDebts  = "💰 Долги"
	BtnObjs   = "🏠 Объекты"
	BtnMore   = "⚙️ Ещё"
	BtnReport = "📊 Отчёт"
	BtnBack   = "🔙 Главное"
	BtnFinish = "✅ Завершить акт"
	BtnCancel = "⏹ Отмена"
	BtnNewObj = "➕ Новый объект"
)

// MainMenu — постоянное меню.
func MainMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnNewAct}, tg.KBButton{Text: BtnPhoto}},
		{tg.KBButton{Text: BtnDebts}, tg.KBButton{Text: BtnObjs}},
		{tg.KBButton{Text: BtnMore}},
	}, "Диктуй или нажми…")
}

// MoreMenu — раздел «Ещё».
func MoreMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnReport}, tg.KBButton{Text: BtnBack}},
	}, "Ещё…")
}

// ActMenu — контекстное меню во время ввода позиций.
func ActMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{
		{tg.KBButton{Text: BtnFinish}, tg.KBButton{Text: BtnCancel}},
	}, "Вводи позицию строкой…")
}

// CancelMenu — меню с одной кнопкой отмены.
func CancelMenu() tg.ReplyKeyboardMarkup {
	return tg.Reply(tg.KB{{tg.KBButton{Text: BtnCancel}}}, "…")
}

// Commands — меню команд Telegram.
func Commands() []tg.BotCommand {
	return []tg.BotCommand{
		{Command: "start", Description: "запуск и статус"},
		{Command: "new", Description: "новый акт"},
		{Command: "photo", Description: "фото скрытых работ"},
		{Command: "debts", Description: "долги и оплаты"},
		{Command: "acts", Description: "список актов"},
		{Command: "objects", Description: "объекты"},
		{Command: "report", Description: "сводка за всё время"},
		{Command: "cancel", Description: "отменить действие"},
		{Command: "help", Description: "помощь"},
	}
}

// welcome — приветствие (с голосом, если шлюз настроен).
func (b *Bot) welcome() string {
	if b.ai != nil {
		return `Привет! Я Прораб — твой цифровой прораб на телефоне 🧰

Что умею:
📋 Новый акт — позиции строкой или голосом 🎤, файл Excel готов сразу
📷 Фото — фото скрытых работ с привязкой к акту
💰 Долги — кто сколько должен, оплаты на месте
🏠 Объекты — квартиры и заказчики
📊 Отчёт — сводка по деньгам

Пример ввода позиции:
штукатурка 45 260
(наименование, количество, цена — «м²» можно писать после количества)
или готовой суммой: демонтаж стен 2000

В акте можно и надиктовать: «штукатурка сорок пять метров двести шестьдесят» — я распознаю и покажу на проверку.

Жми кнопки — и поехали!`
	}
	return welcomeText
}

// help — справка (с голосом, если шлюз настроен).
func (b *Bot) help() string {
	if b.ai != nil {
		return helpText + `
6) 🎤 Голосом: в акте надиктуй позиции одной фразой или несколько подряд —
   бот расшифрует и покажет список на подтверждение`
	}
	return helpText
}

const welcomeText = `Привет! Я Прораб — твой цифровой прораб на телефоне 🧰

Что умею:
📋 Новый акт — позиции строкой, файл Excel готов сразу
📷 Фото — фото скрытых работ с привязкой к акту
💰 Долги — кто сколько должен, оплаты на месте
🏠 Объекты — квартиры и заказчики
📊 Отчёт — сводка по деньгам

Пример ввода позиции:
штукатурка 45 260
(наименование, количество, цена — «м²» можно писать после количества)
или готовой суммой: демонтаж стен 2000

Жми кнопки — и поехали!`

const helpText = `Как пользоваться:

1) 🏠 Объекты → ➕ Новый объект — заведи квартиру/дом (название + заказчик)
2) 📋 Новый акт → выбери объект → вводи позиции построчно:
   штукатурка 45 м² 260   → 45 м² × 260 грн = 11 700 грн
   демонтаж стен 2000     → готовая сумма 2000 грн
3) ✅ Завершить акт → бот пришлёт Excel-файл акта (укр.)
4) 💰 Долги → отметь оплату — увидишь остаток
5) 📷 Фото → пришли фото скрытых работ, бот привяжет к акту

Команды: /new /photo /debts /acts /objects /report /cancel
Отчёт-чеклист проекта приходит сюда же автоматически.`

const notUnderstood = "Не понял 🤔 Жми кнопки внизу или /help"
