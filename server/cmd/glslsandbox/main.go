package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
	"github.com/mrdoob/glsl-sandbox/server"
	"github.com/mrdoob/glsl-sandbox/server/store"
)

const dbName = "glslsandbox.db"

type Config struct {
	DataPath string `envconfig:"DATA_PATH" default:"./data"`
	Import   string `envconfig:"IMPORT"`
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

	err := os.MkdirAll(filepath.Join(cfg.DataPath, "thumbs"), 0666)
	if err != nil {
		return fmt.Errorf("could not create data directory: %w", err)
	}

	db, err := store.NewSqlite(dbURL(cfg.DataPath))
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}
	err = db.Init()
	if err != nil {
		return fmt.Errorf("could not initialize database: %w", err)
	}

	if cfg.Import != "" {
		err = importDatabase(db, cfg.Import)
		if err != nil {
			return fmt.Errorf("could not import database: %w", err)
		}
	}

	s := server.New(db, cfg.DataPath)
	return s.Start()
}

func dbURL(path string) string {
	file := filepath.Join(path, dbName)
	// return fmt.Sprintf("file:%s?_journal_mode=WAL", file)
	return fmt.Sprintf("file:%s", file)
}

func importDatabase(db store.Store, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("could not open import file: %w", err)
	}
	defer f.Close()

	err = store.Import(f, db)
	if err != nil {
		return fmt.Errorf("could not import effects: %w", err)
	}

	return nil
}
