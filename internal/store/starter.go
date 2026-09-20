// Стартовые данные 360: метки, очистка, удаление объекта, флаг финансов.
package store

import (
	"context"
)

// ClientGrant — привязка заказчика с флагом видимости финансов.
type ClientGrant struct {
	UserID      int64
	ShowFinance bool
}

// StarterReport — что удалила очистка стартовых примеров.
type StarterReport struct {
	Objects int
	Prices  int
}

// MarkStarterObject помечает объект как стартовый (чат мастера).
func (s *Store) MarkStarterObject(ctx context.Context, chatID, objectID int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO seed_objects(object_id, chat_id) VALUES($1,$2) ON CONFLICT DO NOTHING`,
		objectID, chatID)
	return err
}

// MarkStarterCatalog помечает прайс-позиции как стартовые.
func (s *Store) MarkStarterCatalog(ctx context.Context, chatID int64, names []string) error {
	for _, n := range names {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO seed_catalog(chat_id, name) VALUES($1,$2) ON CONFLICT DO NOTHING`,
			chatID, n); err != nil {
			return err
		}
	}
	return nil
}

// StarterHasData — есть ли неудалённые стартовые примеры у чата.
func (s *Store) StarterHasData(ctx context.Context, chatID int64) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM seed_objects WHERE chat_id=$1) + (SELECT count(*) FROM seed_catalog WHERE chat_id=$1)`, chatID).Scan(&n)
	return n > 0, err
}

// DeleteStarterData удаляет ТОЛЬКО помеченные стартовые записи чата:
// объекты (акты, сметы, фото, оплаты уходят каскадом) и стартовый прайс.
// Данные Виталика не тронуты. Транзакция целиком.
func (s *Store) DeleteStarterData(ctx context.Context, chatID int64) (StarterReport, error) {
	var rep StarterReport
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return rep, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT object_id FROM seed_objects WHERE chat_id=$1`, chatID)
	if err != nil {
		return rep, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return rep, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		tag, err := tx.Exec(ctx, `DELETE FROM objects WHERE id=$1 AND chat_id=$2`, id, chatID)
		if err != nil {
			return rep, err
		}
		rep.Objects += int(tag.RowsAffected())
	}
	names := tx.QueryRow(ctx, `SELECT count(*) FROM seed_catalog WHERE chat_id=$1`, chatID)
	var c int
	if err := names.Scan(&c); err != nil {
		return rep, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM catalog_items WHERE chat_id=$1 AND name IN (SELECT name FROM seed_catalog WHERE chat_id=$1)`, chatID); err != nil {
		return rep, err
	}
	rep.Prices = c
	if _, err := tx.Exec(ctx, `DELETE FROM seed_catalog WHERE chat_id=$1`, chatID); err != nil {
		return rep, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM seed_objects WHERE chat_id=$1`, chatID); err != nil {
		return rep, err
	}
	return rep, tx.Commit(ctx)
}

// DeleteObject — полное удаление объекта мастера (всё каскадом).
func (s *Store) DeleteObject(ctx context.Context, chatID, objID int64) (string, error) {
	var name string
	err := s.pool.QueryRow(ctx,
		`DELETE FROM objects WHERE id=$1 AND chat_id=$2 RETURNING name`, objID, chatID).Scan(&name)
	if err != nil {
		return "", ErrNotFound
	}
	return name, nil
}

// SetClientFinance — показать/скрыть финансы заказчику.
func (s *Store) SetClientFinance(ctx context.Context, objectID, tgUserID int64, show bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE client_access SET show_finance=$3 WHERE object_id=$1 AND tg_user_id=$2 AND status='active'`,
		objectID, tgUserID, show)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ClientFinanceVisible — разрешены ли финансы заказчику.
func (s *Store) ClientFinanceVisible(ctx context.Context, tgUserID, objectID int64) (bool, error) {
	var show bool
	err := s.pool.QueryRow(ctx,
		`SELECT show_finance FROM client_access WHERE object_id=$1 AND tg_user_id=$2 AND status='active'`,
		objectID, tgUserID).Scan(&show)
	if err != nil {
		return false, ErrNotFound
	}
	return show, nil
}
