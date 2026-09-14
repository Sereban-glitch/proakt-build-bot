// Уведомления мастеру (v0.6): заказчик согласовал смету или оставил вопрос
// на публичной странице — сообщение прилетает в чат мгновенно, а не «когда
// мастер откроет приложение». Реализация интерфейса httpapi.Notifier.
package app

import (
	"context"
	"fmt"
	"log"
	"time"
)

// sendTimeout — мягкий таймаут на уведомление (Telegram может «думать»).
const sendTimeout = 10 * time.Second

// NotifyEstimateApproved — «заказчик принял смету».
func (b *Bot) NotifyEstimateApproved(chatID int64, title string) {
	b.notifyChat(chatID, fmt.Sprintf(
		"🎉 Заказчик принял смету «%s»!\nМожно закупать материалы и планировать сроки.\nСтатус сметы обновлён: согласована.", title))
}

// NotifyClientComment — вопрос заказчика по смете.
func (b *Bot) NotifyClientComment(chatID int64, title, text string) {
	b.notifyChat(chatID, fmt.Sprintf(
		"💬 Вопрос по смете «%s»:\n%s\n\nОтветить можно в мини-апп (вкладка «Сметы» → диалог) — заказчик увидит ответ на странице.", title, text))
}

// notifyChat — защищённая отправка (уведомление не должно ронять веб-запрос).
func (b *Bot) notifyChat(chatID int64, text string) {
	defer func() { recover() }()
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	if err := b.tg.SendMessage(ctx, chatID, text, nil); err != nil {
		log.Printf("уведомление чат %d: %v", chatID, err)
	}
}
