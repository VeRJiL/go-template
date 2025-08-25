package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTService(t *testing.T) {
	t.Run("should create JWT service with correct configuration", func(t *testing.T) {
		secret := "test-secret-key"
		expiration := 3600 // 1 hour

		service := NewJWTService(secret, expiration)

		assert.NotNil(t, service)
		assert.Equal(t, []byte(secret), service.secret)
		assert.Equal(t, time.Duration(expiration)*time.Second, service.expiration)
	})

	t.Run("should handle different expiration values", func(t *testing.T) {
		tests := []struct {
			name       string
			expiration int
			expected   time.Duration
		}{
			{"1 second", 1, 1 * time.Second},
			{"1 minute", 60, 60 * time.Second},
			{"1 hour", 3600, 3600 * time.Second},
			{"1 day", 86400, 86400 * time.Second},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				service := NewJWTService("secret", tt.expiration)
				assert.Equal(t, tt.expected, service.expiration)
			})
		}
	})
}

func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 3600)
	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	t.Run("should generate valid token", func(t *testing.T) {
		token, expiresAt, err := service.GenerateToken(userID, email, role)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, expiresAt.After(time.Now()))

		// Token should have 3 parts (header.payload.signature)
		parts := strings.Split(token, ".")
		assert.Len(t, parts, 3)
	})

	t.Run("should set correct expiration time", func(t *testing.T) {
		beforeGeneration := time.Now()
		token, expiresAt, err := service.GenerateToken(userID, email, role)
		afterGeneration := time.Now()

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		expectedExpiration := beforeGeneration.Add(service.expiration)
		timeDiff := expiresAt.Sub(expectedExpiration)

		// Should be within 1 second of expected time
		assert.True(t, timeDiff >= -time.Second && timeDiff <= time.Second)
		assert.True(t, expiresAt.After(afterGeneration))
	})

	t.Run("should include correct claims", func(t *testing.T) {
		token, _, err := service.GenerateToken(userID, email, role)
		require.NoError(t, err)

		// Parse token to verify claims
		parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return service.secret, nil
		})
		require.NoError(t, err)

		claims, ok := parsedToken.Claims.(*Claims)
		require.True(t, ok)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.NotNil(t, claims.ExpiresAt)
		assert.NotNil(t, claims.IssuedAt)
		assert.NotNil(t, claims.NotBefore)
	})

	t.Run("should generate different tokens for different inputs", func(t *testing.T) {
		token1, _, err1 := service.GenerateToken(userID, email, role)
		token2, _, err2 := service.GenerateToken(uuid.New(), "different@example.com", "admin")

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, token1, token2)
	})

	t.Run("should generate different tokens each time", func(t *testing.T) {
		token1, _, err1 := service.GenerateToken(userID, email, role)
		time.Sleep(1 * time.Second) // Ensure different timestamp (JWT uses second precision)
		token2, _, err2 := service.GenerateToken(userID, email, role)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, token1, token2) // Different due to IssuedAt timestamp
	})
}

func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 3600)
	userID := uuid.New()
	email := "test@example.com"
	role := "admin"

	t.Run("should validate valid token", func(t *testing.T) {
		token, _, err := service.GenerateToken(userID, email, role)
		require.NoError(t, err)

		claims, err := service.ValidateToken(token)
		require.NoError(t, err)
		require.NotNil(t, claims)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("should reject invalid token format", func(t *testing.T) {
		invalidTokens := []string{
			"invalid-token",
			"not.a.jwt",
			"",
			"header.payload", // Missing signature
			"too.many.parts.here.invalid",
		}

		for _, token := range invalidTokens {
			claims, err := service.ValidateToken(token)
			assert.Error(t, err)
			assert.Nil(t, claims)
			assert.Contains(t, err.Error(), "failed to parse token")
		}
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		wrongSecretService := NewJWTService("wrong-secret", 3600)
		token, _, err := service.GenerateToken(userID, email, role)
		require.NoError(t, err)

		claims, err := wrongSecretService.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "failed to parse token")
	})

	t.Run("should reject expired token", func(t *testing.T) {
		shortLivedService := NewJWTService("test-secret-key", -1) // Already expired
		token, _, err := shortLivedService.GenerateToken(userID, email, role)
		require.NoError(t, err)

		claims, err := shortLivedService.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "failed to parse token")
	})

	t.Run("should reject token with wrong signing method", func(t *testing.T) {
		// Create token with RS256 instead of HS256
		claims := Claims{
			UserID: userID,
			Email:  email,
			Role:   role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		// This would normally require RSA keys, but we'll create a malformed token
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		validatedClaims, err := service.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, validatedClaims)
	})
}

