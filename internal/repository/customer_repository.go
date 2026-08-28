package repository

import (
	"context"
	"errors"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/application"
	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type customerRepository struct {
	db *pgxpool.Pool
}

func NewCustomerRepository(db *pgxpool.Pool) application.CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) Create(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	if customer.ID == uuid.Nil {
		customer.ID = uuid.New()
	}

	query := `
		INSERT INTO customers (id, name, email, document, document_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, name, email, document, document_type, created_at, updated_at`

	var result domain.Customer
	err := r.db.QueryRow(ctx, query, customer.ID, customer.Name, customer.Email, customer.Document, customer.DocumentType).
		Scan(&result.ID, &result.Name, &result.Email, &result.Document, &result.DocumentType, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *customerRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	query := `SELECT id, name, email, document, document_type FROM customers WHERE id = $1`

	var result domain.Customer
	err := r.db.QueryRow(ctx, query, id).
		Scan(&result.ID, &result.Name, &result.Email, &result.Document, &result.DocumentType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

func (r *customerRepository) FindAll(ctx context.Context) ([]domain.Customer, error) {
	query := `SELECT id, name, email, document, document_type FROM customers`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []domain.Customer
	for rows.Next() {
		var customer domain.Customer
		if err := rows.Scan(&customer.ID, &customer.Name, &customer.Email, &customer.Document, &customer.DocumentType); err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	return customers, nil
}

func (r *customerRepository) FindAllWithFilters(ctx context.Context, filters domain.CustomerListFilters) ([]domain.Customer, error) {
	query := `SELECT id, name, email, document, document_type FROM customers`
	args := []any{}

	if filters.Document != "" {
		query += ` WHERE document = $1`
		args = append(args, filters.Document)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []domain.Customer
	for rows.Next() {
		var customer domain.Customer
		if err := rows.Scan(&customer.ID, &customer.Name, &customer.Email, &customer.Document, &customer.DocumentType); err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	return customers, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	query := `
		UPDATE customers
		SET name = $1, email = $2, document = $3, document_type = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, name, email, document, document_type, created_at, updated_at`

	var result domain.Customer
	err := r.db.QueryRow(ctx, query, customer.Name, customer.Email, customer.Document, customer.DocumentType, customer.ID).
		Scan(&result.ID, &result.Name, &result.Email, &result.Document, &result.DocumentType, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *customerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM customers WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}
