// Package config загружает настройки ПрорАКТА из переменных окружения (.env подгружает systemd).
package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	BotToken   string // токен Telegram-бота
	DBDSN      string // строка подключения к PostgreSQL
	FilesDir   string // каталог для фото и файлов актов (например, /var/lib/proakt/files)
	Executor   string // имя исполнителя в актах (укр.)
	AIGateway  string // URL Anthropic-совместимого LLM-шлюза (для голоса)
	AIModel    string // модель шлюза для текста (gemini-3-flash)
	AIModelASR string // модель шлюза для распознавания голоса (gemini-2.5-flash)
	AIKey      string // опциональный Bearer-ключ шлюза

	// Mini App (v0.4): пустой Listen — веб-часть выключена (0 портов, как раньше).
	WebappListen  string        // адрес HTTP-сервера Mini App, например ":8443"
	WebappURL     string        // публичный https-адрес Mini App (кнопка в меню бота)
	WebappAuthTTL time.Duration // максимум возраста initData (0 — 24 часа)

	// v0.6: интервал фоновой синхронизации прайса с Google Таблицей
	// (PRICE_SYNC_INTERVAL, формат Go duration; 0 — 6 часов)
	PriceSyncInterval time.Duration
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load проверяет обязательные переменные и заполняет дефолты.
func Load() (*Config, error) {
	syncInterval := 6 * time.Hour
	if v := os.Getenv("PRICE_SYNC_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= time.Minute {
			syncInterval = d
		}
	}
	cfg := &Config{
		BotToken:   os.Getenv("BOT_TOKEN"),
		DBDSN:      os.Getenv("DB_DSN"),
		FilesDir:   getenv("FILES_DIR", "files"),
		Executor:   getenv("EXECUTOR_NAME", "Виконавець"),
		AIGateway:  os.Getenv("AI_GATEWAY_URL"),
		AIModel:    getenv("AI_MODEL", "gemini-3-flash"),
		AIModelASR: getenv("AI_MODEL_ASR", "gemini-2.5-flash"),
		AIKey:      os.Getenv("AI_GATEWAY_KEY"),

		WebappListen:  os.Getenv("WEBAPP_LISTEN"),
		WebappURL:     os.Getenv("WEBAPP_URL"),
		WebappAuthTTL: 24 * time.Hour,

		PriceSyncInterval: syncInterval,
	}
	if cfg.BotToken == "" {
		return nil, errors.New("BOT_TOKEN не задан")
	}
	if cfg.DBDSN == "" {
		return nil, errors.New("DB_DSN не задан")
	}
	return cfg, nil
}
