// ПрорАКТ 360 (@proakt_build_bot) — Telegram-бот строителя-отделочника.
// v0.1: объекты, акты (текстовый ввод), Excel-файлы, оплаты/долги, фото.
// v0.2: голосовой ввод позиций через Anthropic-совместимый LLM-шлюз (/v1/messages).
// v0.2.2: аудио блоком «image»+audio/ogg (обход ограничения конвертера шлюза) — см. internal/ai/gateway.go.
// v0.3: прайс-лист — каталог цен, автоподстановка в акты, импорт из Excel/CSV.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proakt/internal/ai"
	"proakt/internal/app"
	"proakt/internal/config"
	"proakt/internal/httpapi"
	"proakt/internal/store"
	"proakt/internal/tg"
)

// version — подставляется при релизной сборке:
// go build -ldflags "-s -w -X main.version=v0.4.0"
var version = "dev"

func main() {
	migrateOnly := flag.Bool("migrate", false, "применить схему БД и выйти")
	showVersion := flag.Bool("version", false, "показать версию и выйти")
	flag.Parse()

	if *showVersion {
		fmt.Println("proakt", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	st, err := store.New(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("подключение к БД: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("миграции: %v", err)
	}
	log.Printf("БД: схема готова")

	if *migrateOnly {
		return
	}

	client := tg.New(cfg.BotToken)
	me, err := client.GetMe(ctx)
	if err != nil {
		log.Fatalf("токен бота не работает: %v", err)
	}
	if err := client.DeleteWebhook(ctx); err != nil {
		var api *tg.APIError
		if !errors.As(err, &api) {
			log.Fatalf("webhook: %v", err)
		}
	}
	if err := client.SetCommands(ctx, app.Commands()); err != nil {
		log.Printf("setMyCommands: %v", err)
	}

	var gw *ai.Gateway
	if cfg.AIGateway != "" {
		gw = ai.New(cfg.AIGateway, cfg.AIKey, cfg.AIModel, cfg.AIModelASR)
	}

	bot := app.New(cfg, client, st, gw)
	// v0.6: фоновые циклы — автобэкап (раз в сутки) и синк прайса
	// из Google Таблиц (интервал PRICE_SYNC_INTERVAL, по умолчанию 6ч)
	bot.StartBackupLoop(ctx)
	bot.StartPriceSyncLoop(ctx, cfg.PriceSyncInterval)
	log.Printf("старт: бот @%s (%s), long polling", me.Username, me.FirstName)
	if gw != nil {
		log.Printf("ai: шлюз %s (текст %s, голос %s) — голос включён",
			cfg.AIGateway, cfg.AIModel, cfg.AIModelASR)
	} else {
		log.Printf("ai: AI_GATEWAY_URL не задан — голос выключен, ввод текстом")
	}

	// NOPOLL=1 — режим staging-пилота: только Mini App + API, без long polling
	// (иначе второй процесс на том же токене отбирал бы обновления у прод-бота).
	// Кнопку меню тоже не трогаем — прод-бот продолжает вести на свой URL.
	nopoll := os.Getenv("NOPOLL") == "1"

	// Mini App (v0.4): если задан WEBAPP_LISTEN — поднимаем HTTP-сервер
	// с API и встроенным веб-приложением. Не задан — всё как раньше (0 портов).
	if cfg.WebappListen != "" {
		wsrv := httpapi.New(httpapi.Config{
			Listen:    cfg.WebappListen,
			BotToken:  cfg.BotToken,
			AuthTTL:   cfg.WebappAuthTTL,
			FilesDir:  cfg.FilesDir,
			PublicURL: cfg.WebappURL, // share-ссылки смет строятся от него
			Gateway:   gw,            // v0.6: голос в мини-апп (nil — 503)
		}, st)
		wsrv.SetNotifier(bot) // v0.6: заказчик согласовал/спросил → в чат
		go func() {
			if err := wsrv.Run(ctx); err != nil {
				log.Printf("webapp: %v", err)
			}
		}()
		if cfg.WebappURL != "" && !nopoll {
			if err := client.SetChatMenuButton(ctx, "🧰 ПрорАКТ 360", cfg.WebappURL); err != nil {
				log.Printf("setChatMenuButton: %v", err)
			} else {
				log.Printf("webapp: кнопка меню установлена — %s", cfg.WebappURL)
			}
		} else {
			log.Printf("webapp: WEBAPP_URL не задан или NOPOLL — кнопка в меню бота не обновлена")
		}
	}

	if nopoll {
		log.Printf("NOPOLL=1: polling выключен, только Mini App — жду сигнал остановки")
		<-ctx.Done()
		log.Printf("остановка по сигналу")
		return
	}

	offset := int64(0)
	for {
		if ctx.Err() != nil {
			log.Printf("остановка по сигналу")
			return
		}
		ups, err := client.GetUpdates(ctx, offset, 25)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			var api *tg.APIError
			if errors.As(err, &api) && api.RetryAfter > 0 {
				log.Printf("лимит Telegram: пауза %d c", api.RetryAfter)
				time.Sleep(time.Duration(api.RetryAfter) * time.Second)
			} else {
				log.Printf("getUpdates: %v", err)
				time.Sleep(3 * time.Second)
			}
			continue
		}
		for _, u := range ups {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			bot.Handle(ctx, u)
		}
	}
}
