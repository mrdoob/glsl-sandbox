package thumb

import (
	"errors"
	"fmt"
	"os"
	"path"
)

type Thumbs struct {
	path string
}

func NewThumbs(p string) (*Thumbs, error) {
	f, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(p, 0700)
			if err != nil {
				return nil, fmt.Errorf("could not create directory: %w", err)
			}
			f, err = os.Stat(p)
			if err != nil {
				return nil, fmt.Errorf("could not stat thumbs: %w", err)
			}
		} else {
			return nil, fmt.Errorf("thumbs stat error: %w", err)
		}
	}

	if !f.IsDir() {
		return nil, fmt.Errorf("thumbs path is not a directory")
	}

	return &Thumbs{path: p}, nil
}

func (t Thumbs) Path() string {
	return t.path
}

func (t Thumbs) Save(n string, d []byte) error {
	dir, n := path.Split(n)
	if dir != "" {
		return fmt.Errorf("malformed path")
	}

	p := path.Join(t.path, n)
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("could not create thumb %s: %w", p, err)
	}
	defer f.Close()

	_, err = f.Write(d)
	if err != nil {
		return fmt.Errorf("could not write thumb: %w", err)
	}

	return nil
}
