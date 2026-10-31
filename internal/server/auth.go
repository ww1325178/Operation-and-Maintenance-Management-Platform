package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Auth struct {
	pool   *pgxpool.Pool
	secret []byte
}

func NewAuth(pool *pgxpool.Pool, secret string) *Auth {
	return &Auth{pool: pool, secret: []byte(secret)}
}

// SeedAdmin ensures an admin user exists, using the configured password.
// Existing rows are never modified.
func (a *Auth) SeedAdmin(ctx context.Context, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = a.pool.Exec(ctx, `
		INSERT INTO users(username, password_hash, role)
		VALUES($1, $2, 'admin')
		ON CONFLICT (username) DO NOTHING`, username, string(hash))
	return err
}

func (a *Auth) Login(ctx context.Context, username, password string) (string, error) {
	var (
		id   int64
		hash string
		role string
	)
	err := a.pool.QueryRow(ctx,
		`SELECT id, password_hash, role FROM users WHERE username=$1`, username).
		Scan(&id, &hash, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	claims := Claims{
		UserID:   id,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

func (a *Auth) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return a.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// Middleware protects routes; it reads the Bearer token and injects claims.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "missing token")
			return
		}
		claims, err := a.Parse(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
