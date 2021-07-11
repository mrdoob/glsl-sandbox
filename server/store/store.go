package store

import (
	"fmt"
	"time"
)

var (
	ErrNotFound = fmt.Errorf("effect not found")
)

type Effect struct {
	ID            int
	CreatedAt     time.Time
	ModifiedAt    time.Time
	ImageURL      string
	Parent        int
	ParentVersion int
	User          string
	Hidden        bool
	Versions      []Version
}

type Version struct {
	CreatedAt time.Time
	Code      string
}

type Store interface {
	AddEffect(e Effect) error
	Add(parent int, parentVersion int, imageURL string, user string,
		version string,
	) (int, error)
	AddVersion(id int, code string) (int, error)
	Page(num int, size int, hidden bool) ([]Effect, error)
	Effect(id int) (Effect, error)
}
