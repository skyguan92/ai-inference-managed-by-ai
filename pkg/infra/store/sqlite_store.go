package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements ModelStore using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed model store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS models (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		format TEXT NOT NULL,
		status TEXT NOT NULL,
		source TEXT,
		path TEXT,
		size INTEGER DEFAULT 0,
		checksum TEXT,
		metadata TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_models_type ON models(type);
	CREATE INDEX IF NOT EXISTS idx_models_status ON models(status);
	CREATE INDEX IF NOT EXISTS idx_models_name ON models(name);
	`
	_, err := s.db.Exec(query)
	return err
}

// Create implements ModelStore.Create
func (s *SQLiteStore) Create(ctx context.Context, m *model.Model) error {
	// Serialize tags as JSON for storage
	tagsJSON, _ := json.Marshal(m.Tags)

	query := `
		INSERT INTO models (id, name, type, format, status, source, path, size, checksum, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		m.ID, m.Name, string(m.Type), string(m.Format), string(m.Status),
		m.Source, m.Path, m.Size, m.Checksum, string(tagsJSON),
		m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert model: %w", err)
	}
	return nil
}

// Get implements ModelStore.Get
func (s *SQLiteStore) Get(ctx context.Context, id string) (*model.Model, error) {
	query := `SELECT id, name, type, format, status, source, path, size, checksum, metadata, created_at, updated_at FROM models WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	m := &model.Model{}
	var tagsStr string
	var typeStr, formatStr, statusStr string

	err := row.Scan(
		&m.ID, &m.Name, &typeStr, &formatStr, &statusStr,
		&m.Source, &m.Path, &m.Size, &m.Checksum, &tagsStr,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrModelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan model: %w", err)
	}

	m.Type = model.ModelType(typeStr)
	m.Format = model.ModelFormat(formatStr)
	m.Status = model.ModelStatus(statusStr)

	if tagsStr != "" {
		json.Unmarshal([]byte(tagsStr), &m.Tags)
	}

	return m, nil
}

// List implements ModelStore.List
func (s *SQLiteStore) List(ctx context.Context, filter model.ModelFilter) ([]model.Model, int, error) {
	whereClause := "1=1"
	args := []interface{}{}

	if filter.Type != "" {
		whereClause += " AND type = ?"
		args = append(args, string(filter.Type))
	}
	if filter.Status != "" {
		whereClause += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.Format != "" {
		whereClause += " AND format = ?"
		args = append(args, string(filter.Format))
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM models WHERE %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count models: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, name, type, format, status, source, path, size, checksum, metadata, created_at, updated_at
		FROM models
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query models: %w", err)
	}
	defer rows.Close()

	var models []model.Model
	for rows.Next() {
		m := model.Model{}
		var tagsStr string
		var typeStr, formatStr, statusStr string

		err := rows.Scan(
			&m.ID, &m.Name, &typeStr, &formatStr, &statusStr,
			&m.Source, &m.Path, &m.Size, &m.Checksum, &tagsStr,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan model: %w", err)
		}

		m.Type = model.ModelType(typeStr)
		m.Format = model.ModelFormat(formatStr)
		m.Status = model.ModelStatus(statusStr)

		if tagsStr != "" {
			json.Unmarshal([]byte(tagsStr), &m.Tags)
		}

		models = append(models, m)
	}

	return models, total, nil
}

// Delete implements ModelStore.Delete
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM models WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete model: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return model.ErrModelNotFound
	}
	return nil
}

// Update implements ModelStore.Update
func (s *SQLiteStore) Update(ctx context.Context, m *model.Model) error {
	// Serialize tags as JSON for storage
	tagsJSON, _ := json.Marshal(m.Tags)

	query := `
		UPDATE models SET 
			name = ?, type = ?, format = ?, status = ?, source = ?, 
			path = ?, size = ?, checksum = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query,
		m.Name, string(m.Type), string(m.Format), string(m.Status), m.Source,
		m.Path, m.Size, m.Checksum, string(tagsJSON), time.Now().Unix(),
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("update model: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return model.ErrModelNotFound
	}
	return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Ensure SQLiteStore implements ModelStore interface
var _ model.ModelStore = (*SQLiteStore)(nil)
