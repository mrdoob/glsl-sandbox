package server

import (
	"fmt"
	"net/http"
	"time"

	"slices"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/mrdoob/glsl-sandbox/server/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenCookieName = "access-token"
	tokenDuration         = time.Hour * 24 * 30
	bcryptCost            = 8
)

var ErrNotAuthorized = fmt.Errorf("user not authorized")

type Claims struct {
	jwt.RegisteredClaims

	Name string     `json:"name"`
	Role store.Role `json:"role"`
}

type Auth struct {
	users  *store.Users
	secret string
}

func NewAuth(users *store.Users, secret string) *Auth {
	return &Auth{
		users:  users,
		secret: secret,
	}
}

func (a *Auth) GenerateToken(c echo.Context, u store.User) error {
	if u.Name == "" {
		return fmt.Errorf("invalid name")
	}
	if u.Role == "" {
		return fmt.Errorf("invalid role")
	}

	expirationTime := jwt.NewNumericDate(time.Now().Add(tokenDuration))
	claims := Claims{
		Name: u.Name,
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: expirationTime,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)

	tokenString, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return fmt.Errorf("could not generate token: %w", err)
	}

	cookie := http.Cookie{
		Name:     accessTokenCookieName,
		Value:    tokenString,
		Expires:  expirationTime.Time,
		Path:     "/",
		HttpOnly: true,
	}
	c.SetCookie(&cookie)

	return nil
}

func (a *Auth) Add(
	name string,
	password string,
	email string,
	role store.Role,
) error {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("could not hash password: %w", err)
	}
	u := store.User{
		Name:      name,
		Password:  hashedPassword,
		Email:     email,
		Role:      role,
		Active:    true,
		CreatedAt: time.Now(),
	}
	return a.users.Add(u)
}

func (a *Auth) Login(c echo.Context, name, password string) error {
	u, err := a.users.User(name)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword(u.Password, []byte(password))
	if err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	err = a.GenerateToken(c, u)
	if err != nil {
		return fmt.Errorf("could not generate cookie: %w", err)
	}

	return nil
}

func (a *Auth) CheckPermissions(c echo.Context, roles ...store.Role) error {
	user := c.Get("user")
	if user == nil {
		return fmt.Errorf("token not set")
	}

	u, ok := user.(*jwt.Token)
	if !ok {
		return fmt.Errorf("malformed token")
	}

	if !u.Valid {
		return fmt.Errorf("invalid claims")
	}

	claims, ok := u.Claims.(*Claims)
	if !ok {
		return fmt.Errorf("invalid claims")
	}

	if slices.Contains(roles, claims.Role) {
		return nil
	}

	return fmt.Errorf("not enough permissions: %s", claims.Role)
}

func (a *Auth) Middleware(
	f func(echo.Context, error) error,
) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:   []byte(a.secret),
		TokenLookup:  "cookie:" + accessTokenCookieName,
		ErrorHandler: f,
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(Claims)
		},
	})
}
