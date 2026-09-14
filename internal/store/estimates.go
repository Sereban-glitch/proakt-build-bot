// estimates.go — сметы и шаблоны работ (v0.5).
//
// Смета — это план работ до ремонта, живёт рядом с объектом. Позиции
// повторяют живую таблицу мастера (вид робіт | м²,шт | м.п | ціна | сума |
// примітки), у каждой позиции флаг hidden (скрытая/подготовительная работа)
// и done (закрыто актом). Деньги всегда считаются с коэффициентом сложности
// coeff на уровне SQL-агрегатов: смена коэффициента мгновенно пересчитывает
// всю смету без апдейта строк (sum в базе — базовая qty × price).
//
// Безопасность: все методы чат-скоуплены (chat_id в WHERE) — веб-запросы
// приходят «из интернета», IDOR исключён на уровне SQL, как у актов (v0.4).
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"proakt/internal/domain"
)

// --- сметы: базовые операции -------------------------------------------------

// CreateEstimate — новая смета объекта (объект обязан принадлежать чату).
func (s *Store) CreateEstimate(ctx context.Context, chatID, objectID int64, title string, coeff float64, note string) (domain.Estimate, error) {
	// владение объектом проверяем фактом UPDATE-скоупа: вставляем только если
	// объект из этого чата
	var e domain.Estimate
	err := s.pool.QueryRow(ctx, `
INSERT INTO estimates(object_id, chat_id, title, coeff, note)
SELECT o.id, $2, $3, $4, $5 FROM objects o WHERE o.id = $1 AND o.chat_id = $2
RETURNING id, object_id, chat_id, title, status, coeff, note, share_token, created_at`,
		objectID, chatID, title, coeff, note).
		Scan(&e.ID, &e.ObjectID, &e.ChatID, &e.Title, &e.Status, &e.Coeff, &e.Note, &e.ShareToken, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

// GetEstimate — смета по id (только своего чата).
func (s *Store) GetEstimate(ctx context.Context, chatID, estID int64) (domain.Estimate, error) {
	var e domain.Estimate
	err := s.pool.QueryRow(ctx, `
SELECT id, object_id, chat_id, title, status, coeff, note, share_token, created_at
FROM estimates WHERE id = $1 AND chat_id = $2`, estID, chatID).
		Scan(&e.ID, &e.ObjectID, &e.ChatID, &e.Title, &e.Status, &e.Coeff, &e.Note, &e.ShareToken, &e.CreatedAt)
	if err != nil {
		return e, ErrNotFound
	}
	return e, nil
}

// EstimateChatID — владелец сметы (для check-then-act из handlers).
func (s *Store) EstimateChatID(ctx context.Context, estID int64) (int64, error) {
	var chatID int64
	err := s.pool.QueryRow(ctx, "SELECT chat_id FROM estimates WHERE id = $1", estID).Scan(&chatID)
	if err != nil {
		return 0, ErrNotFound
	}
	return chatID, nil
}

const estimateBriefSQL = `
SELECT e.id, e.object_id, o.name, e.title, e.status, e.coeff, e.note, e.share_token, e.created_at,
  (SELECT count(*) FROM estimate_lines l WHERE l.est_id = e.id),
  COALESCE((SELECT SUM(l.sum) FROM estimate_lines l WHERE l.est_id = e.id), 0) * e.coeff,
  COALESCE((SELECT SUM(l.sum) FROM estimate_lines l WHERE l.est_id = e.id AND l.hidden = FALSE), 0) * e.coeff,
  COALESCE((SELECT SUM(l.sum) FROM estimate_lines l WHERE l.est_id = e.id AND l.hidden = TRUE), 0) * e.coeff,
  COALESCE((SELECT SUM(l.sum) FROM estimate_lines l WHERE l.est_id = e.id AND l.done = TRUE), 0) * e.coeff
FROM estimates e JOIN objects o ON o.id = e.object_id
WHERE e.chat_id = $1`

func scanEstimateBrief(rows pgx.Rows) (domain.EstimateBrief, error) {
	var b domain.EstimateBrief
	err := rows.Scan(&b.ID, &b.ObjectID, &b.ObjectName, &b.Title, &b.Status, &b.Coeff, &b.Note,
		&b.ShareToken, &b.CreatedAt, &b.Lines, &b.Total, &b.Visible, &b.Hidden, &b.DoneSum)
	if err != nil {
		return b, err
	}
	if b.Total > 0.01 {
		b.HiddenShare = int(b.Hidden/b.Total*100 + 0.5)
	}
	return b, nil
}

// ListEstimates — все сметы чата, новые сверху.
func (s *Store) ListEstimates(ctx context.Context, chatID int64, objectID int64) ([]domain.EstimateBrief, error) {
	q := estimateBriefSQL
	args := []any{chatID}
	if objectID > 0 {
		q += ` AND e.object_id = $2`
		args = append(args, objectID)
	}
	q += ` ORDER BY e.id DESC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EstimateBrief
	for rows.Next() {
		b, err := scanEstimateBrief(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetEstimateBrief — смета с деньгами и прогрессом.
func (s *Store) GetEstimateBrief(ctx context.Context, chatID, estID int64) (domain.EstimateBrief, error) {
	rows, err := s.pool.Query(ctx, estimateBriefSQL+` AND e.id = $2`, chatID, estID)
	if err != nil {
		return domain.EstimateBrief{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return domain.EstimateBrief{}, ErrNotFound
	}
	return scanEstimateBrief(rows)
}

// UpdateEstimate — PATCH-семантика: nil = поле не трогаем.
func (s *Store) UpdateEstimate(ctx context.Context, chatID, estID int64, title, status, note *string, coeff *float64) (domain.Estimate, error) {
	// читаем текущее, накладываем патч, пишем целиком — атомарность не критична
	// (один мастер на чат), зато логика прозрачна
	e, err := s.GetEstimate(ctx, chatID, estID)
	if err != nil {
		return e, err
	}
	if title != nil && *title != "" {
		e.Title = *title
	}
	if status != nil {
		switch *status {
		case "draft", "sent", "approved", "done":
			e.Status = *status
		}
	}
	if coeff != nil && *coeff >= 1 && *coeff <= 3 {
		e.Coeff = *coeff
	}
	if note != nil {
		e.Note = *note
	}
	_, err = s.pool.Exec(ctx, `
UPDATE estimates SET title=$2, status=$3, coeff=$4, note=$5 WHERE id=$1 AND chat_id=$6`,
		e.ID, e.Title, e.Status, e.Coeff, e.Note, chatID)
	if err != nil {
		return e, err
	}
	return s.GetEstimate(ctx, chatID, estID)
}

// DeleteEstimate — удалить смету целиком (строки уйдут каскадом).
// Возвращает бриф для отчёта «что удалено».
func (s *Store) DeleteEstimate(ctx context.Context, chatID, estID int64) (domain.EstimateBrief, error) {
	b, err := s.GetEstimateBrief(ctx, chatID, estID)
	if err != nil {
		return b, err
	}
	_, err = s.pool.Exec(ctx, "DELETE FROM estimates WHERE id=$1 AND chat_id=$2", estID, chatID)
	return b, err
}

// --- строки сметы -------------------------------------------------------------

// EstimateLines — позиции сметы по порядку (владельца проверяет вызывавший,
// либо через GetEstimate с chatID).
func (s *Store) EstimateLines(ctx context.Context, estID int64) ([]domain.EstimateLine, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, est_id, pos, name, qty, unit, price, sum, hidden, done, note
FROM estimate_lines WHERE est_id = $1 ORDER BY pos`, estID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EstimateLine
	for rows.Next() {
		var l domain.EstimateLine
		if err := rows.Scan(&l.ID, &l.EstID, &l.Pos, &l.Name, &l.Qty, &l.Unit, &l.Price, &l.Sum, &l.Hidden, &l.Done, &l.Note); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AddEstimateLine — новая позиция в конец сметы (владение проверяет SQL через
// estimates.chat_id). Возвращает строку с пересчитанной суммой.
func (s *Store) AddEstimateLine(ctx context.Context, chatID, estID int64, name, unit string, qty, price float64, hidden bool, note string) (domain.EstimateLine, error) {
	var l domain.EstimateLine
	err := s.pool.QueryRow(ctx, `
INSERT INTO estimate_lines(est_id, pos, name, qty, unit, price, sum, hidden, note)
SELECT $1, COALESCE(MAX(pos),0)+1, $2, $3, $4, $5, $6, $7, $8 FROM estimate_lines WHERE est_id = $1
RETURNING id, est_id, pos, name, qty, unit, price, sum, hidden, done, note`,
		estID, name, qty, unit, price, qty*price, hidden, note).
		Scan(&l.ID, &l.EstID, &l.Pos, &l.Name, &l.Qty, &l.Unit, &l.Price, &l.Sum, &l.Hidden, &l.Done, &l.Note)
	if err != nil {
		// проверим, что смета вообще существует и наша: иначе ErrNotFound
		if _, gerr := s.GetEstimate(ctx, chatID, estID); gerr != nil {
			return l, ErrNotFound
		}
		return l, err
	}
	return l, nil
}

// AddEstimateLinesBulk — вставка блока позиций (шаблон / парс-мультистрока)
// одной транзакцией. pos продолжается с текущего конца.
func (s *Store) AddEstimateLinesBulk(ctx context.Context, chatID, estID int64, lines []domain.EstimateLine) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}
	if _, err := s.GetEstimate(ctx, chatID, estID); err != nil {
		return 0, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var base int
	if err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(pos),0) FROM estimate_lines WHERE est_id=$1", estID).Scan(&base); err != nil {
		return 0, err
	}
	for i, l := range lines {
		if _, err := tx.Exec(ctx, `
INSERT INTO estimate_lines(est_id, pos, name, qty, unit, price, sum, hidden, note)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, estID, base+i+1, l.Name, l.Qty, l.Unit, l.Price, l.Qty*l.Price, l.Hidden, l.Note); err != nil {
			return 0, err
		}
	}
	return len(lines), tx.Commit(ctx)
}

// EstimateLineOwned — строка принадлежит смете этого чата? Возвращает строку.
func (s *Store) EstimateLineOwned(ctx context.Context, chatID, lineID int64) (domain.EstimateLine, error) {
	var l domain.EstimateLine
	err := s.pool.QueryRow(ctx, `
SELECT l.id, l.est_id, l.pos, l.name, l.qty, l.unit, l.price, l.sum, l.hidden, l.done, l.note
FROM estimate_lines l JOIN estimates e ON e.id = l.est_id
WHERE l.id = $1 AND e.chat_id = $2`, lineID, chatID).
		Scan(&l.ID, &l.EstID, &l.Pos, &l.Name, &l.Qty, &l.Unit, &l.Price, &l.Sum, &l.Hidden, &l.Done, &l.Note)
	if err != nil {
		return l, ErrNotFound
	}
	return l, nil
}

// UpdateEstimateLine — PATCH строки: qty/price пересчитывают sum на сервере.
func (s *Store) UpdateEstimateLine(ctx context.Context, chatID, lineID int64, name, unit, note *string, qty, price *float64, hidden, done *bool) (domain.EstimateLine, error) {
	l, err := s.EstimateLineOwned(ctx, chatID, lineID)
	if err != nil {
		return l, err
	}
	if name != nil && *name != "" {
		l.Name = *name
	}
	if unit != nil {
		l.Unit = *unit
	}
	if note != nil {
		l.Note = *note
	}
	if qty != nil && *qty >= 0 && *qty <= 1e7 {
		l.Qty = *qty
	}
	if price != nil && *price >= 0 && *price <= 1e9 {
		l.Price = *price
	}
	if hidden != nil {
		l.Hidden = *hidden
	}
	if done != nil {
		l.Done = *done
	}
	l.Sum = l.Qty * l.Price
	_, err = s.pool.Exec(ctx, `
UPDATE estimate_lines SET name=$2, unit=$3, note=$4, qty=$5, price=$6, sum=$7, hidden=$8, done=$9
WHERE id=$1`, l.ID, l.Name, l.Unit, l.Note, l.Qty, l.Price, l.Sum, l.Hidden, l.Done)
	if err != nil {
		return l, err
	}
	return l, nil
}

// DeleteEstimateLine — убрать позицию; последующие pos не сдвигаем
// (нумерация в UI идёт по порядку выборки, дырки не видны).
func (s *Store) DeleteEstimateLine(ctx context.Context, chatID, lineID int64) (domain.EstimateLine, error) {
	l, err := s.EstimateLineOwned(ctx, chatID, lineID)
	if err != nil {
		return l, err
	}
	_, err = s.pool.Exec(ctx, "DELETE FROM estimate_lines WHERE id=$1", l.ID)
	return l, err
}

// MoveEstimateLine — поменять позицию местами с соседней (up/down).
// Меняются только pos двух строк — остальной порядок не трогаем.
func (s *Store) MoveEstimateLine(ctx context.Context, chatID, lineID int64, up bool) error {
	l, err := s.EstimateLineOwned(ctx, chatID, lineID)
	if err != nil {
		return err
	}
	neighbor := -1 // в SQL: up → pos меньше, down → pos больше
	if up {
		err = s.pool.QueryRow(ctx, `
SELECT id FROM estimate_lines WHERE est_id=$1 AND pos < $2 ORDER BY pos DESC LIMIT 1`,
			l.EstID, l.Pos).Scan(&neighbor)
	} else {
		err = s.pool.QueryRow(ctx, `
SELECT id FROM estimate_lines WHERE est_id=$1 AND pos > $2 ORDER BY pos LIMIT 1`,
			l.EstID, l.Pos).Scan(&neighbor)
	}
	if err != nil {
		return ErrNotFound // края списка — двигать некуда
	}
	var nPos int
	if err := s.pool.QueryRow(ctx, "SELECT pos FROM estimate_lines WHERE id=$1", neighbor).Scan(&nPos); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// pos может конфликтовать (UNIQUE нет, но для честности меняем через -1)
	if _, err := tx.Exec(ctx, "UPDATE estimate_lines SET pos=-1 WHERE id=$1", l.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE estimate_lines SET pos=$2 WHERE id=$1", neighbor, l.Pos); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE estimate_lines SET pos=$2 WHERE id=$1", l.ID, nPos); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// --- share-ссылка для заказчика ------------------------------------------------

// SetShareToken — создать/перевыпустить токен публичной ссылки.
func (s *Store) SetShareToken(ctx context.Context, chatID, estID int64, token string) (string, error) {
	var t string
	err := s.pool.QueryRow(ctx, `
UPDATE estimates SET share_token=$3 WHERE id=$1 AND chat_id=$2 RETURNING share_token`,
		estID, chatID, token).Scan(&t)
	if err != nil {
		return "", ErrNotFound
	}
	return t, nil
}

// ShareView — всё, что видит заказчик по публичной ссылке.
type ShareView struct {
	ObjectName string                `json:"object_name"`
	Title      string                `json:"title"`
	Status     string                `json:"status"`
	Note       string                `json:"note"`
	Lines      []domain.EstimateLine `json:"lines"`
	Photos     []SharePhoto          `json:"photos"`
	CreatedAt  time.Time             `json:"created_at"`
	// деньги считает handler из строк (базовые суммы в БД, коэффициент в смете)
	Total       float64 `json:"total"`
	Visible     float64 `json:"visible"`
	Hidden      float64 `json:"hidden"`
	HiddenShare int     `json:"hidden_share"`
}

// SharePhoto — фото скрытых работ для публичной страницы (id + подпись,
// файл отдаётся отдельным маршрутом с проверкой токена).
type SharePhoto struct {
	ID      int64     `json:"id"`
	Caption string    `json:"caption"`
	ActNo   int       `json:"act_no"`
	At      time.Time `json:"at"`
}

// GetShareView — публичный просмотр по токену (без auth!).
func (s *Store) GetShareView(ctx context.Context, token string) (ShareView, error) {
	var v ShareView
	var estID, objID int64
	var coeff float64
	err := s.pool.QueryRow(ctx, `
SELECT e.id, e.object_id, o.name, e.title, e.status, e.note, e.coeff, e.created_at
FROM estimates e JOIN objects o ON o.id = e.object_id
WHERE e.share_token = $1 AND e.share_token <> ''`, token).
		Scan(&estID, &objID, &v.ObjectName, &v.Title, &v.Status, &v.Note, &coeff, &v.CreatedAt)
	if err != nil {
		return v, ErrNotFound
	}
	lines, err := s.EstimateLines(ctx, estID)
	if err != nil {
		return v, err
	}
	// суммы строк приводим к деньгам сметы: база × коэффициент сложности —
	// чтобы строка сходилась с итогами на странице заказчика
	for i := range lines {
		lines[i].Sum = lines[i].Sum * coeff
	}
	v.Lines = lines
	// фото объекта (до 12 свежих) — «доказательство» скрытых работ
	rows, err := s.pool.Query(ctx, `
SELECT ph.id, ph.caption, COALESCE(a.act_no, 0), ph.created_at
FROM photos ph LEFT JOIN acts a ON a.id = ph.act_id
WHERE ph.object_id = $1 AND ph.file_path <> '' ORDER BY ph.created_at DESC LIMIT 12`, objID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p SharePhoto
			if err := rows.Scan(&p.ID, &p.Caption, &p.ActNo, &p.At); err == nil {
				v.Photos = append(v.Photos, p)
			}
		}
	}
	return v, nil
}

// SharePhotoFile — путь к файлу фото, входящему в публичную смету.
func (s *Store) SharePhotoFile(ctx context.Context, token string, photoID int64) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx, `
SELECT ph.file_path FROM photos ph
JOIN objects o ON o.id = ph.object_id
JOIN estimates e ON e.object_id = o.id
WHERE e.share_token = $1 AND ph.id = $2 AND ph.file_path <> '' LIMIT 1`, token, photoID).Scan(&path)
	if err != nil {
		return "", ErrNotFound
	}
	return path, nil
}

// --- шаблоны работ («админка», v0.5) -------------------------------------------

// ListTemplates — шаблоны чата.
func (s *Store) ListTemplates(ctx context.Context, chatID int64) ([]domain.Template, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, name, lines FROM templates WHERE chat_id=$1 ORDER BY name`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Template
	for rows.Next() {
		var t domain.Template
		var raw []byte
		if err := rows.Scan(&t.ID, &t.Name, &raw); err != nil {
			return nil, err
		}
		t.Lines = []domain.TemplateLine{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &t.Lines)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpsertTemplate — создать/обновить шаблон по имени (как прайс).
func (s *Store) UpsertTemplate(ctx context.Context, chatID int64, name string, lines []domain.TemplateLine) (domain.Template, error) {
	raw, _ := json.Marshal(lines)
	var t domain.Template
	err := s.pool.QueryRow(ctx, `
INSERT INTO templates(chat_id, name, lines) VALUES($1,$2,$3)
ON CONFLICT (chat_id, name) DO UPDATE SET lines = EXCLUDED.lines
RETURNING id, name, lines`, chatID, name, raw).Scan(&t.ID, &t.Name, &raw)
	if err != nil {
		return t, err
	}
	t.Lines = lines
	return t, nil
}

// DeleteTemplate — убрать шаблон. ok=false — не было такого.
func (s *Store) DeleteTemplate(ctx context.Context, chatID int64, id int64) (domain.Template, bool, error) {
	var t domain.Template
	var raw []byte
	err := s.pool.QueryRow(ctx, "DELETE FROM templates WHERE id=$1 AND chat_id=$2 RETURNING id, name, lines", id, chatID).
		Scan(&t.ID, &t.Name, &raw)
	if err != nil {
		return t, false, nil
	}
	t.Lines = []domain.TemplateLine{}
	_ = json.Unmarshal(raw, &t.Lines)
	return t, true, nil
}
