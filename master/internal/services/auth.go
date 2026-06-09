package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/db"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTOTPRequired       = errors.New("totp_required")
	ErrInvalidTOTP        = errors.New("invalid totp code")
	ErrUserNotFound       = errors.New("user not found")
)

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret []byte
}

func NewAuthService(db *pgxpool.Pool, jwtSecret string) *AuthService {
	return &AuthService{db: db, jwtSecret: []byte(jwtSecret)}
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, email, password, totpCode string) (string, *models.User, error) {
	var user models.User
	var hashedPw string
	var totpSecret *string

	err := s.db.QueryRow(ctx, `
		SELECT id, email, name, role, password, totp_secret, totp_enabled
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &hashedPw, &totpSecret, &user.TOTPEnabled)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPw), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if user.TOTPEnabled {
		if totpCode == "" {
			return "", nil, ErrTOTPRequired
		}
		if totpSecret == nil || !totp.Validate(totpCode, *totpSecret) {
			return "", nil, ErrInvalidTOTP
		}
	}

	token, err := s.generateToken(&user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	_, _ = s.db.Exec(ctx, `UPDATE users SET updated_at = NOW() WHERE id = $1`, user.ID)

	return token, &user, nil
}

// WriteLoginAudit records a login attempt to the audit log.
// Called by the handler so the remote IP is available.
func (s *AuthService) WriteLoginAudit(ctx context.Context, userID, email, remoteAddr string, success bool) {
	action := "login_success"
	if !success {
		action = "login_failed"
	}
	db.WriteAuditLog(ctx, s.db, userID, action, "auth", remoteAddr, map[string]any{
		"email": email,
	})
}

func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, name, role, totp_enabled, created_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.TOTPEnabled, &user.CreatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}
