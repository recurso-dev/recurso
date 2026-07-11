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
	var mfaSecret sql.NullString
	if err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Name, &role, &u.MFAEnabled, &mfaSecret, &u.MFALastTimestep, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.Role = domain.Role(role)
	if mfaSecret.Valid {
		u.MFASecret = mfaSecret.String
	}
	return &u, nil
}

const userSelectCols = `id, tenant_id, email, password_hash, name, role, mfa_enabled, mfa_secret, mfa_last_timestep, created_at, updated_at`

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

func (r *UserRepository) GetByIDGlobal(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT ` + userSelectCols + ` FROM users WHERE id = $1`
	u, err := scanUser(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, id,
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

func (r *UserRepository) SetMFASecret(ctx context.Context, tenantID, id uuid.UUID, secret string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_secret = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		secret, id, tenantID,
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

func (r *UserRepository) SetMFAEnabled(ctx context.Context, tenantID, id uuid.UUID, enabled bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_enabled = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		enabled, id, tenantID,
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

// SetMFALastTimestep records the last consumed TOTP timestep (ENG-151). The
// `$1 > mfa_last_timestep` guard keeps it monotonic, so two concurrent logins
// with the same code can't both advance it — the replay is rejected.
func (r *UserRepository) SetMFALastTimestep(ctx context.Context, tenantID, id uuid.UUID, timestep int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_last_timestep = $1, updated_at = NOW()
		 WHERE id = $2 AND tenant_id = $3 AND $1 > mfa_last_timestep`,
		timestep, id, tenantID,
	)
	return err
}

func (r *UserRepository) ClearMFA(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET mfa_enabled = FALSE, mfa_secret = NULL, updated_at = NOW() WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
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
