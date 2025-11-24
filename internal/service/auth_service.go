package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/haniscreator/agnos-search/internal/repository"
)

// StaffRepoMinimal is the subset of repository.StaffRepo used by AuthService.
type StaffRepoMinimal interface {
	Create(ctx context.Context, s *repository.Staff) error
	GetByUsernameAndHospital(ctx context.Context, username, hospitalID string) (*repository.Staff, error)
}

// AuthService handles staff registration and authentication.
type AuthService interface {
	// Register creates a new staff (hashes password) and stores it.
	Register(ctx context.Context, username, password, hospitalID, displayName string) (*repository.Staff, error)

	// Authenticate verifies credentials and returns a signed JWT token string.
	Authenticate(ctx context.Context, username, password, hospitalID, jwtSecret string, expiresIn time.Duration) (string, error)
}

type authServiceImpl struct {
	repo StaffRepoMinimal
}

// NewAuthService constructs a new AuthService.
func NewAuthService(repo StaffRepoMinimal) AuthService {
	return &authServiceImpl{repo: repo}
}

var (
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidCreds    = errors.New("invalid credentials")
	ErrUserNotFound    = errors.New("user not found")
	ErrWeakPassword    = errors.New("password too weak")
	ErrTokenGeneration = errors.New("token generation failed")
)

// Register creates a staff record. Caller must pass hospital id.
// Password is hashed with bcrypt before saving.
func (s *authServiceImpl) Register(ctx context.Context, username, password, hospitalID, displayName string) (*repository.Staff, error) {
	// simple password policy: min 6 chars (you can strengthen later)
	if len(password) < 6 {
		return nil, ErrWeakPassword
	}

	// check existing user (reuse repo method)
	existing, err := s.repo.GetByUsernameAndHospital(ctx, username, hospitalID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	// hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	st := &repository.Staff{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hashed),
		HospitalID:   hospitalID,
		DisplayName:  displayName,
		Role:         "staff",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, st); err != nil {
		return nil, err
	}
	// Do not return password hash to callers (it's inside st but caller should ignore)
	st.PasswordHash = ""
	return st, nil
}

// Authenticate verifies username/password/hospital, returns signed JWT.
func (s *authServiceImpl) Authenticate(ctx context.Context, username, password, hospitalID, jwtSecret string, expiresIn time.Duration) (string, error) {
	st, err := s.repo.GetByUsernameAndHospital(ctx, username, hospitalID)
	if err != nil {
		return "", err
	}
	if st == nil {
		return "", ErrUserNotFound
	}

	// verify password
	if err := bcrypt.CompareHashAndPassword([]byte(st.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCreds
	}

	// create token
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":         st.ID,
		"username":    st.Username,
		"hospital_id": st.HospitalID,
		"role":        st.Role,
		"iat":         now.Unix(),
		"exp":         now.Add(expiresIn).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", ErrTokenGeneration
	}
	return signed, nil
}
