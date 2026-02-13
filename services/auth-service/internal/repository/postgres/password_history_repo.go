package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hightrex/microservices-platform/libs/go/pkg/errors"
)

// PasswordHistoryRepo implements service.PasswordHistoryRepository.
type PasswordHistoryRepo struct {
	db *pgxpool.Pool
}

// NewPasswordHistoryRepo creates a new PasswordHistoryRepo.
func NewPasswordHistoryRepo(db *pgxpool.Pool) *PasswordHistoryRepo {
	return &PasswordHistoryRepo{db: db}
}

// Add records a password hash in the history.
func (r *PasswordHistoryRepo) Add(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_history (id, user_id, password_hash) VALUES (gen_random_uuid(), $1, $2)`,
		userID, passwordHash,
	)
	if err != nil {
		return errors.InternalServerError("Failed to add password history", err)
	}
	return nil
}

// GetRecent retrieves the most recent N password hashes for a user.
func (r *PasswordHistoryRepo) GetRecent(ctx context.Context, userID uuid.UUID, count int) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT password_hash FROM password_history WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, count,
	)
	if err != nil {
		return nil, errors.InternalServerError("Failed to get password history", err)
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, errors.InternalServerError("Failed to scan password hash", err)
		}
		hashes = append(hashes, h)
	}
	return hashes, nil
}
