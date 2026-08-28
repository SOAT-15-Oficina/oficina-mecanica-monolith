package repository

import (
	"context"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) application.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	query := `
		INSERT INTO users (id, username, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, username, password_hash, role`

	var result domain.User
	err := r.db.QueryRow(ctx, query, user.ID, user.Username, user.PasswordHash, user.Role).
		Scan(&result.ID, &result.Username, &result.PasswordHash, &result.Role)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, username, password_hash, role FROM users WHERE id = $1`

	var result domain.User
	err := r.db.QueryRow(ctx, query, id).
		Scan(&result.ID, &result.Username, &result.PasswordHash, &result.Role)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, password_hash, role FROM users WHERE username = $1`

	var result domain.User
	err := r.db.QueryRow(ctx, query, username).
		Scan(&result.ID, &result.Username, &result.PasswordHash, &result.Role)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *userRepository) FindAll(ctx context.Context) ([]domain.User, error) {
	query := `SELECT id, username, password_hash, role FROM users`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		UPDATE users
		SET username = $1, password_hash = $2, role = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, username, password_hash, role`

	var result domain.User
	err := r.db.QueryRow(ctx, query, user.Username, user.PasswordHash, user.Role, user.ID).
		Scan(&result.ID, &result.Username, &result.PasswordHash, &result.Role)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}
