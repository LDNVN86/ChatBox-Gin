package auth

import (
	"errors"
	"time"

	"chatbox-gin/internal/config"
	"chatbox-gin/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ===========================================================================
// JWT Service
// Generate and validate JWT tokens for authentication
// ===========================================================================

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims custom JWT claims
type Claims struct {
	UserID      uuid.UUID       `json:"user_id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	Email       string          `json:"email"`
	Role        models.UserRole `json:"role"`
	TokenType   string          `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair access và refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// JWTService xử lý JWT tokens
type JWTService struct {
	secret          []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
}

// NewJWTService tạo JWT service mới
func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		secret:          []byte(cfg.Secret),
		accessDuration:  cfg.AccessDuration,
		refreshDuration: cfg.RefreshDuration,
	}
}

// GenerateTokenPair tạo cặp access + refresh token cho user
func (s *JWTService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(s.accessDuration)
	refreshExp := now.Add(s.refreshDuration)

	// Access token
	accessClaims := Claims{
		UserID:      user.ID,
		WorkspaceID: user.WorkspaceID,
		Email:       user.Email,
		Role:        user.Role,
		TokenType:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshClaims := Claims{
		UserID:      user.ID,
		WorkspaceID: user.WorkspaceID,
		Email:       user.Email,
		Role:        user.Role,
		TokenType:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.secret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExp,
	}, nil
}

// ValidateToken validates token và trả về claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAccessToken validates access token
func (s *JWTService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates refresh token
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
