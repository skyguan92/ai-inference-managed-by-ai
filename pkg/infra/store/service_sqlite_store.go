package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// ServiceSQLiteStore implements ServiceStore using SQLite
type ServiceSQLiteStore struct {
	db *sql.DB
}

// NewServiceSQLiteStore creates a new SQLite-backed service store
func NewServiceSQLiteStore(db *sql.DB) (*ServiceSQLiteStore, error) {
	s := &ServiceSQLiteStore{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return s, nil
}

// initSchema creates the database schema
func (s *ServiceSQLiteStore) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS services (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		model_id TEXT NOT NULL,
		status TEXT NOT NULL,
		replicas INTEGER DEFAULT 1,
		resource_class TEXT NOT NULL,
		endpoints TEXT,
		active_replicas INTEGER DEFAULT 0,
		config TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_services_model_id ON services(model_id);
	CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
	`
	_, err := s.db.Exec(query)
	return err
}

// Create implements ServiceStore.Create
func (s *ServiceSQLiteStore) Create(ctx context.Context, svc *service.ModelService) error {
	query := `
		INSERT INTO services (id, name, model_id, status, replicas, resource_class, endpoints, active_replicas, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	endpoints := ""
	if len(svc.Endpoints) > 0 {
		endpoints = svc.Endpoints[0]
	}
	_, err := s.db.ExecContext(ctx, query,
		svc.ID, svc.Name, svc.ModelID, string(svc.Status), svc.Replicas,
		string(svc.ResourceClass), endpoints, svc.ActiveReplicas, "",
		svc.CreatedAt, svc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert service: %w", err)
	}
	return nil
}

// GetByName implements ServiceStore.GetByName
func (s *ServiceSQLiteStore) GetByName(ctx context.Context, name string) (*service.ModelService, error) {
	query := `SELECT id, name, model_id, status, replicas, resource_class, endpoints, active_replicas, created_at, updated_at FROM services WHERE name = ?`
	row := s.db.QueryRowContext(ctx, query, name)

	svc := &service.ModelService{}
	var statusStr, resourceClassStr string
	var endpoints string

	err := row.Scan(
		&svc.ID, &svc.Name, &svc.ModelID, &statusStr, &svc.Replicas,
		&resourceClassStr, &endpoints, &svc.ActiveReplicas,
		&svc.CreatedAt, &svc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrServiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan service: %w", err)
	}

	svc.Status = service.ServiceStatus(statusStr)
	svc.ResourceClass = service.ResourceClass(resourceClassStr)
	if endpoints != "" {
		svc.Endpoints = []string{endpoints}
	}

	return svc, nil
}

// Get implements ServiceStore.Get
func (s *ServiceSQLiteStore) Get(ctx context.Context, id string) (*service.ModelService, error) {
	query := `SELECT id, name, model_id, status, replicas, resource_class, endpoints, active_replicas, created_at, updated_at FROM services WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	svc := &service.ModelService{}
	var statusStr, resourceClassStr string
	var endpoints string

	err := row.Scan(
		&svc.ID, &svc.Name, &svc.ModelID, &statusStr, &svc.Replicas,
		&resourceClassStr, &endpoints, &svc.ActiveReplicas,
		&svc.CreatedAt, &svc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrServiceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan service: %w", err)
	}

	svc.Status = service.ServiceStatus(statusStr)
	svc.ResourceClass = service.ResourceClass(resourceClassStr)
	if endpoints != "" {
		svc.Endpoints = []string{endpoints}
	}

	return svc, nil
}

// List implements ServiceStore.List
func (s *ServiceSQLiteStore) List(ctx context.Context, filter service.ServiceFilter) ([]service.ModelService, int, error) {
	whereClause := "1=1"
	args := []interface{}{}

	if filter.Status != "" {
		whereClause += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.ModelID != "" {
		whereClause += " AND model_id = ?"
		args = append(args, filter.ModelID)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM services WHERE %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count services: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, name, model_id, status, replicas, resource_class, endpoints, active_replicas, created_at, updated_at
		FROM services
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
		return nil, 0, fmt.Errorf("query services: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var services []service.ModelService
	for rows.Next() {
		svc := service.ModelService{}
		var statusStr, resourceClassStr string
		var endpoints string

		err := rows.Scan(
			&svc.ID, &svc.Name, &svc.ModelID, &statusStr, &svc.Replicas,
			&resourceClassStr, &endpoints, &svc.ActiveReplicas,
			&svc.CreatedAt, &svc.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan service: %w", err)
		}

		svc.Status = service.ServiceStatus(statusStr)
		svc.ResourceClass = service.ResourceClass(resourceClassStr)
		if endpoints != "" {
			svc.Endpoints = []string{endpoints}
		}

		services = append(services, svc)
	}

	return services, total, nil
}

// Delete implements ServiceStore.Delete
func (s *ServiceSQLiteStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM services WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete service: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return service.ErrServiceNotFound
	}
	return nil
}

// Update implements ServiceStore.Update
func (s *ServiceSQLiteStore) Update(ctx context.Context, svc *service.ModelService) error {
	query := `
		UPDATE services SET
			name = ?, model_id = ?, status = ?, replicas = ?, resource_class = ?,
			endpoints = ?, active_replicas = ?, updated_at = ?
		WHERE id = ?
	`
	endpoints := ""
	if len(svc.Endpoints) > 0 {
		endpoints = svc.Endpoints[0]
	}
	result, err := s.db.ExecContext(ctx, query,
		svc.Name, svc.ModelID, string(svc.Status), svc.Replicas, string(svc.ResourceClass),
		endpoints, svc.ActiveReplicas, time.Now().Unix(),
		svc.ID,
	)
	if err != nil {
		return fmt.Errorf("update service: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return service.ErrServiceNotFound
	}
	return nil
}

// Ensure ServiceSQLiteStore implements ServiceStore interface
var _ service.ServiceStore = (*ServiceSQLiteStore)(nil)
