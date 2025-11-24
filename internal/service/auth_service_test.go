package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// mockRepo implements StaffRepoMinimal for tests.
type mockRepo struct {
	createFunc func(ctx context.Context, s *repository.Staff) error
	getFunc    func(ctx context.Context, username, hospitalID string) (*repository.Staff, error)
}

func (m *mockRepo) Create(ctx context.Context, s *repository.Staff) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, s)
	}
	return nil
}

func (m *mockRepo) GetByUsernameAndHospital(ctx context.Context, username, hospitalID string) (*repository.Staff, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, username, hospitalID)
	}
	return nil, nil
}

func TestRegister_Success(t *testing.T) {
	ctx := context.Background()
	called := false

	mock := &mockRepo{
		getFunc: func(ctx context.Context, username, hospitalID string) (*repository.Staff, error) {
			return nil, nil // not existing
		},
		createFunc: func(ctx context.Context, s *repository.Staff) error {
			called = true
			// ensure password hash looks like bcrypt hash
			assert.NotEmpty(t, s.PasswordHash)
			err := bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte("secretpass"))
			assert.NoError(t, err)
			return nil
		},
	}

	svc := NewAuthService(mock)
	st, err := svc.Register(ctx, "alice", "secretpass", "HIS-1", "Alice")
	assert.NoError(t, err)
	assert.NotNil(t, st)
	assert.True(t, called)
	assert.Equal(t, "alice", st.Username)
}

func TestRegister_WeakPassword(t *testing.T) {
	mock := &mockRepo{}
	svc := NewAuthService(mock)
	_, err := svc.Register(context.Background(), "bob", "123", "HIS-1", "Bob")
	assert.ErrorIs(t, err, ErrWeakPassword)
}

func TestAuthenticate_Success(t *testing.T) {
	// prepare repo returning existing staff with bcrypt hash
	pw := "mypassword"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)

	mock := &mockRepo{
		getFunc: func(ctx context.Context, username, hospitalID string) (*repository.Staff, error) {
			return &repository.Staff{
				ID:           "s1",
				Username:     "alice",
				PasswordHash: string(hashed),
				HospitalID:   "HIS-1",
				Role:         "staff",
			}, nil
		},
	}

	svc := NewAuthService(mock)
	secret := "test-secret"
	tokenStr, err := svc.Authenticate(context.Background(), "alice", pw, "HIS-1", secret, time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	// parse token to confirm claims
	parsed, err := jwt.Parse(tokenStr, func(tkn *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	if claims, ok := parsed.Claims.(jwt.MapClaims); assert.True(t, ok) {
		assert.Equal(t, "s1", claims["sub"])
		assert.Equal(t, "alice", claims["username"])
		assert.Equal(t, "HIS-1", claims["hospital_id"])
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	pw := "rightpw"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)

	mock := &mockRepo{
		getFunc: func(ctx context.Context, username, hospitalID string) (*repository.Staff, error) {
			return &repository.Staff{
				ID:           "s1",
				Username:     "alice",
				PasswordHash: string(hashed),
				HospitalID:   "HIS-1",
				Role:         "staff",
			}, nil
		},
	}
	svc := NewAuthService(mock)
	_, err := svc.Authenticate(context.Background(), "alice", "wrongpw", "HIS-1", "secret", time.Hour)
	assert.ErrorIs(t, err, ErrInvalidCreds)
}

func TestAuthenticate_NotFound(t *testing.T) {
	mock := &mockRepo{
		getFunc: func(ctx context.Context, username, hospitalID string) (*repository.Staff, error) {
			return nil, nil
		},
	}
	svc := NewAuthService(mock)
	_, err := svc.Authenticate(context.Background(), "missing", "pw", "HIS-1", "secret", time.Hour)
	assert.ErrorIs(t, err, ErrUserNotFound)
}
