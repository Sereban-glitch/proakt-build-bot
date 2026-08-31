// Package ai — доступ к Anthropic-совместимому LLM-шлюзу.
// Протокол: POST {base}/messages (Messages API, заголовок anthropic-version).
// Две задачи:
//  1. Transcribe: голос (OGG/Opus) → текст. Рабочая форма блока:
//     type:"image" + source.base64 + media_type:"audio/ogg".
//     ⚠⚠⚠ Прокси-конвертер НЕ понимает блоки type:"audio"/"input_audio" —
//     молча выбрасывает их (умеет только text/image/document/tool_use/
//     tool_result/thinking), и модель отвечает «не вижу аудио».
//     Блок "image" конвертируется в Gemini inlineData, а Gemini понимает
//     аудио-маймы нативно. Проверено живым тестом 2026-08-31: TTS-речь через
//     OGG/Opus распознаётся дословно (gemini-2.5-flash и gemini-3-flash).
//  2. ExtractPositions: текст (в т.ч. числа словами) → позиции акта в JSON.
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"proakt/internal/domain"
)

const transcriptPrompt = "Расшифруй эту голосовую запись дословно на языке записи. Верни только текст без пояснений."

const extractPrompt = `Ты — парсер строительных смет. Из текста подрядчика извлеки позиции акта выполненных работ.
Верни ТОЛЬКО JSON-массив без пояснений и без markdown-разметки.
Элемент массива — объект одного из двух видов:
{"name":"<наименование>","qty":<число>,"unit":"<единица: м2, м.п., шт, кг или пусто>","price":<цена за единицу>}
или, если названа только готовая сумма:
{"name":"<наименование>","sum":<сумма>}
Правила: числа пиши цифрами («сорок пять» → 45, «двести шестьдесят» → 260);
«метров квадратных/квадрат» → unit "м2"; «погонных» → unit "м.п.";
лишние реплики и приветствия игнорируй; если позиций нет — верни [].`

// Gateway — клиент шлюза.
type Gateway struct {
	base     string       // например http://127.0.0.1:18080/v1
	key      string       // опциональный ключ (с localhost обычно не нужен)
	model    string       // модель для текста, например gemini-3-flash
	modelASR string       // модель для распознавания голоса, например gemini-2.5-flash
	httpS    *http.Client // текстовые запросы
	httpL    *http.Client // аудио-запросы (могут быть долгими)
}

// New — конструктор (model "" → gemini-3-flash; modelASR "" → gemini-2.5-flash).
func New(base, key, model, modelASR string) *Gateway {
	if model == "" {
		model = "gemini-3-flash"
	}
	if modelASR == "" {
		modelASR = "gemini-2.5-flash"
	}
	return &Gateway{
		base:  strings.TrimRight(base, "/"),
		key:   key,
		model: model,
		httpS: &http.Client{Timeout: 45 * time.Second},
		httpL: &http.Client{Timeout: 120 * time.Second},
	}
}

func (g *Gateway) auth(req *http.Request) {
	req.Header.Set("anthropic-version", "2023-06-01")
	if g.key != "" {
		req.Header.Set("x-api-key", g.key)
		req.Header.Set("Authorization", "Bearer "+g.key)
	}
}

// Transcribe — голос → текст. Блок «image»+media_type audio/ogg:
// прокси превращает его в Gemini inlineData, и модель получает настоящее аудио.
// Если модель всё-таки ответила «не вижу аудио» (форму снова сменили на шлюзе) —
// считаем это ошибкой, а не транскриптом.
func (g *Gateway) Transcribe(ctx context.Context, ogg []byte) (string, error) {
	if len(ogg) == 0 {
		return "", errors.New("ai: пустое аудио")
	}
	b64 := base64.StdEncoding.EncodeToString(ogg)
	text := map[string]string{"type": "text", "text": transcriptPrompt}
	audio := map[string]any{
		"type": "image",
		"source": map[string]string{
			"type":       "base64",
			"media_type": "audio/ogg",
			"data":       b64,
		},
	}
	model := g.modelASR
	if model == "" {
		model = g.model
	}
	txt, err := g.message(ctx, model, "", []any{text, audio}, g.httpL, 8192)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(txt) == "" {
		return "", errors.New("ai: пустая расшифровка")
	}
	if looksLikeRefusal(txt) {
		log.Printf("ai: модель не получила аудио (ответ-отказ): %.200s", txt)
		return "", errors.New("ai: аудио не дошло до модели (ответ-отказ)")
	}
	return strings.TrimSpace(txt), nil
}

