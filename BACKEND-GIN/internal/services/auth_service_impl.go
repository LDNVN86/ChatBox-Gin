package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"chatbox-gin/internal/auth"
	apperrors "chatbox-gin/internal/errors"
	"chatbox-gin/internal/models"
	"chatbox-gin/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ===========================================================================
// Auth Service Implementation
// ===========================================================================

// authServiceImpl implements AuthService
type authServiceImpl struct {
	userRepo   repositories.UserRepository
	jwtService *auth.JWTService
	logger     *zap.Logger
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userRepo repositories.UserRepository,
	jwtService *auth.JWTService,
	logger *zap.Logger,
) AuthService {
	return &authServiceImpl{
		userRepo:   userRepo,
		jwtService: jwtService,
		logger:     logger,
	}
}

// hashToken creates SHA256 hash of token for storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Login authenticates user with email and password
func (s *authServiceImpl) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	// Find user by email (uuid.Nil = global search)
	user, err := s.userRepo.FindByEmail(ctx, uuid.Nil, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrInvalidCredentials
		}
		s.logger.Error("find user by email failed",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	// Verify password
	if !user.CheckPassword(password) {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := s.jwtService.GenerateTokenPair(user)
	if err != nil {
		s.logger.Error("generate token failed",
			zap.Error(err),
			zap.String("user_id", user.ID.String()),
		)
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Hash và lưu refresh token vào DB
	tokenHash := hashToken(tokens.RefreshToken)
	user.RefreshTokenHash = &tokenHash
	user.UpdateLastSeen()

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("save refresh token hash failed",
			zap.Error(err),
			zap.String("user_id", user.ID.String()),
		)
		// Không return error, vẫn cho đăng nhập nhưng log warning
	}

	s.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email),
	)

	return &LoginResult{
		User: user,
		Tokens: &TokenPair{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresIn:    900, // 15 minutes
		},
	}, nil
}

// RefreshTokens generates new token pair using refresh token
func (s *authServiceImpl) RefreshTokens(ctx context.Context, refreshToken string) (*LoginResult, error) {
	// Validate refresh token JWT
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		if err == auth.ErrExpiredToken {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}

	// Validate refresh token hash với DB
	tokenHash := hashToken(refreshToken)
	if user.RefreshTokenHash == nil || *user.RefreshTokenHash != tokenHash {
		s.logger.Warn("refresh token hash mismatch - token possibly revoked",
			zap.String("user_id", user.ID.String()),
		)
		return nil, apperrors.ErrInvalidToken
	}

	// Generate new tokens
	tokens, err := s.jwtService.GenerateTokenPair(user)
	if err != nil {
		s.logger.Error("generate token failed",
			zap.Error(err),
			zap.String("user_id", user.ID.String()),
		)
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Token rotation: Update hash với token mới
	newTokenHash := hashToken(tokens.RefreshToken)
	user.RefreshTokenHash = &newTokenHash

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("update refresh token hash failed",
			zap.Error(err),
			zap.String("user_id", user.ID.String()),
		)
	}

	return &LoginResult{
		User: user,
		Tokens: &TokenPair{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresIn:    900,
		},
	}, nil
}

// ValidateAccessToken validates access token and returns claims
func (s *authServiceImpl) ValidateAccessToken(token string) (*Claims, error) {
	jwtClaims, err := s.jwtService.ValidateAccessToken(token)
	if err != nil {
		if err == auth.ErrExpiredToken {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	return &Claims{
		UserID:      jwtClaims.UserID,
		WorkspaceID: jwtClaims.WorkspaceID,
		Email:       jwtClaims.Email,
		Role:        jwtClaims.Role,
	}, nil
}

// ValidateRefreshToken validates refresh token and returns claims
func (s *authServiceImpl) ValidateRefreshToken(token string) (*Claims, error) {
	jwtClaims, err := s.jwtService.ValidateRefreshToken(token)
	if err != nil {
		if err == auth.ErrExpiredToken {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	return &Claims{
		UserID:      jwtClaims.UserID,
		WorkspaceID: jwtClaims.WorkspaceID,
		Email:       jwtClaims.Email,
		Role:        jwtClaims.Role,
	}, nil
}

// GetUserByID gets user by ID
func (s *authServiceImpl) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

// RevokeRefreshToken invalidates refresh token (for logout)
func (s *authServiceImpl) RevokeRefreshToken(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.ErrNotFound
	}

	// Clear refresh token hash
	user.RefreshTokenHash = nil
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	s.logger.Info("refresh token revoked",
		zap.String("user_id", userID.String()),
	)

	return nil
}

