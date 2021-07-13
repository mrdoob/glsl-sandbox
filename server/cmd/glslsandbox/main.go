package main

import (
	"os"

	"github.com/mrdoob/glsl-sandbox/server"
	"github.com/mrdoob/glsl-sandbox/server/store"
)

func main() {
	err := start()
	if err != nil {
		panic(err)
	}
}

func start() error {
	db, err := store.NewSqlite(":memory:")
	if err != nil {
		return err
	}
	err = db.Init()
	if err != nil {
		return err
	}

	f, err := os.Open("code.100")
	if err != nil {
		return err
	}

	err = store.Import(f, db)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	s := server.New(db)
	return s.Start()
}