// looksLikeRefusal — ответ-отказ вместо транскрипта («не вижу аудиофайла» и т.п.).
// Ложных срабатываний на строительных позициях практически не бывает.
func looksLikeRefusal(s string) bool {
	s = strings.ToLower(s)
	hasNo := strings.Contains(s, "не вижу") || strings.Contains(s, "нет аудио") ||
		strings.Contains(s, "нет голосов") || strings.Contains(s, "не могу расшифр")
	markers := []string{"аудио", "аудиофайл", "голосов", "записи", "файл", "attachment", "audio"}
	if hasNo {
		for _, m := range markers {
			if strings.Contains(s, m) {
				return true
			}
		}
	}
	return strings.Contains(s, "загрузите аудио") ||
		(strings.Contains(s, "прикреп") && strings.Contains(s, "аудио")) ||
		strings.Contains(s, "no audio") ||
		(strings.Contains(s, "upload") && strings.Contains(s, "audio"))
}

// ExtractPositions — текст → позиции акта (модель переводит числа словами в цифры).
func (g *Gateway) ExtractPositions(ctx context.Context, text string) ([]domain.DraftLine, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("ai: пустой текст")
	}
	raw, err := g.message(ctx, g.model, extractPrompt, text, g.httpS, 2048)
	if err != nil {
		return nil, err
	}
	return ParsePositionsJSON(raw)
}

// message — POST {base}/messages (Anthropic Messages API).
// content — строка или массив блоков user-сообщения; system — системный промт.
func (g *Gateway) message(ctx context.Context, model, system string, content any, hc *http.Client, maxTokens int) (string, error) {
	if model == "" {
		model = g.model
	}
	payload := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages": []any{
			map[string]any{"role": "user", "content": content},
		},
		"temperature": 0,
	}
	if system != "" {
		payload["system"] = system
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.base+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	g.auth(req)
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("messages http %d: %.200s", resp.StatusCode, raw)
	}
	var out struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("разбор ответа: %w", err)
	}
	if len(out.Content) == 0 {
		return "", errors.New("ai: пустой ответ (нет content)")
	}
	return ContentText(out.Content), nil
}

// ContentText — content ответа бывает строкой или массивом блоков
// (блоки thinking без поля text молча пропускаются).
func ContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var sb strings.Builder
		for _, p := range parts {
			if p.Text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(p.Text)
			}
		}
		return strings.TrimSpace(sb.String())
	}
	return ""
}

// ParsePositionsJSON — разбор ответа модели: массив, возможно в ```json …```
// или с лишним текстом вокруг. Дополняет вычислимые поля (sum, qty, price).
func ParsePositionsJSON(s string) ([]domain.DraftLine, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("ai: пустой ответ")
	}
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		s = strings.TrimPrefix(strings.TrimSpace(s), "json")
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	start, end := strings.Index(s, "["), strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil, errors.New("ai: JSON-массив не найден")
	}
	var items []struct {
		Name  string  `json:"name"`
		Qty   float64 `json:"qty"`
		Unit  string  `json:"unit"`
		Price float64 `json:"price"`
		Sum   float64 `json:"sum"`
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &items); err != nil {
		return nil, fmt.Errorf("ai: разбор JSON: %w", err)
	}
	var lines []domain.DraftLine
	for _, it := range items {
		name := strings.TrimSpace(it.Name)
		if name == "" {
			continue
		}
		l := domain.DraftLine{Name: name, Qty: it.Qty, Unit: strings.TrimSpace(it.Unit), Price: it.Price, Sum: it.Sum}
		if l.Sum == 0 && l.Qty > 0 && l.Price > 0 {
			l.Sum = l.Qty * l.Price
		}
		if l.Qty == 0 && l.Sum > 0 {
			l.Qty, l.Price = 1, l.Sum
		}
		if l.Sum <= 0 || l.Price < 0 {
			continue
		}
		lines = append(lines, l)
	}
	return lines, nil
}
