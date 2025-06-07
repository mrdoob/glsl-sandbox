package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleUser      Role = "user"
)

type User struct {
	ID         int       `db:"id"`
	Name       string    `db:"name"`
	Password   []byte    `db:"password"`
	Email      string    `db:"email"`
	Role       Role      `db:"role"`
	Active     bool      `db:"active"`
	CreatedAt  time.Time `db:"created_at"`
	Provider   string    `db:"provider"`
	ProviderID string    `db:"provider_id"`
}

func (u User) Validate() error {
	if u.Provider == "" {
		return errors.New("provider is empty")
	}

	if u.Provider != "test" && u.ProviderID == "" {
		return errors.New("provider_id is empty")
	}

	if u.Provider == "password" && len(u.Password) == 0 {
		return errors.New("password is empty")
	}

	return nil
}

const (
	sqlCreateUsers = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	password BLOB,
	email TEXT,
	role TEXT,
	active INTEGER,
	created_at TIMESTAMP,
	provider TEXT,
	provider_id TEXT
)
`

	sqlIndexUsersName = `
CREATE INDEX IF NOT EXISTS idx_users_name ON users (name)
`

	sqlIndexUsersProviderID = `
CREATE INDEX IF NOT EXISTS idx_users_provider_id ON users (provider, provider_id)
`
)

type Users struct {
	db *sqlx.DB
}

func NewUsers(db *sqlx.DB) (*Users, error) {
	u := &Users{
		db: db,
	}
	err := u.Init()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Users) Init() error {
	_, err := s.db.Exec(sqlCreateUsers)
	if err != nil {
		return fmt.Errorf("could not create table users: %w", err)
	}

	_, err = s.db.Exec(sqlIndexUsersName)
	if err != nil {
		return fmt.Errorf("could not create users(name): %w", err)
	}

	_, err = s.db.Exec(sqlIndexUsersProviderID)
	if err != nil {
		return fmt.Errorf("could not create users(provider, provider_id): %w", err)
	}

	return nil
}

const (
	sqlSelectUsers = `
SELECT * FROM users
`

	sqlSelectUser = `
SELECT * FROM users
	WHERE id = ?
`

	sqlSelectUserName = `
SELECT * FROM users
	WHERE name = ?
`

	sqlSelectUserProviderID = `
SELECT * FROM users
	WHERE provider = ? AND provider_id = ?
`

	sqlInsertUser = `
INSERT INTO users (
	name,
	password,
	email,
	role,
	active,
	created_at,
	provider,
	provider_id
) VALUES(
	:name,
	:password,
	:email,
	:role,
	:active,
	:created_at,
	:provider,
	:provider_id
) RETURNING id
`

	sqlUpdateUser = `
UPDATE users
	SET
	  name = :name,
		password = :password,
		email = :email,
		role = :role,
		active = :active,
		created_at = :created_at,
		provider = :provider,
		provider_id = :provider_id
	WHERE id = :id
`
)

func (s *Users) User(id int) (User, error) {
	var u User
	r := s.db.QueryRowx(sqlSelectUser, id)
	err := r.StructScan(&u)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("could not get user: %w", err)
	}

	return u, nil
}

func (s *Users) Name(name string) (User, error) {
	var u User
	r := s.db.QueryRowx(sqlSelectUserName, name)
	err := r.StructScan(&u)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("could not get user: %w", err)
	}

	return u, nil
}

func (s *Users) ProviderID(provider, providerID string) (User, error) {
	var u User
	r := s.db.QueryRowx(sqlSelectUserProviderID, provider, providerID)
	err := r.StructScan(&u)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("could not get user: %w", err)
	}

	return u, nil
}

func (s *Users) Add(user User) (int, error) {
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	if err := user.Validate(); err != nil {
		return -1, err
	}

	res, err := s.db.NamedExec(sqlInsertUser, user)
	if err != nil {
		return -1, fmt.Errorf("could not add user: %w", err)
	}

	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Users) Update(user User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	r, err := s.db.NamedExec(sqlUpdateUser, user)
	if err != nil {
		return fmt.Errorf("could not update user: %w", err)
	}
	rows, err := r.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Users) UpdateFunc(id int, f func(User) User) error {
	return s.transaction(func(tx *sqlx.Tx) error {
		var u User
		r := tx.QueryRowx(sqlSelectUser, id)
		err := r.StructScan(&u)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("could not get user: %w", err)
		}

		u = f(u)
		if err := u.Validate(); err != nil {
			return err
		}

		res, err := tx.NamedExec(sqlUpdateUser, u)
		if err != nil {
			return fmt.Errorf("could not update user: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("could not get affected rows: %w", err)
		}
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *Users) Users() ([]User, error) {
	rows, err := s.db.Queryx(sqlSelectUsers)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("could not get users: %w", err)
	}

	var users []User
	for rows.Next() {
		var u User
		err = rows.StructScan(&u)
		if err != nil {
			return nil, fmt.Errorf("could not read user: %w", err)
		}
		users = append(users, u)
	}
	if rows.Err() != nil {
		if errors.Is(rows.Err(), sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("could not get users: %w", rows.Err())
	}

	return users, nil
}

func (s *Users) transaction(f func(*sqlx.Tx) error) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("could not create transaction: %w", err)
	}

	err = f(tx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}
