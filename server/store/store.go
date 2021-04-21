package store

import (
	"fmt"
	"sort"
	"sync"
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

type Memory struct {
	Effects map[int]Effect
	last    int
	m       sync.RWMutex
}

func NewMemory() (*Memory, error) {
	return &Memory{
		Effects: make(map[int]Effect),
		last:    0,
	}, nil
}

func (m *Memory) AddEffect(e Effect) error {
	m.m.Lock()
	defer m.m.Unlock()

	if e.ID == 0 {
		e.ID = m.last
	}

	if m.last <= e.ID {
		m.last = e.ID + 1
	}

	m.Effects[e.ID] = e
	return nil
}

func (m *Memory) Add(
	parent int,
	parentVersion int,
	imageURL string,
	user string,
	version string,
) (int, error) {
	m.m.Lock()
	defer m.m.Unlock()

	e := Effect{
		ID:            m.last,
		CreatedAt:     time.Now(),
		ImageURL:      imageURL,
		ModifiedAt:    time.Now(),
		Parent:        parent,
		ParentVersion: parentVersion,
		User:          user,
		Versions: []Version{
			{
				CreatedAt: time.Now(),
				Code:      version,
			},
		},
	}

	m.Effects[e.ID] = e
	m.last++

	return e.ID, nil
}

func (m *Memory) AddVersion(id int, code string) (int, error) {
	m.m.Lock()
	defer m.m.Unlock()

	e, ok := m.Effects[id]
	if !ok {
		return 0, ErrNotFound
	}

	e.Versions = append(e.Versions, Version{
		CreatedAt: time.Now(),
		Code:      code,
	})
	e.ModifiedAt = time.Now()
	m.Effects[id] = e

	return len(e.Versions) - 1, nil
}

func (m *Memory) Get(id int) (Effect, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	e, ok := m.Effects[id]
	if !ok {
		return Effect{}, ErrNotFound
	}

	return e, nil
}

type idModified struct {
	id       int
	modified time.Time
}

func (m *Memory) Page(num int, size int, hidden bool) ([]Effect, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	ids := make([]idModified, 0, len(m.Effects))
	for i, e := range m.Effects {
		if !hidden && e.Hidden {
			continue
		}
		ids = append(ids, idModified{id: i, modified: e.ModifiedAt})
	}

	sort.Slice(ids, func(a, b int) bool {
		return ids[a].modified.Before(ids[b].modified)
	})

	start := num * size
	if start >= len(ids) {
		return []Effect{}, nil
	}
	end := start + size
	if end > len(ids) {
		end = len(ids)
	}

	effects := make([]Effect, 0, end-start)
	for _, i := range ids {
		effects = append(effects, m.Effects[i.id])
	}

	return effects, nil
}
