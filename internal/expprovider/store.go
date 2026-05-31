package expprovider

// store.go — Experimental Provider Registry Persistence (P1).
//
// This is the persistence layer for the Experimental Provider lifecycle. It
// stores admission results, lifecycle state transitions, validation evidence,
// and descriptor snapshots so they survive restarts and are queryable by the
// API (P2) and dashboard (P3).
//
// DESIGN PRINCIPLES:
//   - Store wraps AROUND AdmitProvider output, not inside it. framework.go is
//     unchanged (boundary: no framework modification).
//   - Credentials are NEVER stored (Invariant 3). Only env var NAMES are
//     persisted (from the descriptor snapshot). Values are read live from env.
//   - Admission report is immutable after write (append-only audit semantics).
//   - Descriptor snapshot is captured at admission time (not a pointer to code).
//   - Single table, JSON columns for nested data (4-10 providers, normalization
//     is overkill).

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ProviderRecord is the persistent representation of an Experimental provider's
// lifecycle. It maps 1:1 onto the `experimental_providers` table row.
type ProviderRecord struct {
	Name                string          `json:"name"`
	Track               string          `json:"track"`
	State               string          `json:"state"`
	DescriptorJSON      json.RawMessage `json:"descriptor"`
	AdmissionReportJSON json.RawMessage `json:"admission_report"`
	AdmittedAt          *time.Time      `json:"admitted_at"`
	ActivatedAt         *time.Time      `json:"activated_at"`
	DeactivatedAt       *time.Time      `json:"deactivated_at"`
	ValidationEvidence  string          `json:"validation_evidence"`
	RiskBadge           string          `json:"risk_badge"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// Store is the persistence interface for Experimental provider records. It is
// consumed by the API layer (P2) and decouples persistence from the admission
// engine (framework.go is never modified).
type Store interface {
	// Save persists a provider record (upsert semantics: insert or update).
	Save(ctx context.Context, rec *ProviderRecord) error
	// Get retrieves a single provider record by name. Returns nil, nil if not found.
	Get(ctx context.Context, name string) (*ProviderRecord, error)
	// List returns all provider records ordered by name.
	List(ctx context.Context) ([]*ProviderRecord, error)
	// UpdateState transitions the lifecycle state and sets the appropriate timestamp.
	UpdateState(ctx context.Context, name string, state string, at time.Time) error
	// SetValidationEvidence updates the validation evidence field.
	SetValidationEvidence(ctx context.Context, name string, evidence string) error
}

// SQLiteStore implements Store backed by the application's SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a Store backed by the given database connection.
// The caller is responsible for ensuring the experimental_providers table exists
// (via the migration in db.go).
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Save persists a provider record with upsert semantics.
func (s *SQLiteStore) Save(ctx context.Context, rec *ProviderRecord) error {
	if rec == nil {
		return fmt.Errorf("expprovider: cannot save nil record")
	}
	now := time.Now().UTC()
	rec.UpdatedAt = now
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}

	query := `
		INSERT INTO experimental_providers (
			name, track, state, descriptor_json, admission_report_json,
			admitted_at, activated_at, deactivated_at, validation_evidence,
			risk_badge, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			track = excluded.track,
			state = excluded.state,
			descriptor_json = excluded.descriptor_json,
			admission_report_json = excluded.admission_report_json,
			admitted_at = excluded.admitted_at,
			activated_at = excluded.activated_at,
			deactivated_at = excluded.deactivated_at,
			validation_evidence = excluded.validation_evidence,
			risk_badge = excluded.risk_badge,
			updated_at = excluded.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		rec.Name, rec.Track, rec.State,
		string(rec.DescriptorJSON), string(rec.AdmissionReportJSON),
		timePtr(rec.AdmittedAt), timePtr(rec.ActivatedAt), timePtr(rec.DeactivatedAt),
		rec.ValidationEvidence, rec.RiskBadge,
		rec.CreatedAt.Format(time.RFC3339), rec.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// Get retrieves a provider record by name. Returns nil, nil if not found.
func (s *SQLiteStore) Get(ctx context.Context, name string) (*ProviderRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT name, track, state, descriptor_json, admission_report_json,
		       admitted_at, activated_at, deactivated_at, validation_evidence,
		       risk_badge, created_at, updated_at
		FROM experimental_providers WHERE name = ?
	`, name)
	return scanRecord(row)
}

// List returns all provider records ordered by name.
func (s *SQLiteStore) List(ctx context.Context) ([]*ProviderRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, track, state, descriptor_json, admission_report_json,
		       admitted_at, activated_at, deactivated_at, validation_evidence,
		       risk_badge, created_at, updated_at
		FROM experimental_providers ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*ProviderRecord
	for rows.Next() {
		rec, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// UpdateState transitions the lifecycle state and sets the appropriate timestamp.
func (s *SQLiteStore) UpdateState(ctx context.Context, name string, state string, at time.Time) error {
	ts := at.UTC().Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)

	var query string
	switch state {
	case "admitted":
		query = `UPDATE experimental_providers SET state = ?, admitted_at = ?, updated_at = ? WHERE name = ?`
	case "active":
		query = `UPDATE experimental_providers SET state = ?, activated_at = ?, updated_at = ? WHERE name = ?`
	case "deprecated", "retired":
		query = `UPDATE experimental_providers SET state = ?, deactivated_at = ?, updated_at = ? WHERE name = ?`
	default:
		query = `UPDATE experimental_providers SET state = ?, updated_at = ? WHERE name = ?`
		_, err := s.db.ExecContext(ctx, query, state, now, name)
		return err
	}
	_, err := s.db.ExecContext(ctx, query, state, ts, now, name)
	return err
}

// SetValidationEvidence updates the validation evidence field.
func (s *SQLiteStore) SetValidationEvidence(ctx context.Context, name string, evidence string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`UPDATE experimental_providers SET validation_evidence = ?, updated_at = ? WHERE name = ?`,
		evidence, now, name)
	return err
}

// --- helpers ---

func timePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return nil
	}
	return &t
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanRecord(row *sql.Row) (*ProviderRecord, error) {
	var rec ProviderRecord
	var descJSON, reportJSON sql.NullString
	var admittedAt, activatedAt, deactivatedAt, createdAt, updatedAt sql.NullString

	err := row.Scan(
		&rec.Name, &rec.Track, &rec.State,
		&descJSON, &reportJSON,
		&admittedAt, &activatedAt, &deactivatedAt,
		&rec.ValidationEvidence, &rec.RiskBadge,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if descJSON.Valid {
		rec.DescriptorJSON = json.RawMessage(descJSON.String)
	}
	if reportJSON.Valid {
		rec.AdmissionReportJSON = json.RawMessage(reportJSON.String)
	}
	rec.AdmittedAt = parseTime(admittedAt)
	rec.ActivatedAt = parseTime(activatedAt)
	rec.DeactivatedAt = parseTime(deactivatedAt)
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			rec.CreatedAt = t
		}
	}
	if updatedAt.Valid {
		if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
			rec.UpdatedAt = t
		}
	}
	return &rec, nil
}

func scanRows(rows *sql.Rows) (*ProviderRecord, error) {
	var rec ProviderRecord
	var descJSON, reportJSON sql.NullString
	var admittedAt, activatedAt, deactivatedAt, createdAt, updatedAt sql.NullString

	err := rows.Scan(
		&rec.Name, &rec.Track, &rec.State,
		&descJSON, &reportJSON,
		&admittedAt, &activatedAt, &deactivatedAt,
		&rec.ValidationEvidence, &rec.RiskBadge,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if descJSON.Valid {
		rec.DescriptorJSON = json.RawMessage(descJSON.String)
	}
	if reportJSON.Valid {
		rec.AdmissionReportJSON = json.RawMessage(reportJSON.String)
	}
	rec.AdmittedAt = parseTime(admittedAt)
	rec.ActivatedAt = parseTime(activatedAt)
	rec.DeactivatedAt = parseTime(deactivatedAt)
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			rec.CreatedAt = t
		}
	}
	if updatedAt.Valid {
		if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
			rec.UpdatedAt = t
		}
	}
	return &rec, nil
}
