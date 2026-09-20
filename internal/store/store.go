// Package store — доступ к PostgreSQL (pgx, пул ограничен — экономия RAM).
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"proakt/internal/domain"
)

var ErrNotFound = errors.New("не найдено")

const schema = `
CREATE TABLE IF NOT EXISTS users (
  chat_id    BIGINT PRIMARY KEY,
  tg_user_id BIGINT NOT NULL,
  name       TEXT NOT NULL DEFAULT '',
  username   TEXT NOT NULL DEFAULT '',
  is_admin   BOOLEAN NOT NULL DEFAULT FALSE,
  state      TEXT NOT NULL DEFAULT 'idle',
  state_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS objects (
  id         BIGSERIAL PRIMARY KEY,
  chat_id    BIGINT NOT NULL,
  name       TEXT NOT NULL,
  customer   TEXT NOT NULL DEFAULT '',
  status     TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_objects_chat ON objects(chat_id);
CREATE TABLE IF NOT EXISTS acts (
  id         BIGSERIAL PRIMARY KEY,
  object_id  BIGINT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
  act_no     INT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'confirmed',
  date       DATE NOT NULL DEFAULT CURRENT_DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(object_id, act_no)
);
CREATE TABLE IF NOT EXISTS act_lines (
  id      BIGSERIAL PRIMARY KEY,
  act_id  BIGINT NOT NULL REFERENCES acts(id) ON DELETE CASCADE,
  pos     INT NOT NULL,
  name    TEXT NOT NULL,
  qty     NUMERIC(12,2) NOT NULL,
  unit    TEXT NOT NULL DEFAULT '',
  price   NUMERIC(12,2) NOT NULL,
  sum     NUMERIC(12,2) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lines_act ON act_lines(act_id);
CREATE TABLE IF NOT EXISTS payments (
  id         BIGSERIAL PRIMARY KEY,
  act_id     BIGINT NOT NULL REFERENCES acts(id) ON DELETE CASCADE,
  amount     NUMERIC(12,2) NOT NULL,
  note       TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pay_act ON payments(act_id);
CREATE TABLE IF NOT EXISTS photos (
  id         BIGSERIAL PRIMARY KEY,
  act_id     BIGINT REFERENCES acts(id) ON DELETE SET NULL,
  object_id  BIGINT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
  file_id    TEXT NOT NULL,
  file_path  TEXT NOT NULL DEFAULT '',
  caption    TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_photos_obj ON photos(object_id);
CREATE TABLE IF NOT EXISTS catalog_items (
  id       BIGSERIAL PRIMARY KEY,
  chat_id  BIGINT NOT NULL,
  name     TEXT NOT NULL,
  unit     TEXT NOT NULL DEFAULT '',
  price    NUMERIC(12,2) NOT NULL,
  UNIQUE(chat_id, name)
);
CREATE TABLE IF NOT EXISTS estimates (
  id          BIGSERIAL PRIMARY KEY,
  object_id   BIGINT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
  chat_id     BIGINT NOT NULL,
  title       TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'draft',
  coeff       NUMERIC(4,2) NOT NULL DEFAULT 1.00,
  note        TEXT NOT NULL DEFAULT '',
  share_token TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_estimates_chat ON estimates(chat_id);
CREATE INDEX IF NOT EXISTS idx_estimates_obj ON estimates(object_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_estimates_share ON estimates(share_token) WHERE share_token <> '';
CREATE TABLE IF NOT EXISTS estimate_lines (
  id     BIGSERIAL PRIMARY KEY,
  est_id BIGINT NOT NULL REFERENCES estimates(id) ON DELETE CASCADE,
  pos    INT NOT NULL,
  name   TEXT NOT NULL,
  qty    NUMERIC(12,2) NOT NULL DEFAULT 0,
  unit   TEXT NOT NULL DEFAULT '',
  price  NUMERIC(12,2) NOT NULL DEFAULT 0,
  sum    NUMERIC(14,2) NOT NULL DEFAULT 0,
  hidden BOOLEAN NOT NULL DEFAULT FALSE,
  done   BOOLEAN NOT NULL DEFAULT FALSE,
  note   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_est_lines_est ON estimate_lines(est_id);
CREATE TABLE IF NOT EXISTS templates (
  id      BIGSERIAL PRIMARY KEY,
  chat_id BIGINT NOT NULL,
  name    TEXT NOT NULL,
  lines   JSONB NOT NULL DEFAULT '[]'::jsonb,
  UNIQUE(chat_id, name)
);
CREATE TABLE IF NOT EXISTS schema_migrations (
  version    INT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- v0.6: фото, привязанное к строке сметы («снял комнату — снимок к строке
-- "штукатурка 40 м²"»). Nullable + ON DELETE SET NULL: старые фото не трогаем.
ALTER TABLE photos ADD COLUMN IF NOT EXISTS est_line_id BIGINT REFERENCES estimate_lines(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_photos_line ON photos(est_line_id);
-- v0.6: диалог мастера и заказчика прямо на смете.
CREATE TABLE IF NOT EXISTS estimate_comments (
  id         BIGSERIAL PRIMARY KEY,
  est_id     BIGINT NOT NULL REFERENCES estimates(id) ON DELETE CASCADE,
  author     TEXT NOT NULL DEFAULT 'master', -- master | client
  text       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_est_comments_est ON estimate_comments(est_id);
-- v0.6: источник прайса (Google Таблица) — одна ссылка на чат.
CREATE TABLE IF NOT EXISTS price_sources (
  chat_id    BIGINT PRIMARY KEY,
  url        TEXT NOT NULL,
  file_id    TEXT NOT NULL DEFAULT '',
  gid        TEXT NOT NULL DEFAULT '',
  last_sync  TIMESTAMPTZ NOT NULL DEFAULT 'epoch',
  last_count INT NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT ''
);
-- 360: доступ заказчика (object_id + telegram_user_id).
CREATE TABLE IF NOT EXISTS client_access (
  object_id  BIGINT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
  tg_user_id BIGINT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (object_id, tg_user_id)
);
`

