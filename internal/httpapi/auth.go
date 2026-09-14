// Package httpapi — HTTP-сервер для Telegram Mini App (v0.4).
//
// Единая точка входа веб-приложения: раздача статики (встроена в бинарник
// через internal/webapp) и JSON REST API поверх того же store, которым
// пользуется бот. Каждый запрос подписан Telegram initData — валидируем
// HMAC-SHA256 и работаем строго с данными своего chat_id.
package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TMAUser — пользователь, извлечённый из initData.
type TMAUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Language  string `json:"language_code,omitempty"`
	IsPremium bool   `json:"is_premium,omitempty"`
}

// Name — отображаемое имя (FirstName [+ LastName]).
func (u TMAUser) Name() string {
	n := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if n == "" {
		n = u.Username
	}
	return n
}

// errBadInitData — общий ответ на любую проблему подписи (детали — в логи).
var errBadInitData = errors.New("initData: подпись не прошла проверку")

// ValidateInitData — каноничная проверка подписи Telegram WebApp (docs:
// "Validating data received via the Mini App"). Алгоритм:
//
//	secret_key   = HMAC_SHA256(key=bot_token,   msg="WebAppData")
//	data_check   = "k=v\n..." (все поля, отсортированные по ключу, без hash;
//	                          поле signature — ВКЛЮЧАЕТСЯ в расчёт, так
//	                          Telegram считает с 2026)
//	calculated   = HMAC_SHA256(key=secret_key,  msg=data_check) — hex
//
// Дополнительно: auth_date не старше ttl (защита от replay), user обязателен.
func ValidateInitData(raw, botToken string, ttl time.Duration, now time.Time) (TMAUser, error) {
	if raw == "" || botToken == "" {
		return TMAUser{}, errBadInitData
	}
	// Authorization: tma <initData> — официальная схема для Mini App.
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "tma "))

	vals, err := url.ParseQuery(raw)
	if err != nil {
		return TMAUser{}, errBadInitData
	}
	hash := vals.Get("hash")
	if hash == "" {
		return TMAUser{}, errBadInitData
	}
	vals.Del("hash")
	// ВАЖНО: поле "signature" НЕ удаляем — с 2026-формата Telegram включает
	// его в data_check при расчёте hash (подтверждено на реальном устройстве).

	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sortStrings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+vals.Get(k))
	}
	dataCheck := strings.Join(parts, "\n")

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	calc := hex.EncodeToString(hmacSHA256(secret, []byte(dataCheck)))
	if !hmac.Equal([]byte(calc), []byte(strings.ToLower(hash))) {
		return TMAUser{}, errBadInitData
	}

	// Свежесть: Telegram присылает auth_date (unix). Старые подписи — replay.
	ad, err := strconv.ParseInt(vals.Get("auth_date"), 10, 64)
	if err != nil || ad <= 0 {
		return TMAUser{}, errBadInitData
	}
	if ttl > 0 && now.Sub(time.Unix(ad, 0)) > ttl {
		return TMAUser{}, fmt.Errorf("initData: устарел (auth_date %d)", ad)
	}

	var u TMAUser
	if err := json.Unmarshal([]byte(vals.Get("user")), &u); err != nil || u.ID <= 0 {
		return TMAUser{}, errBadInitData
	}
	return u, nil
}

func hmacSHA256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return h.Sum(nil)
}

// sortStrings — insertion sort, чтобы не тянуть sort ради трёх строк.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
