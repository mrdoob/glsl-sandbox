package store

import (
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/sqliteshim"
)

var (
	testDatabase = ":memory:"
	testTime     = time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	testUser     = User{
		Name:      "test",
		Password:  []byte("password"),
		Email:     "email",
		Role:      RoleAdmin,
		Active:    true,
		CreatedAt: testTime,
	}
)

func TestUserAdd(t *testing.T) {
	db, err := sqlx.Connect(sqliteshim.ShimName, testDatabase)
	require.NoError(t, err)

	users, err := NewUsers(db)
	require.NoError(t, err)

	u, err := users.User("test")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
	require.Equal(t, User{}, u)

	err = users.Add(testUser)
	require.NoError(t, err)

	u, err = users.User("test")
	require.NoError(t, err)
	require.Equal(t, 1, u.ID)
	u.ID = 0
	require.Equal(t, testUser, u)
}

func TestUserUpdate(t *testing.T) {
	db, err := sqlx.Connect(sqliteshim.ShimName, testDatabase)
	require.NoError(t, err)

	users, err := NewUsers(db)
	require.NoError(t, err)

	err = users.Add(testUser)
	require.NoError(t, err)

	expected := User{
		Name:      "test",
		Password:  []byte("newpassword"),
		Email:     "newemail",
		Role:      RoleModerator,
		Active:    false,
		CreatedAt: time.Now(),
	}
	err = users.Update(expected)
	require.NoError(t, err)

	u, err := users.User("test")
	require.NoError(t, err)
	require.Equal(t, expected.Name, u.Name)
	require.Equal(t, expected.Password, u.Password)
	require.Equal(t, expected.Email, u.Email)
	require.Equal(t, expected.Role, u.Role)
	require.Equal(t, expected.Active, u.Active)
	require.True(t, expected.CreatedAt.Equal(u.CreatedAt))

	expected.Name = "inexistent"
	err = users.Update(expected)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestUserUpdateFunc(t *testing.T) {
	db, err := sqlx.Connect(sqliteshim.ShimName, testDatabase)
	require.NoError(t, err)

	users, err := NewUsers(db)
	require.NoError(t, err)

	err = users.Add(testUser)
	require.NoError(t, err)

	expected := User{
		Name:      "test",
		Password:  []byte("newpassword"),
		Email:     "newemail",
		Role:      RoleModerator,
		Active:    false,
		CreatedAt: time.Now(),
	}
	err = users.UpdateFunc("test", func(u User) User {
		return expected
	})
	require.NoError(t, err)

	u, err := users.User("test")
	require.NoError(t, err)
	require.Equal(t, expected.Name, u.Name)
	require.Equal(t, expected.Password, u.Password)
	require.Equal(t, expected.Email, u.Email)
	require.Equal(t, expected.Role, u.Role)
	require.Equal(t, expected.Active, u.Active)
	require.True(t, expected.CreatedAt.Equal(u.CreatedAt))

	err = users.UpdateFunc("inexistent", func(u User) User {
		return u
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestUserGetAll(t *testing.T) {
	db, err := sqlx.Connect(sqliteshim.ShimName, testDatabase)
	require.NoError(t, err)

	users, err := NewUsers(db)
	require.NoError(t, err)

	err = users.Add(User{Name: "one"})
	require.NoError(t, err)

	err = users.Add(User{Name: "two"})
	require.NoError(t, err)

	list, err := users.Users()
	require.NoError(t, err)

	var names []string
	for _, u := range list {
		names = append(names, u.Name)
	}

	require.ElementsMatch(t, []string{"one", "two"}, names)
}