type Store struct{ pool *pgxpool.Pool }

// New открывает пул соединений (экономно: максимум 2 коннекта).
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("dsn: %w", err)
	}
	cfg.MaxConns = 2
	cfg.MinConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Migrate — идемпотентное применение схемы (CREATE/ALTER IF NOT EXISTS —
// можно накатывать на живую базу сколько угодно раз).
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO schema_migrations(version) VALUES (1), (2) ON CONFLICT DO NOTHING`); err != nil {
		return err
	}
	return nil
}

// --- пользователи и FSM -----------------------------------------------------

// EnsureUser регистрирует пользователя; первый в системе получает is_admin.
func (s *Store) EnsureUser(ctx context.Context, chatID, tgUserID int64, name, username string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE chat_id=$1)", chatID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if !exists {
		var cnt int
		if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&cnt); err != nil {
			return false, err
		}
		_, err := s.pool.Exec(ctx,
			"INSERT INTO users(chat_id, tg_user_id, name, username, is_admin) VALUES($1,$2,$3,$4,$5)",
			chatID, tgUserID, name, username, cnt == 0)
		return cnt == 0, err
	}
	_, err = s.pool.Exec(ctx, "UPDATE users SET name=$2, username=$3 WHERE chat_id=$1 AND (name<>$2 OR username<>$3)",
		chatID, name, username)
	return false, err
}

// State возвращает текущее FSM-состояние и данные.
func (s *Store) State(ctx context.Context, chatID int64) (string, map[string]string, error) {
	var st string
	var raw []byte
	err := s.pool.QueryRow(ctx, "SELECT state, state_data FROM users WHERE chat_id=$1", chatID).Scan(&st, &raw)
	if err != nil {
		return "idle", map[string]string{}, err
	}
	data := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &data)
	}
	return st, data, nil
}

// SetState сохраняет FSM-состояние с данными.
func (s *Store) SetState(ctx context.Context, chatID int64, state string, data map[string]string) error {
	raw, _ := json.Marshal(data)
	_, err := s.pool.Exec(ctx, "UPDATE users SET state=$2, state_data=$3 WHERE chat_id=$1", chatID, state, raw)
	return err
}

// --- объекты ----------------------------------------------------------------

func (s *Store) CreateObject(ctx context.Context, chatID int64, name, customer string) (domain.Object, error) {
	var o domain.Object
	err := s.pool.QueryRow(ctx,
		"INSERT INTO objects(chat_id, name, customer) VALUES($1,$2,$3) RETURNING id, chat_id, name, customer, status, created_at",
		chatID, name, customer).Scan(&o.ID, &o.ChatID, &o.Name, &o.Customer, &o.Status, &o.CreatedAt)
	return o, err
}

// ListObjects — объекты чата с итогами по актам (v0.3.5: «предварительный итог»
// по каждому объекту — сколько выполнено и сколько должны).
func (s *Store) ListObjects(ctx context.Context, chatID int64) ([]domain.ObjectBrief, error) {
	rows, err := s.pool.Query(ctx, `
SELECT o.id, o.chat_id, o.name, o.customer, o.status, o.created_at,
  (SELECT count(*) FROM acts a WHERE a.object_id = o.id),
  COALESCE((SELECT SUM(l.sum) FROM act_lines l JOIN acts a ON a.id = l.act_id WHERE a.object_id = o.id), 0),
  COALESCE((SELECT SUM(p.amount) FROM payments p JOIN acts a ON a.id = p.act_id WHERE a.object_id = o.id), 0),
  (SELECT count(*) FROM photos ph WHERE ph.object_id = o.id)
FROM objects o
WHERE o.chat_id = $1 AND o.status = 'active'
ORDER BY o.id`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ObjectBrief
	for rows.Next() {
		var o domain.ObjectBrief
		if err := rows.Scan(&o.ID, &o.ChatID, &o.Name, &o.Customer, &o.Status, &o.CreatedAt, &o.Acts, &o.Total, &o.Paid, &o.Photos); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) GetObject(ctx context.Context, id int64) (domain.Object, error) {
	var o domain.Object
	err := s.pool.QueryRow(ctx,
		"SELECT id, chat_id, name, customer, status, created_at FROM objects WHERE id=$1", id).
		Scan(&o.ID, &o.ChatID, &o.Name, &o.Customer, &o.Status, &o.CreatedAt)
	if err != nil {
		return o, ErrNotFound
	}
	return o, nil
}

// ArchiveObject — убрать объект из меню (status='archived', v0.3.8).
// Мягкое удаление: акты, оплаты и фото остаются в /acts и /report —
// ничего не теряется, объект просто исчезает из списков выбора.
func (s *Store) ArchiveObject(ctx context.Context, chatID, objID int64) (string, error) {
	var name string
	err := s.pool.QueryRow(ctx,
		"UPDATE objects SET status='archived' WHERE id=$1 AND chat_id=$2 RETURNING name", objID, chatID).
		Scan(&name)
	if err != nil {
		return "", ErrNotFound
	}
	return name, nil
}

// --- акты -------------------------------------------------------------------

func (s *Store) CreateAct(ctx context.Context, objectID int64, lines []domain.DraftLine) (domain.Act, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Act{}, err
	}
	defer tx.Rollback(ctx)

	var nextNo int
	if err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(act_no),0)+1 FROM acts WHERE object_id=$1", objectID).Scan(&nextNo); err != nil {
		return domain.Act{}, err
	}
	var a domain.Act
	err = tx.QueryRow(ctx,
		"INSERT INTO acts(object_id, act_no) VALUES($1,$2) RETURNING id, object_id, act_no, status, created_at",
		objectID, nextNo).Scan(&a.ID, &a.ObjectID, &a.ActNo, &a.Status, &a.Date)
	if err != nil {
		return a, err
	}
	for i, l := range lines {
		if _, err := tx.Exec(ctx,
			"INSERT INTO act_lines(act_id, pos, name, qty, unit, price, sum) VALUES($1,$2,$3,$4,$5,$6,$7)",
			a.ID, i+1, l.Name, l.Qty, l.Unit, l.Price, l.Sum); err != nil {
			return a, err
		}
	}
	return a, tx.Commit(ctx)
}

const actBriefSQL = `
SELECT a.id, a.act_no, a.object_id, o.name, o.customer, a.date,
  COALESCE((SELECT SUM(l.sum)   FROM act_lines l WHERE l.act_id = a.id), 0) AS total,
  COALESCE((SELECT SUM(p.amount) FROM payments p WHERE p.act_id = a.id), 0) AS paid,
  (SELECT count(*) FROM photos ph WHERE ph.act_id = a.id) AS photos
FROM acts a JOIN objects o ON o.id = a.object_id
WHERE o.chat_id = $1
ORDER BY a.created_at DESC
LIMIT $2`

func (s *Store) ListActs(ctx context.Context, chatID int64, limit int) ([]domain.ActBrief, error) {
	rows, err := s.pool.Query(ctx, actBriefSQL, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ActBrief
	for rows.Next() {
		var b domain.ActBrief
		var total, paid float64
		if err := rows.Scan(&b.ID, &b.ActNo, &b.ObjectID, &b.ObjectName, &b.Customer, &b.Date, &total, &paid, &b.Photos); err != nil {
			return nil, err
		}
		b.Total, b.Paid = total, paid
		out = append(out, b)
	}
	return out, rows.Err()
}

// ActChatID — чат-владелец акта одним запросом. Нужен Mini App API:
// веб-запросы приходят «из интернета», поэтому перед удалением/оплатой
// сервер обязан проверить, что акт принадлежит именно этому chat_id (v0.4).
func (s *Store) ActChatID(ctx context.Context, actID int64) (int64, error) {
	var chatID int64
	err := s.pool.QueryRow(ctx,
		"SELECT o.chat_id FROM acts a JOIN objects o ON o.id = a.object_id WHERE a.id = $1", actID).
		Scan(&chatID)
	if err != nil {
		return 0, ErrNotFound
	}
	return chatID, nil
}

func (s *Store) GetAct(ctx context.Context, actID int64) (domain.ActBrief, error) {
	var b domain.ActBrief
	var total, paid float64
	err := s.pool.QueryRow(ctx, `
SELECT a.id, a.act_no, a.object_id, o.name, o.customer, a.date,
  COALESCE((SELECT SUM(l.sum)   FROM act_lines l WHERE l.act_id = a.id), 0),
  COALESCE((SELECT SUM(p.amount) FROM payments p WHERE p.act_id = a.id), 0),
  (SELECT count(*) FROM photos ph WHERE ph.act_id = a.id)
FROM acts a JOIN objects o ON o.id = a.object_id WHERE a.id = $1`, actID).
		Scan(&b.ID, &b.ActNo, &b.ObjectID, &b.ObjectName, &b.Customer, &b.Date, &total, &paid, &b.Photos)
	if err != nil {
		return b, ErrNotFound
	}
	b.Total, b.Paid = total, paid
	return b, nil
}

func (s *Store) ActLines(ctx context.Context, actID int64) ([]domain.ActLine, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT pos, name, qty, unit, price, sum FROM act_lines WHERE act_id=$1 ORDER BY pos", actID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ActLine
	for rows.Next() {
		var l domain.ActLine
		if err := rows.Scan(&l.Pos, &l.Name, &l.Qty, &l.Unit, &l.Price, &l.Sum); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// --- оплаты -----------------------------------------------------------------

// GetPayment — оплата по id (только своего чата) с контекстом акта (v0.3.8).
func (s *Store) GetPayment(ctx context.Context, chatID, payID int64) (domain.PaymentRec, error) {
	var p domain.PaymentRec
	err := s.pool.QueryRow(ctx, `
SELECT p.id, p.act_id, p.amount, p.created_at, a.act_no, o.name
FROM payments p JOIN acts a ON a.id = p.act_id JOIN objects o ON o.id = a.object_id
WHERE p.id = $1 AND o.chat_id = $2`, payID, chatID).
		Scan(&p.ID, &p.ActID, &p.Amount, &p.At, &p.ActNo, &p.ObjName)
	if err != nil {
		return p, ErrNotFound
	}
	return p, nil
}

// DeletePayment — убрать ошибочную оплату (v0.3.8: «записал 100000 вместо 10000» —
// долг в отчёте тут же вернётся к правде). Возвращает запись для отчёта.
func (s *Store) DeletePayment(ctx context.Context, chatID, payID int64) (domain.PaymentRec, error) {
	p, err := s.GetPayment(ctx, chatID, payID)
	if err != nil {
		return p, err
	}
	if _, err := s.pool.Exec(ctx, "DELETE FROM payments WHERE id=$1 AND act_id=$2", p.ID, p.ActID); err != nil {
		return p, err
	}
	return p, nil
}

// DeleteAct — удалить акт (v0.3.6): позиции и оплаты уходят каскадом
// (ON DELETE CASCADE), фото отвязываются от акта и остаются у объекта
// (ON DELETE SET NULL). Возвращает бриф удалённого акта для отчёта.
func (s *Store) DeleteAct(ctx context.Context, actID int64) (domain.ActBrief, error) {
	brief, err := s.GetAct(ctx, actID)
	if err != nil {
		return brief, err
	}
	if _, err := s.pool.Exec(ctx, "DELETE FROM acts WHERE id=$1", actID); err != nil {
		return brief, err
	}
	return brief, nil
}

func (s *Store) CreatePayment(ctx context.Context, actID int64, amount float64, note string) error {
	_, err := s.pool.Exec(ctx, "INSERT INTO payments(act_id, amount, note) VALUES($1,$2,$3)", actID, amount, note)
	return err
}

func (s *Store) Payments(ctx context.Context, actID int64) ([]domain.Payment, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT id, act_id, amount, note, created_at FROM payments WHERE act_id=$1 ORDER BY id", actID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.ActID, &p.Amount, &p.Note, &p.At); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- фото -------------------------------------------------------------------

// AddPhoto — запись фото (actID/estLineID могут быть nil: привязка к объекту).
// estLineID — фото строки сметы (v0.6: «фото цепляется к последней строке»).
func (s *Store) AddPhoto(ctx context.Context, actID *int64, objectID int64, fileID, filePath, caption string) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO photos(act_id, object_id, file_id, file_path, caption) VALUES($1,$2,$3,$4,$5)",
		actID, objectID, fileID, filePath, caption)
	return err
}

// AddEstLinePhoto — фото с привязкой к строке сметы (v0.6). Объект выводится
// из строки, так что фото всегда живёт в правильном фотоотчёте объекта.
func (s *Store) AddEstLinePhoto(ctx context.Context, chatID, estID, lineID int64, fileID, filePath, caption string) error {
	tag, err := s.pool.Exec(ctx, `
INSERT INTO photos(act_id, object_id, est_line_id, file_id, file_path, caption)
SELECT NULL, e.object_id, $3, $4, $5, $6
FROM estimates e WHERE e.id = $1 AND e.chat_id = $2`, estID, chatID, lineID, fileID, filePath, caption)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListLinePhotos — фото строки сметы, новые сверху (v0.6).
func (s *Store) ListLinePhotos(ctx context.Context, chatID, estID, lineID int64) ([]domain.PhotoRec, error) {
	rows, err := s.pool.Query(ctx, `
SELECT ph.id, ph.act_id, ph.object_id, ph.file_id, ph.file_path, ph.caption, ph.created_at, 0
FROM photos ph
JOIN estimates e ON e.id = $2 AND e.chat_id = $3
WHERE ph.est_line_id = $1
ORDER BY ph.created_at DESC, ph.id DESC LIMIT 20`, lineID, estID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PhotoRec
	for rows.Next() {
		var p domain.PhotoRec
		if err := rows.Scan(&p.ID, &p.ActID, &p.ObjectID, &p.FileID, &p.FilePath, &p.Caption, &p.CreatedAt, &p.ActNo); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPhoto — фото по id (только своего чата) для подтверждения удаления (v0.3.8).
func (s *Store) GetPhoto(ctx context.Context, chatID, id int64) (domain.PhotoRec, error) {
	var p domain.PhotoRec
	err := s.pool.QueryRow(ctx, `
SELECT ph.id, ph.act_id, ph.object_id, ph.file_id, ph.file_path, ph.caption, ph.created_at,
  COALESCE(a.act_no, 0)
FROM photos ph JOIN objects o ON o.id = ph.object_id LEFT JOIN acts a ON a.id = ph.act_id
WHERE ph.id = $1 AND o.chat_id = $2`, id, chatID).
		Scan(&p.ID, &p.ActID, &p.ObjectID, &p.FileID, &p.FilePath, &p.Caption, &p.CreatedAt, &p.ActNo)
	if err != nil {
		return p, ErrNotFound
	}
	return p, nil
}

// DeletePhoto — убрать неудачный снимок из фотоотчёта (v0.3.8: навигация
// «в обе стороны» — не только добавить, но и убрать). Файл с диска снимает бот.
func (s *Store) DeletePhoto(ctx context.Context, chatID, id int64) (domain.PhotoRec, error) {
	p, err := s.GetPhoto(ctx, chatID, id)
	if err != nil {
		return p, err
	}
	if _, err := s.pool.Exec(ctx, "DELETE FROM photos WHERE id=$1", p.ID); err != nil {
		return p, err
	}
	return p, nil
}

// ListPhotos — фото объекта (actID > 0: только фото этого акта), новые сверху.
// Лимит 10: Telegram не любит длинные простыни, а для демонстрации заказчику
// последних снимков достаточно (v0.3.7).
func (s *Store) ListPhotos(ctx context.Context, objectID, actID int64) ([]domain.PhotoRec, error) {
	q := `
SELECT ph.id, ph.act_id, ph.object_id, ph.file_id, ph.file_path, ph.caption, ph.created_at,
  COALESCE(a.act_no, 0)
FROM photos ph LEFT JOIN acts a ON a.id = ph.act_id
WHERE ph.object_id = $1`
	args := []any{objectID}
	if actID > 0 {
		q += ` AND ph.act_id = $2`
		args = append(args, actID)
	}
	q += ` ORDER BY ph.created_at DESC, ph.id DESC LIMIT 10`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PhotoRec
	for rows.Next() {
		var p domain.PhotoRec
		if err := rows.Scan(&p.ID, &p.ActID, &p.ObjectID, &p.FileID, &p.FilePath, &p.Caption, &p.CreatedAt, &p.ActNo); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- статистика --------------------------------------------------------------

// --- прайс-лист (catalog_items, v0.3) -----------------------------------------

// UpsertCatalogItem добавляет/обновляет позицию прайса (name — уже нормализован).
// Возвращает прежнюю цену, если позиция уже была, — бот показывает «было → стало»,
// чтобы обновление цены не проходило молча (анти-дрейф, v0.3.5).
func (s *Store) UpsertCatalogItem(ctx context.Context, chatID int64, name, unit string, price float64) (prevPrice float64, existed bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback(ctx)

	var prev sql.NullFloat64
	err = tx.QueryRow(ctx, "SELECT price FROM catalog_items WHERE chat_id=$1 AND name=$2", chatID, name).Scan(&prev)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO catalog_items(chat_id, name, unit, price) VALUES($1,$2,$3,$4)
ON CONFLICT (chat_id, name) DO UPDATE SET unit=EXCLUDED.unit, price=EXCLUDED.price`,
		chatID, name, unit, price); err != nil {
		return 0, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, false, err
	}
	return prev.Float64, prev.Valid, nil
}

