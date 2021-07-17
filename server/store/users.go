package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Role string

const (
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleUser      = "user"
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
