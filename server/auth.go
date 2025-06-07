package server

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
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

	ID         int    `json:"id"`
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
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
		ID:         u.ID,
		Provider:   u.Provider,
		ProviderID: u.ProviderID,
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

func (a *Auth) AddPassword(
	name string,
	password string,
	email string,
	role store.Role,
) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password), bcryptCost)
	if err != nil {
		return -1, fmt.Errorf("could not hash password: %w", err)
	}
	u := store.User{
		Name:      name,
		Password:  hashedPassword,
		Email:     email,
		Role:      role,
		Active:    true,
		CreatedAt: time.Now().UTC(),
		Provider:  "password",
	}

	return a.users.Add(u)
}

func (a *Auth) Add(
	name string,
	email string,
	role store.Role,
	provider string,
	providerID string,
) (int, error) {
	u := store.User{
		Name:       name,
		Email:      email,
		Role:       role,
		Active:     true,
		CreatedAt:  time.Now().UTC(),
		Provider:   provider,
		ProviderID: providerID,
	}

	return a.users.Add(u)
}

func (a *Auth) LoginPassword(c echo.Context, name, password string) error {
	u, err := a.users.Name(name)
	if err != nil {
		return err
	}

	if !u.Active {
		return fmt.Errorf("user not active: %s", name)
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

func (a *Auth) LoginGoth(c echo.Context, gUser goth.User) error {
	u, err := a.users.ProviderID(gUser.Provider, gUser.UserID)
	if err != nil && errors.Is(err, store.ErrNotFound) {
		id, err := a.Add(
			gUser.NickName,
			gUser.Email,
			store.RoleUser,
			gUser.Provider,
			gUser.UserID,
		)
		if err != nil {
			return err
		}

		u, err = a.users.User(id)
		if err != nil {
			return err
		}
	}

	if !u.Active {
		return fmt.Errorf("user not active: %s-%s", u.Provider, u.ProviderID)
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

	dbUser, err := a.users.User(claims.ID)
	if err != nil {
		return err
	}

	if !dbUser.Active {
		return errors.New("user not active")
	}

	if slices.Contains(roles, dbUser.Role) {
		return nil
	}

	return fmt.Errorf("not enough permissions: %s", dbUser.Role)
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