// BulkUpsertCatalog импортирует позиции прайса одной транзакцией (импорт файла).
// Возвращает число записанных строк.
func (s *Store) BulkUpsertCatalog(ctx context.Context, chatID int64, items []domain.CatalogItem) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	n := 0
	for _, it := range items {
		if _, err := tx.Exec(ctx, `
INSERT INTO catalog_items(chat_id, name, unit, price) VALUES($1,$2,$3,$4)
ON CONFLICT (chat_id, name) DO UPDATE SET unit=EXCLUDED.unit, price=EXCLUDED.price`,
			chatID, it.Name, it.Unit, it.Price); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit(ctx)
}

// ListCatalog — весь прайс чата (по имени).
func (s *Store) ListCatalog(ctx context.Context, chatID int64) ([]domain.CatalogItem, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT name, unit, price FROM catalog_items WHERE chat_id=$1 ORDER BY name", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogItem
	for rows.Next() {
		var it domain.CatalogItem
		if err := rows.Scan(&it.Name, &it.Unit, &it.Price); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CatalogCount — число позиций прайса (для сводки).
func (s *Store) CatalogCount(ctx context.Context, chatID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, "SELECT count(*) FROM catalog_items WHERE chat_id=$1", chatID).Scan(&n)
	return n, err
}

// DeleteCatalogItem удаляет одну позицию прайса и возвращает её (для отчёта
// «убрал: штукатурка, было 260»). ok=false — позиции с таким именем не было.
func (s *Store) DeleteCatalogItem(ctx context.Context, chatID int64, name string) (domain.CatalogItem, bool, error) {
	var it domain.CatalogItem
	rows, err := s.pool.Query(ctx,
		"DELETE FROM catalog_items WHERE chat_id=$1 AND name=$2 RETURNING name, unit, price", chatID, name)
	if err != nil {
		return it, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return it, false, rows.Err()
	}
	if err := rows.Scan(&it.Name, &it.Unit, &it.Price); err != nil {
		return it, false, err
	}
	return it, true, rows.Err()
}

// ClearCatalog удаляет весь прайс чата.
func (s *Store) ClearCatalog(ctx context.Context, chatID int64) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM catalog_items WHERE chat_id=$1", chatID)
	return err
}

