// photoprobe — live-проверка показа фото скрытых работ (v0.3.7).
// Вызывает ТОТ ЖЕ код, что и бот в production (tg.Client.SendPhoto).
// Использование на VM (токен из .env, чат из БД):
//
//	photoprobe -token $TOKEN -chat 123456789 -fileid <FILE_ID> -caption "тест"
//
// Шаг 1 (получить file_id) делается curl-загрузкой тестового JPEG:
//
//	curl -sS -F chat_id=$CHAT -F photo=@demo.jpg \
//	     https://api.telegram.org/bot$TOKEN/sendPhoto | jq '.result.photo[-1].file_id'
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"proakt/internal/tg"
)

func main() {
	token := flag.String("token", "", "BOT_TOKEN (из .env на VM)")
	chat := flag.Int64("chat", 0, "chat_id получателя")
	fileID := flag.String("fileid", "", "file_id фото из Telegram")
	caption := flag.String("caption", "", "подпись к фото")
	flag.Parse()

	if *token == "" || *chat == 0 || *fileID == "" {
		log.Fatalf("нужны -token, -chat, -fileid (см. пример в шапке файла)")
	}

	c := tg.New(*token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.SendPhoto(ctx, *chat, *fileID, *caption, nil); err != nil {
		log.Fatalf("ОШИБКА sendPhoto: %v", err)
	}
	log.Printf("OK: фото отправлено по file_id (chat %d)", *chat)
}
