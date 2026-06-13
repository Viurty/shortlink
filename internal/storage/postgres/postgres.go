package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"shortlink/internal/storage"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Database struct {
	db *sqlx.DB
}

const (
	insertLinkQuery      = `INSERT INTO links (code, original_url) VALUES ($1, $2)`
	selectCodeByOriginal = `SELECT code FROM links WHERE original_url = $1`
	selectOriginalByCode = `SELECT original_url FROM links WHERE code = $1`
)

func New(ctx context.Context, dsn string) (*Database, error) {
	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Save(ctx context.Context, code, originalURL string) error {
	_, err := d.db.ExecContext(ctx, insertLinkQuery, code, originalURL)
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
		return fmt.Errorf("save: %w", err)
	}
	return nil
}

func (d *Database) GetByOriginal(ctx context.Context, originalURL string) (string, error) {
	var code string
	err := d.db.GetContext(ctx, &code, selectCodeByOriginal, originalURL)
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
	err := d.db.GetContext(ctx, &originalURL, selectOriginalByCode, code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get original by code: %w", err)
	}
	return originalURL, nil
}
