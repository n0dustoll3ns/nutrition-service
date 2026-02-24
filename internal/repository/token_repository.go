package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/model"
)

// TokenRepository defines the interface for token data operations
type TokenRepository interface {
	// Refresh tokens
	CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	GetRefreshTokenByUserAndDevice(ctx context.Context, userID uuid.UUID, deviceInfo *string) (*model.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error
	RevokeAllRefreshTokensForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredRefreshTokens(ctx context.Context) error
	
	// Password reset tokens
	CreatePasswordResetToken(ctx context.Context, token *model.PasswordResetToken) error
	GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	DeleteExpiredPasswordResetTokens(ctx context.Context) error
	
	// Revoked access tokens
	CreateRevokedAccessToken(ctx context.Context, token *model.RevokedAccessToken) error
	IsAccessTokenRevoked(ctx context.Context, tokenID string) (bool, error)
	DeleteExpiredRevokedTokens(ctx context.Context) error
}

// tokenRepository implements TokenRepository
type tokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) TokenRepository {
	return &tokenRepository{db: db}
}

// CreateRefreshToken creates a new refresh token
func (r *tokenRepository) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO auth.refresh_tokens (
			id, user_id, token_hash, device_info, ip_address, expires_at, revoked_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.DeviceInfo,
		token.IPAddress,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *tokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT 
			id, user_id, token_hash, device_info, ip_address, 
			expires_at, revoked_at, created_at
		FROM auth.refresh_tokens
		WHERE token_hash = $1
	`

	token := &model.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.DeviceInfo,
		&token.IPAddress,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}

	return token, nil
}

// GetRefreshTokenByUserAndDevice retrieves a refresh token by user ID and device info
func (r *tokenRepository) GetRefreshTokenByUserAndDevice(ctx context.Context, userID uuid.UUID, deviceInfo *string) (*model.RefreshToken, error) {
	var query string
	var err error
	token := &model.RefreshToken{}

	if deviceInfo == nil {
		query = `
			SELECT 
				id, user_id, token_hash, device_info, ip_address, 
				expires_at, revoked_at, created_at
			FROM auth.refresh_tokens
			WHERE user_id = $1 AND device_info IS NULL
		`
		err = r.db.QueryRowContext(ctx, query, userID).Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.DeviceInfo,
			&token.IPAddress,
			&token.ExpiresAt,
			&token.RevokedAt,
			&token.CreatedAt,
		)
	} else {
		query = `
			SELECT 
				id, user_id, token_hash, device_info, ip_address, 
				expires_at, revoked_at, created_at
			FROM auth.refresh_tokens
			WHERE user_id = $1 AND device_info = $2
		`
		err = r.db.QueryRowContext(ctx, query, userID, deviceInfo).Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.DeviceInfo,
			&token.IPAddress,
			&token.ExpiresAt,
			&token.RevokedAt,
			&token.CreatedAt,
		)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token by user and device: %w", err)
	}

	return token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *tokenRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE auth.refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found or already revoked")
	}

	return nil
}

// RevokeAllRefreshTokensForUser revokes all refresh tokens for a user
func (r *tokenRepository) RevokeAllRefreshTokensForUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE auth.refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all refresh tokens for user: %w", err)
	}

	return nil
}

// DeleteExpiredRefreshTokens deletes expired refresh tokens
func (r *tokenRepository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	query := `
		DELETE FROM auth.refresh_tokens
		WHERE expires_at < NOW() OR (revoked_at IS NOT NULL AND revoked_at < NOW() - INTERVAL '30 days')
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	return nil
}

// CreatePasswordResetToken creates a new password reset token
func (r *tokenRepository) CreatePasswordResetToken(ctx context.Context, token *model.PasswordResetToken) error {
	query := `
		INSERT INTO auth.password_reset_tokens (
			id, user_id, token_hash, expires_at, used_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.UsedAt,
		token.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	return nil
}

// GetPasswordResetTokenByHash retrieves a password reset token by its hash
func (r *tokenRepository) GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error) {
	query := `
		SELECT 
			id, user_id, token_hash, expires_at, used_at, created_at
		FROM auth.password_reset_tokens
		WHERE token_hash = $1
	`

	token := &model.PasswordResetToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("password reset token not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get password reset token by hash: %w", err)
	}

	return token, nil
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (r *tokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE auth.password_reset_tokens
		SET used_at = NOW()
		WHERE id = $1 AND used_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("failed to mark password reset token as used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("password reset token not found or already used")
	}

	return nil
}

// DeleteExpiredPasswordResetTokens deletes expired password reset tokens
func (r *tokenRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	query := `
		DELETE FROM auth.password_reset_tokens
		WHERE expires_at < NOW() OR (used_at IS NOT NULL AND used_at < NOW() - INTERVAL '7 days')
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired password reset tokens: %w", err)
	}

	return nil
}

// CreateRevokedAccessToken creates a new revoked access token record
func (r *tokenRepository) CreateRevokedAccessToken(ctx context.Context, token *model.RevokedAccessToken) error {
	query := `
		INSERT INTO auth.revoked_access_tokens (
			id, token_id, user_id, expires_at, revoked_at, reason
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.TokenID,
		token.UserID,
		token.ExpiresAt,
		token.RevokedAt,
		token.Reason,
	)

	if err != nil {
		return fmt.Errorf("failed to create revoked access token: %w", err)
	}

	return nil
}

// IsAccessTokenRevoked checks if an access token is revoked
func (r *tokenRepository) IsAccessTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM auth.revoked_access_tokens 
			WHERE token_id = $1 AND expires_at > NOW()
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tokenID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if access token is revoked: %w", err)
	}

	return exists, nil
}

// DeleteExpiredRevokedTokens deletes expired revoked tokens
func (r *tokenRepository) DeleteExpiredRevokedTokens(ctx context.Context) error {
	query := `
		DELETE FROM auth.revoked_access_tokens
		WHERE expires_at < NOW()
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired revoked tokens: %w", err)
	}

	return nil
}