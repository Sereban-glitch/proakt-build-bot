// Package tg — минимальный типизированный клиент Telegram Bot API (long polling).
// Сознательно без внешних библиотек: меньше зависимостей — меньше сюрпризов.
package tg

import "fmt"

// Update — входящее событие.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// User — пользователь Telegram.
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

// Chat — чат.
type Chat struct {
	ID int64 `json:"id"`
}

// PhotoSize — один размер фото.
type PhotoSize struct {
	FileID string `json:"file_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Voice — голосовое сообщение (OGG/Opus).
type Voice struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

// Message — сообщение.
type Message struct {
	MessageID int64       `json:"message_id"`
	Chat      Chat        `json:"chat"`
	From      *User       `json:"from,omitempty"`
	Text      string      `json:"text,omitempty"`
	Caption   string      `json:"caption,omitempty"`
	Photo     []PhotoSize `json:"photo,omitempty"`
	Voice     *Voice      `json:"voice,omitempty"`
}

// CallbackQuery — нажатие inline-кнопки.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Data    string   `json:"data,omitempty"`
	Message *Message `json:"message,omitempty"`
}

// BotCommand — пункт меню команд.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// KBButton — кнопка.
type KBButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// KB — сетка кнопок.
type KB [][]KBButton

// ReplyKeyboardMarkup — постоянная клавиатура (большие кнопки-мишени).
type ReplyKeyboardMarkup struct {
	Keyboard              KB     `json:"keyboard"`
	ResizeKeyboard        bool   `json:"resize_keyboard"`
	IsPersistent          bool   `json:"is_persistent"`
	InputFieldPlaceholder string `json:"input_field_placeholder,omitempty"`
}

// InlineKeyboardMarkup — контекстные кнопки внутри сообщения.
type InlineKeyboardMarkup struct {
	InlineKeyboard KB `json:"inline_keyboard"`
}

// ReplyKeyboardRemove — убрать постоянную клавиатуру.
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

// Reply — собрать постоянную клавиатуру.
func Reply(rows KB, placeholder string) ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{Keyboard: rows, ResizeKeyboard: true, IsPersistent: true, InputFieldPlaceholder: placeholder}
}

// Inline — собрать inline-клавиатуру.
func Inline(rows KB) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{InlineKeyboard: rows}
}

// APIError — ошибка Telegram API (с кодом и задержкой при лимитах).
type APIError struct {
	Code        int
	Description string
	RetryAfter  int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram api: %d %s", e.Code, e.Description)
}
