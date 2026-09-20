// Доступ заказчика — привязка object_id + telegram_user_id (пилот 360).
package store

import (
	"context"

	"proakt/internal/domain"
)

// GrantClient привязывает заказчика к объекту (повторно — включает обратно).
func (s *Store) GrantClient(ctx context.Context, objectID, tgUserID int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO client_access(object_id, tg_user_id, status) VALUES($1,$2,'active')
		 ON CONFLICT (object_id, tg_user_id) DO UPDATE SET status='active'`,
		objectID, tgUserID)
	return err
}

// RevokeClient отключает доступ заказчика.
func (s *Store) RevokeClient(ctx context.Context, objectID, tgUserID int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE client_access SET status='revoked' WHERE object_id=$1 AND tg_user_id=$2`,
		objectID, tgUserID)
	return err
}

// CanClientSee проверяет активный доступ заказчика к объекту.
func (s *Store) CanClientSee(ctx context.Context, tgUserID, objectID int64) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM client_access WHERE object_id=$1 AND tg_user_id=$2 AND status='active'`,
		objectID, tgUserID).Scan(&n)
	return n == 1, err
}

// ClientObjects — объекты, разрешённые заказчику.
func (s *Store) ClientObjects(ctx context.Context, tgUserID int64) ([]domain.ObjectBrief, error) {
	rows, err := s.pool.Query(ctx, `
SELECT o.id, o.chat_id, o.name, o.customer, o.status, o.created_at,
  (SELECT count(*) FROM acts a WHERE a.object_id=o.id) AS acts,
  COALESCE((SELECT SUM(l.sum) FROM acts a JOIN act_lines l ON l.act_id=a.id WHERE a.object_id=o.id),0) AS total,
  COALESCE((SELECT SUM(p.amount) FROM acts a JOIN payments p ON p.act_id=a.id WHERE a.object_id=o.id),0) AS paid,
  (SELECT count(*) FROM photos ph WHERE ph.object_id=o.id) AS photos
FROM objects o JOIN client_access c ON c.object_id=o.id
WHERE c.tg_user_id=$1 AND c.status='active'
ORDER BY o.created_at DESC`, tgUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ObjectBrief
	for rows.Next() {
		var b domain.ObjectBrief
		if err := rows.Scan(&b.ID, &b.ChatID, &b.Name, &b.Customer, &b.Status, &b.CreatedAt, &b.Acts, &b.Total, &b.Paid, &b.Photos); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ClientList — привязанные ID заказчика по объекту (для админки).
func (s *Store) ClientList(ctx context.Context, objectID int64) ([]int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT tg_user_id FROM client_access WHERE object_id=$1 AND status='active' ORDER BY tg_user_id`,
		objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
