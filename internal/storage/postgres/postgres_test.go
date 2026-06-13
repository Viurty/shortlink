package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"shortlink/internal/storage"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

func newMockDatabase(t *testing.T) (*Database, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	db := sqlx.NewDb(sqlDB, "sqlmock")
	cleanup := func() {
		_ = db.Close()
	}

	return &Database{db: db}, mock, cleanup
}

func TestDatabaseSave(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	mock.ExpectExec(insertLinkQuery).
		WithArgs("short_code", "https://example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := db.Save(context.Background(), "short_code", "https://example.com"); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDatabaseSaveConflicts(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		wantErr    error
	}{
		{name: "url conflict", constraint: "links_original_url_key", wantErr: storage.ErrURLConflict},
		{name: "code conflict", constraint: "links_code_key", wantErr: storage.ErrCodeConflict},
		{name: "unknown constraint falls back to code conflict", constraint: "links_unknown_key", wantErr: storage.ErrCodeConflict},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, cleanup := newMockDatabase(t)
			defer cleanup()

			mock.ExpectExec(insertLinkQuery).
				WithArgs("short_code", "https://example.com").
				WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: tc.constraint})

			err := db.Save(context.Background(), "short_code", "https://example.com")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

func TestDatabaseSaveUnknownDBError(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	dbErr := errors.New("connection reset by peer")
	mock.ExpectExec(insertLinkQuery).
		WithArgs("short_code", "https://example.com").
		WillReturnError(dbErr)

	err := db.Save(context.Background(), "short_code", "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, storage.ErrURLConflict) || errors.Is(err, storage.ErrCodeConflict) || errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("unexpected sentinel error: %v", err)
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("original error not wrapped: %v", err)
	}
}

func TestDatabaseGetByOriginal(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	mock.ExpectQuery(selectCodeByOriginal).
		WithArgs("https://example.com").
		WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("short_code"))

	code, err := db.GetByOriginal(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("GetByOriginal returned error: %v", err)
	}
	if code != "short_code" {
		t.Fatalf("unexpected code: got %q want %q", code, "short_code")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDatabaseGetByOriginalNotFound(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	mock.ExpectQuery(selectCodeByOriginal).
		WithArgs("https://missing.example.com").
		WillReturnError(sql.ErrNoRows)

	_, err := db.GetByOriginal(context.Background(), "https://missing.example.com")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDatabaseGetByOriginalUnknownDBError(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	dbErr := errors.New("connection reset by peer")
	mock.ExpectQuery(selectCodeByOriginal).
		WithArgs("https://example.com").
		WillReturnError(dbErr)

	_, err := db.GetByOriginal(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, storage.ErrNotFound) {
		t.Fatal("unexpected ErrNotFound for unknown DB error")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("original error not wrapped: %v", err)
	}
}

func TestDatabaseGetByCode(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	mock.ExpectQuery(selectOriginalByCode).
		WithArgs("short_code").
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("https://example.com"))

	originalURL, err := db.GetByCode(context.Background(), "short_code")
	if err != nil {
		t.Fatalf("GetByCode returned error: %v", err)
	}
	if originalURL != "https://example.com" {
		t.Fatalf("unexpected original URL: got %q want %q", originalURL, "https://example.com")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDatabaseGetByCodeNotFound(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	mock.ExpectQuery(selectOriginalByCode).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	_, err := db.GetByCode(context.Background(), "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDatabaseGetByCodeUnknownDBError(t *testing.T) {
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	dbErr := errors.New("connection reset by peer")
	mock.ExpectQuery(selectOriginalByCode).
		WithArgs("short_code").
		WillReturnError(dbErr)

	_, err := db.GetByCode(context.Background(), "short_code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, storage.ErrNotFound) {
		t.Fatal("unexpected ErrNotFound for unknown DB error")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("original error not wrapped: %v", err)
	}
}
