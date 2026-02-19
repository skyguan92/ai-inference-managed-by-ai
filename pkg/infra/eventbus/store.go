package eventbus

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type EventStore interface {
	Save(ctx context.Context, event unit.Event) error
	Query(ctx context.Context, filter EventQueryFilter) ([]unit.Event, error)
	GetByID(ctx context.Context, id string) (unit.Event, error)
}

type EventQueryFilter struct {
	Domain        string
	Type          string
	CorrelationID string
	StartTime     time.Time
	EndTime       time.Time
	Limit         int
}

type storedEvent struct {
	id            string
	eventType     string
	domain        string
	correlationID string
	payload       []byte
	timestamp     time.Time
}

func (e *storedEvent) Type() string          { return e.eventType }
func (e *storedEvent) Domain() string        { return e.domain }
func (e *storedEvent) Payload() any          { return e.payload }
func (e *storedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *storedEvent) CorrelationID() string { return e.correlationID }

var _ unit.Event = (*storedEvent)(nil)

type SQLiteEventStore struct {
	db *sql.DB
}

func NewSQLiteEventStore(db *sql.DB) *SQLiteEventStore {
	return &SQLiteEventStore{db: db}
}

func (s *SQLiteEventStore) Save(ctx context.Context, event unit.Event) error {
	payload, err := json.Marshal(event.Payload())
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO events (id, type, domain, correlation_id, payload, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`, generateID(), event.Type(), event.Domain(), event.CorrelationID(), payload, event.Timestamp().Unix())

	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (s *SQLiteEventStore) Query(ctx context.Context, filter EventQueryFilter) ([]unit.Event, error) {
	query := "SELECT id, type, domain, correlation_id, payload, timestamp FROM events WHERE 1=1"
	args := []any{}

	if filter.Domain != "" {
		query += " AND domain = ?"
		args = append(args, filter.Domain)
	}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.CorrelationID != "" {
		query += " AND correlation_id = ?"
		args = append(args, filter.CorrelationID)
	}
	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime.Unix())
	}
	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime.Unix())
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []unit.Event
	for rows.Next() {
		var e storedEvent
		var ts int64
		if err := rows.Scan(&e.id, &e.eventType, &e.domain, &e.correlationID, &e.payload, &ts); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.timestamp = time.Unix(ts, 0)
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

func (s *SQLiteEventStore) GetByID(ctx context.Context, id string) (unit.Event, error) {
	var e storedEvent
	var ts int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, domain, correlation_id, payload, timestamp 
		FROM events WHERE id = ?
	`, id).Scan(&e.id, &e.eventType, &e.domain, &e.correlationID, &e.payload, &ts)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get event by id: %w", err)
	}

	e.timestamp = time.Unix(ts, 0)
	return &e, nil
}

func (s *SQLiteEventStore) SaveBatch(ctx context.Context, events []unit.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (id, type, domain, correlation_id, payload, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		payload, err := json.Marshal(event.Payload())
		if err != nil {
			return fmt.Errorf("marshal event payload: %w", err)
		}

		_, err = stmt.ExecContext(ctx, generateID(), event.Type(), event.Domain(), event.CorrelationID(), payload, event.Timestamp().Unix())
		if err != nil {
			return fmt.Errorf("insert event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
