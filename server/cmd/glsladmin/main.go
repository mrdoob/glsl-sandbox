package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	"github.com/mrdoob/glsl-sandbox/server/store"
	"github.com/uptrace/bun/driver/sqliteshim"
	"golang.org/x/crypto/bcrypt"
)

const dbName = "glslsandbox.db"

var (
	ErrNotEnoughParameters = fmt.Errorf("not enough parameters")
)

type Config struct {
	DataPath string `envconfig:"DATA_PATH" default:"./data"`
}

func main() {
	err := start()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

type cmd func(*store.Users) error

var commands = map[string]cmd{
	"list":   list,
	"add":    createUser,
	"passwd": changePassword,
}

func usage() {
	fmt.Println(`Usage:
	glsladmin list -- list users
	glsladmin add <name> [<email>] -- add new user
	glsladmin passwd <name> -- change user password`)
	fmt.Println()
}

func start() error {
	var cfg Config
	if err := envconfig.Process("GLSL_", &cfg); err != nil {
		return fmt.Errorf("could not read environment config: %w", err)
	}

	db, err := sqlx.Open(sqliteshim.ShimName, dbURL(cfg.DataPath))
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}

	users, err := store.NewUsers(db)
	if err != nil {
		return fmt.Errorf("could not initialize users database: %w", err)
	}

	if len(os.Args) < 2 {
		usage()
		return ErrNotEnoughParameters
	}

	c, ok := commands[os.Args[1]]
	if !ok {
		usage()
		return fmt.Errorf("bad command")
	}

	err = c(users)
	if err != nil {
		usage()
	}
	return err
}

func dbURL(path string) string {
	file := filepath.Join(path, dbName)
	return fmt.Sprintf("file:%s", file)
}

func list(users *store.Users) error {
	list, err := users.Users()
	if err != nil {
		return err
	}

	for _, u := range list {
		fmt.Printf("%s %s %s %v %v\n",
			u.Name,
			u.Email,
			u.Role,
			u.Active,
			u.CreatedAt,
		)
	}

	return nil
}

func createUser(users *store.Users) error {
	user := ""
	email := ""
	switch len(os.Args) {
	case 3:
		user = os.Args[2]
	case 4:
		user = os.Args[2]
		email = os.Args[3]
	default:
		return ErrNotEnoughParameters
	}

	_, err := users.User(user)
	if err == nil {
		return fmt.Errorf("user already exist")
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	password, hashedPassword, err := genPassword()
	if err != nil {
		return err
	}

	u := store.User{
		Name:      user,
		Email:     email,
		Password:  hashedPassword,
		Role:      store.RoleModerator,
		Active:    true,
		CreatedAt: time.Now(),
	}
	err = users.Add(u)
	if err != nil {
		return fmt.Errorf("could not create user: %w", err)
	}

	fmt.Printf("created user '%s' with password '%s'\n", user, password)
	return nil
}

func changePassword(users *store.Users) error {
	if len(os.Args) < 3 {
		return ErrNotEnoughParameters
	}

	user := os.Args[2]
	password, hashedPassword, err := genPassword()
	if err != nil {
		return err
	}

	err = users.UpdateFunc(user, func(u store.User) store.User {
		u.Password = hashedPassword
		return u
	})
	if err != nil {
		return err
	}

	fmt.Printf("updated user '%s' with new password '%s'\n", user, password)
	return nil
}

func genPassword() (string, []byte, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", nil, fmt.Errorf("could not generate password: %w", err)
	}
	m := md5.Sum(b)
	password := hex.EncodeToString(m[:])

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", nil, fmt.Errorf("could not generate password: %w", err)
	}

	return password, hashedPassword, err
}
