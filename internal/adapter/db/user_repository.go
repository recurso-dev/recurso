package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// UserRepository is the Postgres-backed store for dashboard user accounts.
type UserRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db, logger: slog.Default().With("repo", "user")}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (id, tenant_id, email, password_hash, name, role, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.TenantID, normalizeEmail(u.Email), u.PasswordHash, u.Name, string(u.Role), u.CreatedAt, u.UpdatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "users_email_lower_unique") {
		return domain.ErrDuplicateEmail
	}
	return err
}

func scanUser(row interface{ Scan(...any) error }) (*domain.User, error) {
	var u domain.User
	var role string
	if err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Name, &role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.Role = domain.Role(role)
	return &u, nil
}

const userSelectCols = `id, tenant_id, email, password_hash, name, role, created_at, updated_at`

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userSelectCols + ` FROM users WHERE lower(email) = lower($1)`
	u, err := scanUser(r.db.QueryRowContext(ctx, query, normalizeEmail(email)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = lower($1))`, normalizeEmail(email)).Scan(&exists)
	return exists, err
}

func (r *UserRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	query := `SELECT ` + userSelectCols + ` FROM users WHERE id = $1 AND tenant_id = $2`
	u, err := scanUser(r.db.QueryRowContext(ctx, query, id, tenantID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error) {
	query := `SELECT ` + userSelectCols + ` FROM users WHERE tenant_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.Role) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		string(role), id, tenantID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) CountOwners(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND role = 'owner'`, tenantID,
	).Scan(&n)
	return n, err
}
