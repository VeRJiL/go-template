package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	secret     []byte
	expiration time.Duration
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string, expiration int) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		expiration: time.Duration(expiration) * time.Second,
	}
}

func (s *JWTService) GenerateToken(userID uuid.UUID, email, role string) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.expiration)

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *JWTService) RefreshToken(tokenString string) (string, time.Time, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", time.Time{}, err
	}

	// Check if token is close to expiration (within 5 minutes)
	if time.Until(claims.ExpiresAt.Time) > 5*time.Minute {
		return "", time.Time{}, fmt.Errorf("token is not eligible for refresh")
	}

	return s.GenerateToken(claims.UserID, claims.Email, claims.Role)
}
