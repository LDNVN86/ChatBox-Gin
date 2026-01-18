package services

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
)

// ===========================================================================
// Auth Service Interface
// Handle authentication: login, refresh, token validation
// ===========================================================================

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
}

// LoginResult result of login operation
type LoginResult struct {
	User   *models.User
	Tokens *TokenPair
}

// Claims extracted token claims
type Claims struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
	Role        models.UserRole
}

// AuthService interface for authentication operations
type AuthService interface {
	// Login authenticates user with email and password
	// Returns user and tokens if successful
	Login(ctx context.Context, email, password string) (*LoginResult, error)

	// RefreshTokens generate new token pair using refresh token
	RefreshTokens(ctx context.Context, refreshToken string) (*LoginResult, error)

	// ValidateAccessToken validates access token and returns claims
	ValidateAccessToken(token string) (*Claims, error)

	// ValidateRefreshToken validates refresh token and returns claims
	ValidateRefreshToken(token string) (*Claims, error)

	// GetUserByID gets user by ID
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)

	// RevokeRefreshToken invalidates refresh token (for logout)
	RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error
}