func TestJWTService_RefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 300) // 5 minutes
	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	t.Run("should refresh token close to expiration", func(t *testing.T) {
		// Create a service with very short expiration to test refresh
		shortService := NewJWTService("test-secret-key", 250) // 4 minutes 10 seconds
		originalToken, _, err := shortService.GenerateToken(userID, email, role)
		require.NoError(t, err)

		// Wait a moment to ensure token is generated, then sleep to get closer to expiry
		time.Sleep(1 * time.Second)

		// Should be eligible for refresh (within 5 minutes of expiration)
		newToken, newExpiresAt, err := shortService.RefreshToken(originalToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, originalToken, newToken)
		assert.True(t, newExpiresAt.After(time.Now()))

		// Verify new token has same user info
		claims, err := shortService.ValidateToken(newToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("should reject refresh for token not close to expiration", func(t *testing.T) {
		longLivedService := NewJWTService("test-secret-key", 7200) // 2 hours
		token, _, err := longLivedService.GenerateToken(userID, email, role)
		require.NoError(t, err)

		newToken, newExpiresAt, err := longLivedService.RefreshToken(token)
		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.True(t, newExpiresAt.IsZero())
		assert.Contains(t, err.Error(), "not eligible for refresh")
	})

	t.Run("should reject refresh for invalid token", func(t *testing.T) {
		invalidToken := "invalid-token"

		newToken, newExpiresAt, err := service.RefreshToken(invalidToken)
		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.True(t, newExpiresAt.IsZero())
	})

	t.Run("should reject refresh for expired token", func(t *testing.T) {
		expiredService := NewJWTService("test-secret-key", -1) // Already expired
		expiredToken, _, err := expiredService.GenerateToken(userID, email, role)
		require.NoError(t, err)

		newToken, newExpiresAt, err := expiredService.RefreshToken(expiredToken)
		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.True(t, newExpiresAt.IsZero())
	})
}

func TestClaims(t *testing.T) {
	t.Run("should create claims with all required fields", func(t *testing.T) {
		userID := uuid.New()
		email := "test@example.com"
		role := "admin"
		now := time.Now()

		claims := Claims{
			UserID: userID,
			Email:  email,
			Role:   role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
			},
		}

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.NotNil(t, claims.ExpiresAt)
		assert.NotNil(t, claims.IssuedAt)
		assert.NotNil(t, claims.NotBefore)
	})
}

func TestJWTService_EdgeCases(t *testing.T) {
	t.Run("should handle empty secret", func(t *testing.T) {
		service := NewJWTService("", 3600)
		userID := uuid.New()

		token, _, err := service.GenerateToken(userID, "test@example.com", "user")
		require.NoError(t, err)

		// Should still validate with same empty secret
		claims, err := service.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("should handle very long secret", func(t *testing.T) {
		longSecret := strings.Repeat("a", 1000)
		service := NewJWTService(longSecret, 3600)
		userID := uuid.New()

		token, _, err := service.GenerateToken(userID, "test@example.com", "user")
		require.NoError(t, err)

		claims, err := service.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("should handle special characters in user data", func(t *testing.T) {
		service := NewJWTService("test-secret", 3600)
		userID := uuid.New()
		specialEmail := "test+tag@sub.domain.com"
		specialRole := "admin-super"

		token, _, err := service.GenerateToken(userID, specialEmail, specialRole)
		require.NoError(t, err)

		claims, err := service.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, specialEmail, claims.Email)
		assert.Equal(t, specialRole, claims.Role)
	})

	t.Run("should handle zero expiration", func(t *testing.T) {
		service := NewJWTService("test-secret", 0)
		assert.Equal(t, time.Duration(0), service.expiration)

		userID := uuid.New()
		token, expiresAt, err := service.GenerateToken(userID, "test@example.com", "user")
		require.NoError(t, err)

		// Token should be immediately expired
		assert.True(t, expiresAt.Before(time.Now()) || expiresAt.Equal(time.Now()))

		// Validation should fail due to expiration
		claims, err := service.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestJWTService_Integration(t *testing.T) {
	t.Run("should handle complete token lifecycle", func(t *testing.T) {
		service := NewJWTService("integration-test-secret", 600) // 10 minutes
		userID := uuid.New()
		email := "integration@example.com"
		role := "user"

		// Generate token
		token, expiresAt, err := service.GenerateToken(userID, email, role)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, expiresAt.After(time.Now()))

		// Validate token
		claims, err := service.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)

		// Try to refresh (should fail - not close to expiration, 10 minutes > 5 minutes)
		newToken, newExpiresAt, err := service.RefreshToken(token)
		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.True(t, newExpiresAt.IsZero())
		assert.Contains(t, err.Error(), "not eligible for refresh")

		// Original token should still be valid
		claims2, err := service.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, claims.UserID, claims2.UserID)
	})
}