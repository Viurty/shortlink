package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"shortlink/internal/storage"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Database struct {
	db *sqlx.DB
}

func New(ctx context.Context, dsn string) (*Database, error) {
	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Save(ctx context.Context, code, originalURL string) error {
	query := `
        INSERT INTO links (code, original_url)
        VALUES ($1, $2)
    `
	_, err := d.db.ExecContext(ctx, query, code, originalURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				if pgErr.ConstraintName == "links_original_url_key" {
					return storage.ErrURLConflict
				}
				return storage.ErrCodeConflict
			}
		}
		return fmt.Errorf("сохранение: %w", err)
	}
	return nil
}

func (d *Database) GetByOriginal(ctx context.Context, originalURL string) (string, error) {
	var code string
	query := `SELECT code FROM links WHERE original_url = $1`
	err := d.db.GetContext(ctx, &code, query, originalURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get code by original: %w", err)
	}
	return code, nil
}

func (d *Database) GetByCode(ctx context.Context, code string) (string, error) {
	var originalURL string
	query := `SELECT original_url FROM links WHERE code = $1`
	err := d.db.GetContext(ctx, &originalURL, query, code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get original by code: %w", err)
	}
	return originalURL, nil
}
