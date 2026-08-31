package tg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client — клиент Bot API через long polling (порты не занимаем).
type Client struct {
	token string
	api   string
	http  *http.Client
	httpL *http.Client // для долгого опроса
}

func New(token string) *Client {
	return &Client{
		token: token,
		api:   "https://api.telegram.org/bot" + token,
		http:  &http.Client{Timeout: 20 * time.Second},
		httpL: &http.Client{Timeout: 40 * time.Second}, // getUpdates с timeout=25
	}
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

// call — POST JSON-запрос к API.
func (c *Client) call(ctx context.Context, client *http.Client, method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post %s: %w", method, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	var r apiResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("decode %s: %w (http %d)", method, err, resp.StatusCode)
	}
	if !r.OK {
		e := &APIError{Code: r.ErrorCode, Description: r.Description}
		if r.Parameters != nil {
			e.RetryAfter = r.Parameters.RetryAfter
		}
		return e
	}
	if result != nil && len(r.Result) > 0 {
		return json.Unmarshal(r.Result, result)
	}
	return nil
}

// GetUpdates — длинный опрос обновлений.
func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSec int) ([]Update, error) {
	payload := map[string]any{
		"offset":          offset,
		"timeout":         timeoutSec,
		"allowed_updates": []string{"message", "callback_query"},
	}
	var ups []Update
	if err := c.call(ctx, c.httpL, "getUpdates", payload, &ups); err != nil {
		return nil, err
	}
	return ups, nil
}

// SendMessage — отправить текст (markup — любая клавиатура или nil).
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, markup any) error {
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	return c.call(ctx, c.http, "sendMessage", payload, nil)
}

// SendChatAction — индикатор «печатает…».
func (c *Client) SendChatAction(ctx context.Context, chatID int64) error {
	return c.call(ctx, c.http, "sendChatAction", map[string]any{"chat_id": chatID, "action": "typing"}, nil)
}

// AnswerCallbackQuery — подтвердить нажатие кнопки.
func (c *Client) AnswerCallbackQuery(ctx context.Context, id, text string) error {
	return c.call(ctx, c.http, "answerCallbackQuery", map[string]any{"callback_query_id": id, "text": text}, nil)
}

// SetCommands — зарегистрировать меню команд.
func (c *Client) SetCommands(ctx context.Context, cmds []BotCommand) error {
	return c.call(ctx, c.http, "setMyCommands", map[string]any{"commands": cmds}, nil)
}

// DeleteWebhook — гарантировать режим long polling.
func (c *Client) DeleteWebhook(ctx context.Context) error {
	return c.call(ctx, c.http, "deleteWebhook", map[string]any{}, nil)
}

// GetMe — проверить токен (перед стартом).
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	var me User
	if err := c.call(ctx, c.http, "getMe", map[string]any{}, &me); err != nil {
		return nil, err
	}
	return &me, nil
}

// SendPhoto — отправить фото по file_id (фото уже хранится в Telegram, v0.3.7:
// демонстрация скрытых работ заказчику без повторной загрузки).
func (c *Client) SendPhoto(ctx context.Context, chatID int64, fileID, caption string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"photo":   fileID,
	}
	if caption != "" {
		payload["caption"] = caption
	}
	return c.call(ctx, c.http, "sendPhoto", payload, nil)
}

// SendDocumentFile — отправить файл (XLSX-акт) с подписью.
func (c *Client) SendDocumentFile(ctx context.Context, chatID int64, path, caption string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("chat_id", fmt.Sprintf("%d", chatID))
	if caption != "" {
		_ = w.WriteField("caption", caption)
	}
	part, err := w.CreateFormFile("document", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api+"/sendDocument", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r apiResponse
	if err := json.Unmarshal(raw, &r); err != nil || !r.OK {
		return fmt.Errorf("sendDocument: http %d %s", resp.StatusCode, r.Description)
	}
	return nil
}

// DownloadFileBytes — скачать файл по file_id в память (голосовые OGG).
func (c *Client) DownloadFileBytes(ctx context.Context, fileID string, limit int64) ([]byte, error) {
	var res struct {
		FilePath string `json:"file_path"`
	}
	if err := c.call(ctx, c.http, "getFile", map[string]any{"file_id": fileID}, &res); err != nil {
		return nil, err
	}
	if res.FilePath == "" {
		return nil, fmt.Errorf("file_path пуст")
	}
	url := "https://api.telegram.org/file/bot" + c.token + "/" + res.FilePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

// DownloadFile — скачать файл по file_id (фото) в destPath.
func (c *Client) DownloadFile(ctx context.Context, fileID, destPath string) error {
	var res struct {
		FilePath string `json:"file_path"`
	}
	if err := c.call(ctx, c.http, "getFile", map[string]any{"file_id": fileID}, &res); err != nil {
		return err
	}
	if res.FilePath == "" {
		return fmt.Errorf("file_path пуст")
	}
	url := "https://api.telegram.org/file/bot" + c.token + "/" + res.FilePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: http %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, io.LimitReader(resp.Body, 25<<20)) // ограничение 25 МБ
	return err
}
