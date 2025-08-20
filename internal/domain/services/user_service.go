package services

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/VeRJiL/go-template/internal/domain/entities"
	"github.com/VeRJiL/go-template/internal/domain/repositories"
	"github.com/VeRJiL/go-template/internal/pkg/auth"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserService struct {
	userRepo      repositories.UserRepository
	userCacheRepo repositories.UserCacheRepository
	jwtService    *auth.JWTService
}

func NewUserService(
	userRepo repositories.UserRepository,
	jwtService *auth.JWTService,
) *UserService {
	return &UserService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (s *UserService) SetCacheRepository(cacheRepo repositories.UserCacheRepository) {
	s.userCacheRepo = cacheRepo
}

func (s *UserService) Create(ctx context.Context, req *entities.CreateUserRequest) (*entities.User, error) {
	existingUser, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &entities.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
	}
	user.BeforeCreate()

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.invalidateUserListCache(ctx)

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *entities.UpdateUserRequest) (*entities.User, error) {
	_, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	updatedUser, err := s.userRepo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.invalidateUserListCache(ctx)

	return updatedUser, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.invalidateUserListCache(ctx)

	return nil
}

func (s *UserService) List(ctx context.Context, offset, limit int) ([]*entities.User, int, error) {
	cacheKey := s.generateListCacheKey(offset, limit)

	if s.userCacheRepo != nil {
		if cachedData := s.getCachedUserList(ctx, cacheKey); cachedData != nil {
			return cachedData.Users, cachedData.Total, nil
		}
	}

	users, total, err := s.userRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	if s.userCacheRepo != nil {
		s.cacheUserList(ctx, cacheKey, users, total)
	}

	return users, total, nil
}

func (s *UserService) Login(ctx context.Context, req *entities.LoginRequest) (*entities.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	token, expiresAt, err := s.jwtService.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &entities.LoginResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *UserService) Logout(ctx context.Context, token string) error {
	return nil
}

func (s *UserService) Search(ctx context.Context, query string, offset, limit int) ([]*entities.User, int, error) {
	return s.userRepo.Search(ctx, query, offset, limit)
}

type UserListCacheData struct {
	Users []*entities.User `json:"users"`
	Total int              `json:"total"`
}

func (s *UserService) generateListCacheKey(offset, limit int) string {
	key := fmt.Sprintf("users:list:offset:%d:limit:%d", offset, limit)
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("users_list_%x", hash)
}

func (s *UserService) getCachedUserList(ctx context.Context, cacheKey string) *UserListCacheData {
	var cacheData UserListCacheData
	if err := s.userCacheRepo.GetJSON(ctx, cacheKey, &cacheData); err != nil {
		return nil
	}
	return &cacheData
}

func (s *UserService) cacheUserList(ctx context.Context, cacheKey string, users []*entities.User, total int) {
	cacheData := UserListCacheData{
		Users: users,
		Total: total,
	}

	s.userCacheRepo.SetJSON(ctx, cacheKey, cacheData)
}

func (s *UserService) invalidateUserListCache(ctx context.Context) {
	if s.userCacheRepo == nil {
		return
	}
	s.userCacheRepo.DeletePattern(ctx, "users_list_*")
}
