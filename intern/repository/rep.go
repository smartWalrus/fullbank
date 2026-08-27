package repository

import (
	"bank/intern/domain"
	"bank/intern/metrics"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func CreateRepo(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *domain.User) error {
	return MeasureQuery("insert", func() error {
		query := `
	INSERT INTO users (login, password, created_at, role)
	VALUES ($1, $2, $3, $4)
	`
		_, err := r.db.Exec(
			ctx,
			query,
			user.Login,
			user.PasswordHash,
			user.CreatedAt,
			user.Role)
		if err != nil {
			return err
		}
		return nil
	})
}
func (r *Repository) CreateCard(ctx context.Context, card *domain.SimCard) error {
	return MeasureQuery("insert", func() error {
		query := `
	INSERT INTO cards (phone, created_at)
	VALUES ($1, $2)
	`
		_, err := r.db.Exec(ctx, query, card.Phone, card.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to create a card, %w", err)
		}
		return nil
	})
}
func (r *Repository) RentCard(ctx context.Context, user *domain.User, ids []int64, expTime time.Time) error {
	return MeasureQuery("transaction", func() error {
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		updateQuery := `UPDATE cards SET is_busy = TRUE WHERE id = ANY($1)`
		_, err = tx.Exec(ctx, updateQuery, ids)
		if err != nil {
			return err
		}

		insertQuery := `
        INSERT INTO rentals (card_id, user_id, expires_at, status)
        VALUES ($1, $2, $3, $4)
    `
		for _, cardID := range ids {
			_, err = tx.Exec(ctx, insertQuery, cardID, user.Id, expTime, "active")
			if err != nil {
				return err
			}
		}

		return tx.Commit(ctx)
	})
}
func (r *Repository) GetActiveCards(ctx context.Context, id int64) ([]*domain.Rental, error) {
	var cards []*domain.Rental
	err := MeasureQuery("select", func() error {
		query := `
        SELECT c.id, c.phone, c.created_at, r.expires_at, status
        FROM rentals r
        JOIN cards c ON c.id = r.card_id
        WHERE r.user_id = $1 AND r.status = 'active' AND expires_at > NOW() 
        ORDER BY r.rented_at DESC
    `
		rows, err := r.db.Query(ctx, query, id)
		if err != nil {
			return fmt.Errorf("get active cards: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var card domain.Rental
			if err := rows.Scan(&card.Id, &card.Phone, &card.RentedAt, &card.ExpiresAt, &card.Status); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			cards = append(cards, &card)
		}
		return nil
	})
	return cards, err
}
func (r *Repository) FindUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	user := domain.User{
		Login: login,
	}
	err := MeasureQuery("select", func() error {
		query := `
	SELECT id, password, created_at, role FROM users
	WHERE login = $1
	`

		return r.db.QueryRow(ctx, query, login).Scan(&user.Id, &user.PasswordHash, &user.CreatedAt, &user.Role)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *Repository) FindFreeCards(ctx context.Context, amount int) ([]int64, error) {
	ids := make([]int64, 0, amount)
	err := MeasureQuery("select", func() error {
		query := `
		SELECT id FROM cards
		WHERE is_busy IS FALSE
		LIMIT $1
		`
		slog.Info(" SQL Query", "query", query, "amount", amount)
		rows, err := r.db.Query(ctx, query, amount)
		if err != nil {
			return fmt.Errorf("failed to send query, %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			ids = append(ids, id)
		}
		return nil
	})
	return ids, err
}
func (r *Repository) UpdatePass(ctx context.Context, user *domain.User) error {
	return MeasureQuery("update", func() error {
		query := `
	UPDATE users
	SET password = $1
	WHERE id = $2
	`
		if _, err := r.db.Exec(ctx, query, user.PasswordHash, user.Id); err != nil {
			return fmt.Errorf("failed to send query, %w", err)
		}
		return nil
	})
}
func (r *Repository) ExpireRentals(ctx context.Context) (int64, error) {
	var affected int64
	err := MeasureQuery("update", func() error {
		query := `
       WITH expired_cards AS (
            UPDATE rentals
            SET status = 'expired'
            WHERE status = 'active' AND expires_at < NOW()
            RETURNING card_id
        )
        UPDATE cards
        SET is_busy = FALSE
        WHERE id IN (SELECT card_id FROM expired_cards)
    
    `
		rows, err := r.db.Exec(ctx, query)
		if err != nil {
			affected = rows.RowsAffected()
			return fmt.Errorf("expire rentals: %w", err)
		}
		return nil
	})
	return affected, err
}
func (r *Repository) GetExpiredCards(ctx context.Context, id int64) ([]*domain.Rental, error) {
	var rentals []*domain.Rental
	err := MeasureQuery("select", func() error {
		query := `
	SELECT rentals.id, rentals.card_id, cards.phone, rentals.rented_at, rentals.expires_at, rentals.status FROM rentals
	JOIN cards ON cards.id = rentals.card_id
	WHERE status = 'expired'
	ORDER BY rentals.expires_at DESC
	`
		rows, err := r.db.Query(ctx, query)
		if err != nil {
			return fmt.Errorf("get expired cards: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var rental domain.Rental
			if err := rows.Scan(
				&rental.Id,
				&rental.CardId,
				&rental.Phone,
				&rental.RentedAt,
				&rental.ExpiresAt,
				&rental.Status,
			); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			rentals = append(rentals, &rental)
		}
		return nil
	})
	return rentals, err
}
func MeasureQuery(queryType string, fn func() error) error {
	start := time.Now()
	err := fn()
	metrics.DbQueriesTotal.WithLabelValues(queryType).Inc()
	metrics.DbQueryDuration.WithLabelValues(queryType).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.DbErrorsTotal.WithLabelValues(queryType).Inc()
	}
	return err
}
func (r *Repository) WriteNote(ctx context.Context, id int, message string) error {
	query := `
	UPDATE cards
	SET comment = $1
	WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, message, id)
	if err != nil {
		return fmt.Errorf("failed to set comment, %w", err)
	}
	return nil
}
