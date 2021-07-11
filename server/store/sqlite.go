package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteEffect struct {
	ID            int       `db:"id"`
	CreatedAt     time.Time `db:"created_at"`
	ModifiedAt    time.Time `db:"modified_at"`
	Parent        int       `db:"parent"`
	ParentVersion int       `db:"parent_version"`
	User          string    `db:"user"`
	Hidden        bool      `db:"hidden"`
}

type sqliteVersion struct {
	Version   int       `db:"version"`
	Effect    int       `db:"effect"`
	CreatedAt time.Time `db:"created_at"`
	Code      string    `db:"code"`
}

const (
	sqlCreateEffects = `
CREATE TABLE IF NOT EXISTS effects (
	id INTEGER PRIMARY KEY,
	created_at TIMESTAMP,
	modified_at TIMESTAMP,
	parent INTEGER,
	parent_version INTEGER,
	user TEXT,
	hidden INTEGER
)
`

	sqlIndexEffectsModified = `
CREATE INDEX IF NOT EXISTS idx_effects_modified ON effects (modified_at)
`

	sqlCreateVersions = `
CREATE TABLE IF NOT EXISTS versions (
	version STRING,
	effect INTEGER,
	created_at TIMESTAMP,
	code TEXT
)
`

	sqlIndexVersionEffect = `
CREATE INDEX IF NOT EXISTS idx_versions_parent ON versions (effect)
`

	sqlIndexVersionID = `
CREATE INDEX IF NOT EXISTS idx_versions_id ON versions (effect, version)
`
)

type Sqlite struct {
	db *sqlx.DB
}

func NewSqlite(path string) (*Sqlite, error) {
	db, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	return &Sqlite{
		db: db,
	}, nil
}

func (s *Sqlite) Init() error {
	_, err := s.db.Exec(sqlCreateEffects)
	if err != nil {
		return fmt.Errorf("could not create table effects: %w", err)
	}

	_, err = s.db.Exec(sqlCreateVersions)
	if err != nil {
		return fmt.Errorf("could not create table versions: %w", err)
	}

	_, err = s.db.Exec(sqlIndexEffectsModified)
	if err != nil {
		return fmt.Errorf("could not create index modified_at: %w", err)
	}

	_, err = s.db.Exec(sqlIndexVersionEffect)
	if err != nil {
		return fmt.Errorf("could not create index version effect: %w", err)
	}

	_, err = s.db.Exec(sqlIndexVersionID)
	if err != nil {
		return fmt.Errorf("could not create index version id: %w", err)
	}

	return nil
}

const (
	sqlInsertEffectID = `
INSERT INTO effects (
	id,
	created_at,
	modified_at,
	parent,
	parent_version,
	user,
	hidden
) VALUES(
	:id,
	:created_at,
	:modified_at,
	:parent,
	:parent_version,
	:user,
	:hidden
)
`

	sqlInsertVersion = `
INSERT INTO versions (
	version,
	effect,
	created_at,
	code
) VALUES(
	:version,
	:effect,
	:created_at,
	:code
)
`

	sqlSelectEffects = `
SELECT * FROM effects
	WHERE hidden = 0
	ORDER BY modified_at DESC
	LIMIT ? OFFSET ?
`

	sqlSelectEffectsAll = `
SELECT * FROM effects
	ORDER BY modified_at DESC
	LIMIT ? OFFSET ?
`

	sqlSelectVersions = `
SELECT * FROM versions
	WHERE effect = ?
	ORDER BY version
`
)

func (s *Sqlite) AddEffect(e Effect) error {
	effect := sqliteFromEffect(e)
	_, err := s.db.NamedExec(sqlInsertEffectID, effect)
	if err != nil {
		return fmt.Errorf("could not insert effect: %w", err)
	}

	for i, v := range e.Versions {
		version := sqliteFromversion(v)
		version.Version = i
		version.Effect = effect.ID

		println(version.Effect, version.Version)

		_, err := s.db.NamedExec(sqlInsertVersion, version)
		if err != nil {
			return fmt.Errorf("could not insert version: %w", err)
		}
	}

	return nil
}

func (s *Sqlite) Add(parent int, parentVersion int, user string, version string) (int, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Sqlite) AddVersion(id int, code string) (int, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Sqlite) Page(num int, size int, hidden bool) ([]Effect, error) {
	query := sqlSelectEffects
	if hidden {
		query = sqlSelectEffectsAll
	}

	iter, err := s.db.Queryx(query, size, num*size)
	if err != nil {
		return nil, fmt.Errorf("could not get effects: %w", err)
	}
	defer iter.Close()

	var effects []Effect
	for iter.Next() {
		var e sqliteEffect
		err = iter.StructScan(&e)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve effect: %w", err)
		}

		println("effect", e.ID, e.ModifiedAt.String())

		effects = append(effects, sqliteToEffect(e))
	}

	for i, e := range effects {
		versions, err := s.versions(e.ID)
		if err != nil {
			return nil, err
		}
		effects[i].Versions = versions
	}

	return effects, nil
}

func (s *Sqlite) versions(id int) ([]Version, error) {
	iter, err := s.db.Queryx(sqlSelectVersions, id)
	if err != nil {
		return nil, fmt.Errorf("could not get versions: %w", err)
	}
	defer iter.Close()

	var versions []Version
	for iter.Next() {
		var v sqliteVersion
		err = iter.StructScan(&v)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve version: %w", err)
		}

		versions = append(versions, sqliteToVersion(v))
	}

	return versions, nil
}

func (s *Sqlite) Effect(id int) (Effect, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Sqlite) Hide(id int, hidden bool) error {
	panic("not implemented") // TODO: Implement
}

func sqliteToEffect(e sqliteEffect) Effect {
	n := Effect{
		ID:            e.ID,
		CreatedAt:     e.CreatedAt,
		ModifiedAt:    e.ModifiedAt,
		Parent:        e.Parent,
		ParentVersion: e.ParentVersion,
		User:          e.User,
		Hidden:        e.Hidden,
	}
	return n
}

func sqliteToVersion(v sqliteVersion) Version {
	n := Version{
		CreatedAt: v.CreatedAt,
		Code:      v.Code,
	}
	return n
}

func sqliteFromEffect(e Effect) sqliteEffect {
	n := sqliteEffect{
		ID:            e.ID,
		CreatedAt:     e.CreatedAt,
		ModifiedAt:    e.ModifiedAt,
		Parent:        e.Parent,
		ParentVersion: e.ParentVersion,
		User:          e.User,
		Hidden:        e.Hidden,
	}
	return n
}

func sqliteFromversion(e Version) sqliteVersion {
	n := sqliteVersion{
		CreatedAt: e.CreatedAt,
		Code:      e.Code,
	}
	return n
}
