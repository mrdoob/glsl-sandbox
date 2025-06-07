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
		Name:       "test",
		Password:   []byte("password"),
		Email:      "email",
		Role:       RoleAdmin,
		Active:     true,
		CreatedAt:  testTime,
		Provider:   "github",
		ProviderID: "0",
	}
)

func TestUserAdd(t *testing.T) {
	db, err := sqlx.Connect(sqliteshim.ShimName, testDatabase)
	require.NoError(t, err)

	users, err := NewUsers(db)
	require.NoError(t, err)

	all, err := users.Users()
	require.NoError(t, err)
	require.Len(t, all, 0)

	u, err := users.User(0)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
	require.Equal(t, User{}, u)

	id, err := users.Add(testUser)
	require.NoError(t, err)
	require.Equal(t, 1, id)

	u, err = users.ProviderID("github", "0")
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

	id, err := users.Add(testUser)
	require.NoError(t, err)

	expected := User{
		ID:         id,
		Name:       "test2",
		Password:   []byte("newpassword"),
		Email:      "newemail",
		Role:       RoleModerator,
		Active:     false,
		CreatedAt:  time.Now().UTC(),
		Provider:   "new_provider",
		ProviderID: "42",
	}
	err = users.Update(expected)
	require.NoError(t, err)

	u, err := users.User(id)
	require.NoError(t, err)
	require.Equal(t, expected, u)

	expected.ID = 1024
	err = users.Update(expected)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestUserUpdateFunc(t *testing.T) {
	db, err := sqlx.Connect(sqliteshim.ShimName, testDatabase)
	require.NoError(t, err)

	users, err := NewUsers(db)
	require.NoError(t, err)

	id, err := users.Add(testUser)
	require.NoError(t, err)

	expected := User{
		ID:         id,
		Name:       "test",
		Password:   []byte("newpassword"),
		Email:      "newemail",
		Role:       RoleModerator,
		Active:     false,
		CreatedAt:  time.Now().UTC(),
		Provider:   "gitlab",
		ProviderID: "some user",
	}
	err = users.UpdateFunc(id, func(u User) User {
		return expected
	})
	require.NoError(t, err)

	u, err := users.User(id)
	require.NoError(t, err)
	require.Equal(t, expected, u)

	err = users.UpdateFunc(1024, func(u User) User {
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

	_, err = users.Add(User{Name: "one", Provider: "test"})
	require.NoError(t, err)

	_, err = users.Add(User{Name: "two", Provider: "test"})
	require.NoError(t, err)

	list, err := users.Users()
	require.NoError(t, err)

	var names []string
	for _, u := range list {
		names = append(names, u.Name)
	}

	require.ElementsMatch(t, []string{"one", "two"}, names)
}

func TestValidate(t *testing.T) {
	t.Skip("create this test")
}
