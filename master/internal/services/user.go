package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// UserService manages panel user accounts.
type UserService struct {
	db *pgxpool.Pool
}

// NewUserService creates a new UserService.
func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

// List returns all users.
func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, email, name, role, totp_enabled, created_at
		FROM users ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.TOTPEnabled, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if users == nil {
		users = []models.User{}
	}
	return users, nil
}

// Create creates a new user account.
func (s *UserService) Create(ctx context.Context, email, password, name, role string) (*models.User, error) {
	if email == "" || password == "" || name == "" {
		return nil, fmt.Errorf("email, password and name are required")
	}
	if role == "" {
		role = "admin"
	}

	hashedPw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var u models.User
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, password, name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, name, role, totp_enabled, created_at
	`, email, string(hashedPw), name, role).Scan(
		&u.ID, &u.Email, &u.Name, &u.Role, &u.TOTPEnabled, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

// Update updates a user's name and role.
func (s *UserService) Update(ctx context.Context, id, name, role string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users SET name = $1, role = $2, updated_at = NOW() WHERE id = $3
	`, name, role, id)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// Delete removes a user account.
func (s *UserService) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// ChangePassword updates a user's password.
func (s *UserService) ChangePassword(ctx context.Context, id, newPassword string) error {
	if newPassword == "" {
		return fmt.Errorf("password must not be empty")
	}

	hashedPw, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2
	`, string(hashedPw), id)
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	return nil
}

// SetupTOTP generates a new TOTP secret for a user and returns the secret and OTP auth URL.
func (s *UserService) SetupTOTP(ctx context.Context, userID string) (secret, qrCodeURL string, err error) {
	var email string
	if err := s.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "ControlPanelVPS",
		AccountName: email,
		Period:      30,
		SecretSize:  20,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate totp key: %w", err)
	}

	// Store secret temporarily (not yet enabled)
	_, err = s.db.Exec(ctx, `
		UPDATE users SET totp_secret = $1, updated_at = NOW() WHERE id = $2
	`, key.Secret(), userID)
	if err != nil {
		return "", "", fmt.Errorf("store totp secret: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// VerifyAndEnableTOTP verifies the code and enables TOTP for the user.
func (s *UserService) VerifyAndEnableTOTP(ctx context.Context, userID, code string) error {
	var secret *string
	if err := s.db.QueryRow(ctx, `SELECT totp_secret FROM users WHERE id = $1`, userID).Scan(&secret); err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if secret == nil || *secret == "" {
		return fmt.Errorf("TOTP setup not initiated")
	}

	valid := totp.Validate(code, *secret)
	if !valid {
		return ErrInvalidTOTP
	}

	_, err := s.db.Exec(ctx, `
		UPDATE users SET totp_enabled = TRUE, updated_at = NOW() WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	return nil
}

// DisableTOTP disables TOTP for a user and clears the secret.
func (s *UserService) DisableTOTP(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users SET totp_enabled = FALSE, totp_secret = NULL, updated_at = $1 WHERE id = $2
	`, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}
	return nil
}
