package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action performed
type AuditAction string

const (
	AuditActionLogin        AuditAction = "LOGIN"
	AuditActionLogout       AuditAction = "LOGOUT"
	AuditActionRegister     AuditAction = "REGISTER"
	AuditActionPasswordReset AuditAction = "PASSWORD_RESET"
	AuditActionProfileUpdate AuditAction = "PROFILE_UPDATE"
	AuditActionCreate       AuditAction = "CREATE"
	AuditActionUpdate       AuditAction = "UPDATE"
	AuditActionDelete       AuditAction = "DELETE"
	AuditActionView         AuditAction = "VIEW"
	AuditActionSearch       AuditAction = "SEARCH"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID       `db:"id"`
	UserID       *uuid.UUID      `db:"user_id"`
	Action       string          `db:"action"`
	ResourceType *string         `db:"resource_type"`
	ResourceID   *uuid.UUID      `db:"resource_id"`
	IPAddress    *string         `db:"ip_address"`
	UserAgent    *string         `db:"user_agent"`
	Metadata     json.RawMessage `db:"metadata"`
	CreatedAt    time.Time       `db:"created_at"`
}

// AuditLogCreate represents data needed to create an audit log entry
type AuditLogCreate struct {
	UserID       *uuid.UUID
	Action       string
	ResourceType *string
	ResourceID   *uuid.UUID
	IPAddress    *string
	UserAgent    *string
	Metadata     interface{}
}

// AuditRepository defines the interface for audit log operations
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLogCreate) error
	List(ctx context.Context, userID *uuid.UUID, action *string, limit, offset int) ([]*AuditLog, error)
	Count(ctx context.Context, userID *uuid.UUID, action *string) (int, error)
	DeleteOldLogs(ctx context.Context, olderThan time.Time) error
}

// auditRepository implements AuditRepository
type auditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) AuditRepository {
	return &auditRepository{db: db}
}

// Create creates a new audit log entry
func (r *auditRepository) Create(ctx context.Context, log *AuditLogCreate) error {
	// Convert metadata to JSON
	var metadataJSON json.RawMessage
	if log.Metadata != nil {
		bytes, err := json.Marshal(log.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = bytes
	} else {
		metadataJSON = json.RawMessage("{}")
	}

	query := `
		INSERT INTO auth.audit_logs (
			id, user_id, action, resource_type, resource_id,
			ip_address, user_agent, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	id := uuid.New()
	_, err := r.db.ExecContext(ctx, query,
		id,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.IPAddress,
		log.UserAgent,
		metadataJSON,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// List retrieves audit logs with filtering and pagination
func (r *auditRepository) List(ctx context.Context, userID *uuid.UUID, action *string, limit, offset int) ([]*AuditLog, error) {
	query := `
		SELECT 
			id, user_id, action, resource_type, resource_id,
			ip_address, user_agent, metadata, created_at
		FROM auth.audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *userID)
		argIndex++
	}

	if action != nil {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, *action)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.IPAddress,
			&log.UserAgent,
			&log.Metadata,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return logs, nil
}

// Count returns the number of audit logs matching the filters
func (r *auditRepository) Count(ctx context.Context, userID *uuid.UUID, action *string) (int, error) {
	query := "SELECT COUNT(*) FROM auth.audit_logs WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *userID)
		argIndex++
	}

	if action != nil {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, *action)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// DeleteOldLogs deletes audit logs older than the specified time
func (r *auditRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) error {
	query := `
		DELETE FROM auth.audit_logs
		WHERE created_at < $1
	`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the cleanup (optional)
	if rowsAffected > 0 {
		fmt.Printf("Deleted %d old audit logs\n", rowsAffected)
	}

	return nil
}