type Stats struct {
	Objects    int     `json:"objects"`
	Acts       int     `json:"acts"`
	Total      float64 `json:"total"`
	Paid       float64 `json:"paid"`
	Photos     int     `json:"photos"`
	ZeroActs   int     `json:"zero_acts"`   // акты с суммой 0: работы есть — денег нет (v0.3.7)
	UnpaidActs int     `json:"unpaid_acts"` // акты с долгом: выполнено, но не оплачено (v0.3.7)
}

func (s *Store) Stats(ctx context.Context, chatID int64) (Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM objects o WHERE o.chat_id=$1),
  (SELECT count(*) FROM acts a JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1),
  COALESCE((SELECT SUM(l.sum) FROM act_lines l JOIN acts a ON a.id=l.act_id JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1),0),
  COALESCE((SELECT SUM(p.amount) FROM payments p JOIN acts a ON a.id=p.act_id JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1),0),
  (SELECT count(*) FROM photos ph JOIN objects o ON o.id=ph.object_id WHERE o.chat_id=$1),
  (SELECT count(*) FROM acts a JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1
     AND COALESCE((SELECT SUM(l.sum) FROM act_lines l WHERE l.act_id=a.id),0) < 0.01),
  (SELECT count(*) FROM acts a JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1
     AND COALESCE((SELECT SUM(l.sum) FROM act_lines l WHERE l.act_id=a.id),0)
       - COALESCE((SELECT SUM(p.amount) FROM payments p WHERE p.act_id=a.id),0) > 0.01)`, chatID).
		Scan(&st.Objects, &st.Acts, &st.Total, &st.Paid, &st.Photos, &st.ZeroActs, &st.UnpaidActs)
	return st, err
}

// AdminChatID — чат администратора (для отчётов).
func (s *Store) AdminChatID(ctx context.Context) (int64, bool, error) {
	var id int64
	err := s.pool.QueryRow(ctx, "SELECT chat_id FROM users WHERE is_admin ORDER BY chat_id LIMIT 1").Scan(&id)
	if err != nil {
		return 0, false, nil // ещё никто не нажал /start
	}
	return id, true, nil
}

var _ = time.Now // резерв
