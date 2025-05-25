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
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Password  []byte    `db:"password"`
	Email     string    `db:"email"`
	Role      Role      `db:"role"`
	Active    bool      `db:"active"`
	CreatedAt time.Time `db:"created_at"`
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
	created_at TIMESTAMP
)
`

	sqlIndexUsersName = `
CREATE INDEX IF NOT EXISTS idx_users_name ON users (name)
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
	return nil
}

const (
	sqlSelectUsers = `
SELECT * FROM users
`

	sqlSelectUser = `
SELECT * FROM users
	WHERE name = ?
`

	sqlInsertUser = `
INSERT INTO users (
	name,
	password,
	email,
	role,
	active,
	created_at
) VALUES(
	:name,
	:password,
	:email,
	:role,
	:active,
	:created_at
)
`

	sqlUpdateUser = `
UPDATE users
	SET
		password = :password,
		email = :email,
		role = :role,
		active = :active,
		created_at = :created_at
	WHERE name = :name
`
)

func (s *Users) User(name string) (User, error) {
	var u User
	r := s.db.QueryRowx(sqlSelectUser, name)
	err := r.StructScan(&u)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("could not get user: %w", err)
	}

	return u, nil
}

func (s *Users) Add(user User) error {
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	_, err := s.db.NamedExec(sqlInsertUser, user)
	if err != nil {
		return fmt.Errorf("could not add user: %w", err)
	}
	return nil
}

func (s *Users) Update(user User) error {
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

func (s *Users) UpdateFunc(name string, f func(User) User) error {
	return s.transaction(func(tx *sqlx.Tx) error {
		var u User
		r := tx.QueryRowx(sqlSelectUser, name)
		err := r.StructScan(&u)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("could not get user: %w", err)
		}

		u = f(u)

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
