package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/didok362/messenger/internal/domain"
	"github.com/didok362/messenger/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo эмулирует базу данных пользователей в тестах
type mockUserRepo struct {
	createFn        func(ctx context.Context, user *domain.User) error
	getByEmailFn    func(ctx context.Context, email string) (*domain.User, error)
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	getByUsernameFn func(ctx context.Context, username string) (*domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, domain.ErrUserNotFound
}

const (
	testJWTSecret = "test-secret-key-12345"
	testTokenTTL  = 2 * time.Hour
)

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := &mockUserRepo{
			getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound // email свободен
			},
			createFn: func(ctx context.Context, user *domain.User) error {
				user.ID = uuid.New()
				return nil
			},
		}

		authService := service.NewAuthService(repo, testJWTSecret, testTokenTTL)
		user, err := authService.Register(ctx, "vasya", "vasya@test.com", "secret123")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.Username != "vasya" || user.Email != "vasya@test.com" {
			t.Errorf("unexpected user data: %+v", user)
		}
		// Проверяем, что пароль реально захэширован bcrypt
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("secret123")); err != nil {
			t.Errorf("password hash does not match original password: %v", err)
		}
	})

	t.Run("user_already_exists", func(t *testing.T) {
		repo := &mockUserRepo{
			getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
				return &domain.User{ID: uuid.New(), Email: email}, nil // email занят
			},
		}

		authService := service.NewAuthService(repo, testJWTSecret, testTokenTTL)
		_, err := authService.Register(ctx, "vasya", "vasya@test.com", "secret123")

		if !errors.Is(err, domain.ErrUserAlreadyExists) {
			t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
		}
	})

	t.Run("database_error_on_create", func(t *testing.T) {
		repo := &mockUserRepo{
			getByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
			createFn: func(ctx context.Context, user *domain.User) error {
				return errors.New("db connection lost")
			},
		}

		authService := service.NewAuthService(repo, testJWTSecret, testTokenTTL)
		_, err := authService.Register(ctx, "vasya", "vasya@test.com", "secret123")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("secret123"), 12)
	userID := uuid.New()

	existingUser := &domain.User{
		ID:           userID,
		Username:     "vasya",
		Email:        "vasya@test.com",
		PasswordHash: string(hashedPassword),
	}

	t.Run("success", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*domain.User, error) {
				return existingUser, nil
			},
		}

		authService := service.NewAuthService(repo, testJWTSecret, testTokenTTL)
		tokenStr, user, err := authService.Login(ctx, "vasya", "secret123")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.ID != userID {
			t.Errorf("expected user ID %s, got %s", userID, user.ID)
		}

		// Проверяем валидность созданного JWT-токена
		parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(testJWTSecret), nil
		})
		if err != nil || !parsedToken.Valid {
			t.Fatalf("generated token is invalid: %v", err)
		}
		claims := parsedToken.Claims.(jwt.MapClaims)
		if claims["sub"] != userID.String() {
			t.Errorf("expected sub claim %s, got %s", userID.String(), claims["sub"])
		}
	})

	t.Run("wrong_password", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*domain.User, error) {
				return existingUser, nil
			},
		}

		authService := service.NewAuthService(repo, testJWTSecret, testTokenTTL)
		_, _, err := authService.Login(ctx, "vasya", "WRONG_PASSWORD")

		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		authService := service.NewAuthService(repo, testJWTSecret, testTokenTTL)
		_, _, err := authService.Login(ctx, "unknown_user", "secret123")

		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}
