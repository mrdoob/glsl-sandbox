package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	"github.com/mrdoob/glsl-sandbox/server"
	"github.com/mrdoob/glsl-sandbox/server/store"
	"github.com/uptrace/bun/driver/sqliteshim"
)

const dbName = "glslsandbox.db"

type Config struct {
	DataPath     string `envconfig:"DATA_PATH" default:"./data"`
	Import       string `envconfig:"IMPORT"`
	AuthSecret   string `envconfig:"AUTH_SECRET" default:"secret"`
	Addr         string `envconfig:"ADDR" default:":8888"`
	TLSAddr      string `envconfig:"TLS_ADDR"`
	Domains      string `envconfig:"DOMAINS" default:"www.glslsandbox.com,glslsandbox.com"`
	Dev          bool   `envconfig:"DEV" default:"false"`
	ReadOnly     bool   `envconfig:"READ_ONLY" default:"false"`
	CallbackHost string `envconfig:"CALLBACK_HOST"`
}

func main() {
	err := start()
	if err != nil {
		panic(err)
	}
}

func start() error {
	var cfg Config
	if err := envconfig.Process("GLSL_", &cfg); err != nil {
		return fmt.Errorf("could not read environment config: %w", err)
	}

	err := os.MkdirAll(filepath.Join(cfg.DataPath, "thumbs"), 0770)
	if err != nil {
		return fmt.Errorf("could not create data directory: %w", err)
	}

	db, err := sqlx.Open(sqliteshim.ShimName, dbURL(cfg.DataPath))
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}

	effects, err := store.NewEffects(db)
	if err != nil {
		return fmt.Errorf("could not initialize effects database: %w", err)
	}

	users, err := store.NewUsers(db)
	if err != nil {
		return fmt.Errorf("could not initialize users database: %w", err)
	}

	auth := server.NewAuth(users, cfg.AuthSecret)

	// err = createUser(auth, users)
	// if err != nil {
	// 	return err
	// }

	if cfg.Import != "" {
		err = importDatabase(effects, cfg.Import)
		if err != nil {
			return fmt.Errorf("could not import database: %w", err)
		}
	}

	callbackHost := cfg.CallbackHost
	if callbackHost == "" && cfg.Dev {
		callbackHost = "127.0.0.1:8888"
	}
	if callbackHost == "" {
		return errors.New("CALLBACK_HOST must be set")
	}

	s, err := server.New(
		cfg.Addr,
		cfg.TLSAddr,
		cfg.Domains,
		effects,
		auth,
		cfg.DataPath,
		cfg.Dev,
		cfg.ReadOnly,
		callbackHost,
	)
	if err != nil {
		return fmt.Errorf("could not create server: %w", err)
	}

	return s.Start()
}

func dbURL(path string) string {
	file := filepath.Join(path, dbName)
	// return fmt.Sprintf("file:%s?_journal_mode=WAL", file)
	return fmt.Sprintf("file:%s", file)
}

func importDatabase(effects *store.Effects, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("could not open import file: %w", err)
	}
	defer f.Close()

	err = store.Import(f, effects)
	if err != nil {
		return fmt.Errorf("could not import effects: %w", err)
	}

	return nil
}

// TODO(jfontan): is admin user needed at start?
// func createUser(auth *server.Auth, users *store.Users) error {
// 	_, err := users.User("admin")
// 	if err == nil {
// 		return nil
// 	}

// 	b := make([]byte, 16)
// 	_, err = rand.Read(b)
// 	if err != nil {
// 		return fmt.Errorf("could not generate password: %w", err)
// 	}
// 	m := md5.Sum(b)
// 	password := hex.EncodeToString(m[:])
// 	err = auth.Add("admin", password, "", store.RoleAdmin)
// 	if err != nil {
// 		return fmt.Errorf("could not create admin user: %w", err)
// 	}

// 	fmt.Printf("created user 'admin' with password '%s'", password)
// 	return nil
// }
