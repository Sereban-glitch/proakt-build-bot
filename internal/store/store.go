// Package store — доступ к PostgreSQL (pgx, пул ограничен — экономия RAM).
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
CREATE TABLE IF NOT EXISTS schema_migrations (
  version    INT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
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

// Migrate — идемпотентное применение схемы.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	_, err := s.pool.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES (1) ON CONFLICT DO NOTHING")
	return err
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

func (s *Store) ListObjects(ctx context.Context, chatID int64) ([]domain.Object, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT id, chat_id, name, customer, status, created_at FROM objects WHERE chat_id=$1 AND status='active' ORDER BY id", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Object
	for rows.Next() {
		var o domain.Object
		if err := rows.Scan(&o.ID, &o.ChatID, &o.Name, &o.Customer, &o.Status, &o.CreatedAt); err != nil {
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
SELECT a.id, a.act_no, a.object_id, o.name, o.customer,
  COALESCE((SELECT SUM(l.sum)   FROM act_lines l WHERE l.act_id = a.id), 0) AS total,
  COALESCE((SELECT SUM(p.amount) FROM payments p WHERE p.act_id = a.id), 0) AS paid
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
		if err := rows.Scan(&b.ID, &b.ActNo, &b.ObjectID, &b.ObjectName, &b.Customer, &total, &paid); err != nil {
			return nil, err
		}
		b.Total, b.Paid = total, paid
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) GetAct(ctx context.Context, actID int64) (domain.ActBrief, error) {
	var b domain.ActBrief
	var total, paid float64
	err := s.pool.QueryRow(ctx, `
SELECT a.id, a.act_no, a.object_id, o.name, o.customer,
  COALESCE((SELECT SUM(l.sum)   FROM act_lines l WHERE l.act_id = a.id), 0),
  COALESCE((SELECT SUM(p.amount) FROM payments p WHERE p.act_id = a.id), 0)
FROM acts a JOIN objects o ON o.id = a.object_id WHERE a.id = $1`, actID).
		Scan(&b.ID, &b.ActNo, &b.ObjectID, &b.ObjectName, &b.Customer, &total, &paid)
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

func (s *Store) AddPhoto(ctx context.Context, actID *int64, objectID int64, fileID, filePath, caption string) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO photos(act_id, object_id, file_id, file_path, caption) VALUES($1,$2,$3,$4,$5)",
		actID, objectID, fileID, filePath, caption)
	return err
}

// --- статистика --------------------------------------------------------------

type Stats struct {
	Objects int
	Acts    int
	Total   float64
	Paid    float64
	Photos  int
}

func (s *Store) Stats(ctx context.Context, chatID int64) (Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM objects o WHERE o.chat_id=$1),
  (SELECT count(*) FROM acts a JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1),
  COALESCE((SELECT SUM(l.sum) FROM act_lines l JOIN acts a ON a.id=l.act_id JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1),0),
  COALESCE((SELECT SUM(p.amount) FROM payments p JOIN acts a ON a.id=p.act_id JOIN objects o ON o.id=a.object_id WHERE o.chat_id=$1),0),
  (SELECT count(*) FROM photos ph JOIN objects o ON o.id=ph.object_id WHERE o.chat_id=$1)`, chatID).
		Scan(&st.Objects, &st.Acts, &st.Total, &st.Paid, &st.Photos)
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
