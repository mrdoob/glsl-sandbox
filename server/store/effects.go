package store

import (
	"fmt"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
)

type Effect struct {
	ID            int
	CreatedAt     time.Time
	ModifiedAt    time.Time
	Parent        int
	ParentVersion int
	User          string
	Hidden        bool
	Versions      []Version
}

func (e Effect) ImageName() string {
	return fmt.Sprintf("%d.png", e.ID)
}

type Version struct {
	CreatedAt time.Time
	Code      string
}

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

type Effects struct {
	db *sqlx.DB
}

func NewEffects(db *sqlx.DB) (*Effects, error) {
	e := &Effects{
		db: db,
	}
	err := e.init()
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Effects) init() error {
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

	sqlInsertEffect = `
INSERT INTO effects (
	created_at,
	modified_at,
	parent,
	parent_version,
	user,
	hidden
) VALUES(
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

	sqlSelectEffectsSiblings = `
SELECT * FROM effects
	WHERE id = ? OR
		parent = ?
	ORDER BY modified_at DESC
	LIMIT ? OFFSET ?
`

	sqlSelectVersions = `
SELECT * FROM versions
	WHERE effect = ?
	ORDER BY version
`

	sqlSelectVersionsMulti = `
SELECT * FROM versions
	WHERE effect IN (?)
	ORDER BY version
`

	sqlSelectEffect = `
SELECT * FROM effects
	WHERE id = ?
`

	sqlSelectMaxVersion = `
SELECT MAX(version) FROM versions
	WHERE effect = ?
`

	sqlUpdateEffectModification = `
UPDATE effects
	SET modified_at = ?
	WHERE id = ?
`

	sqlUpdateEffectHide = `
UPDATE effects
	SET hidden = ?
	WHERE id = ?
`
)

func (s *Effects) AddEffect(e Effect) error {
	return s.transaction(func(tx *sqlx.Tx) error {
		effect := sqliteFromEffect(e)
		_, err := tx.NamedExec(sqlInsertEffectID, effect)
		if err != nil {
			return fmt.Errorf("could not insert effect: %w", err)
		}

		for i, v := range e.Versions {
			version := sqliteFromversion(v)
			version.Version = i
			version.Effect = effect.ID

			_, err := tx.NamedExec(sqlInsertVersion, version)
			if err != nil {
				return fmt.Errorf("could not insert version: %w", err)
			}
		}

		return nil
	})
}

func (s *Effects) Add(
	parent int, parentVersion int, user string, version string,
) (int, error) {
	var lastID int
	err := s.transaction(func(tx *sqlx.Tx) error {
		t := time.Now()
		e := sqliteEffect{
			CreatedAt:     t,
			ModifiedAt:    t,
			Parent:        parent,
			ParentVersion: parentVersion,
			User:          user,
		}

		r, err := tx.NamedExec(sqlInsertEffect, e)
		if err != nil {
			return fmt.Errorf("could not insert effect: %w", err)
		}

		id, err := r.LastInsertId()
		if err != nil {
			return fmt.Errorf("could not get effect id: %w", err)
		}
		lastID = int(id)

		v := sqliteVersion{
			Version:   0,
			Effect:    int(id),
			CreatedAt: t,
			Code:      version,
		}
		_, err = tx.NamedExec(sqlInsertVersion, v)
		if err != nil {
			return fmt.Errorf("could not insert version: %w", err)
		}
		return nil
	})

	return lastID, err
}

func (s *Effects) AddVersion(id int, code string) (int, error) {
	var lastVersion int
	err := s.transaction(func(tx *sqlx.Tx) error {
		t := time.Now()
		var maxVersion *int
		r := tx.QueryRowx(sqlSelectMaxVersion, id)
		err := r.Scan(&maxVersion)
		if err != nil {
			return fmt.Errorf("could not get max version: %w", err)
		}

		if maxVersion == nil {
			return ErrNotFound
		}

		version := sqliteVersion{
			Version:   *maxVersion + 1,
			Effect:    id,
			CreatedAt: t,
			Code:      code,
		}
		_, err = tx.NamedExec(sqlInsertVersion, version)
		if err != nil {
			return fmt.Errorf("could not insert version: %w", err)
		}
		lastVersion = version.Version

		_, err = tx.Exec(sqlUpdateEffectModification, t, id)
		if err != nil {
			return fmt.Errorf("could not update effect: %w", err)
		}

		return nil
	})

	return lastVersion, err
}

func (s *Effects) Page(num int, size int, hidden bool) ([]Effect, error) {
	query := sqlSelectEffects
	if hidden {
		query = sqlSelectEffectsAll
	}

	return s.page(query, []interface{}{size, num * size})
}

func (s *Effects) PageSiblings(num int, size int, parent int) ([]Effect, error) {
	query := sqlSelectEffectsSiblings

	return s.page(query, []interface{}{parent, parent, size, num * size})
}

func (s *Effects) page(query string, qargs []interface{}) ([]Effect, error) {
	iter, err := s.db.Queryx(query, qargs...)
	if err != nil {
		return nil, fmt.Errorf("could not get effects: %w", err)
	}
	defer iter.Close()

	var effects []Effect
	var ids []int
	for iter.Next() {
		var e sqliteEffect
		err = iter.StructScan(&e)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve effect: %w", err)
		}

		effects = append(effects, sqliteToEffect(e))
		ids = append(ids, e.ID)
	}

	if len(effects) == 0 {
		return effects, nil
	}

	query, args, err := sqlx.In(sqlSelectVersionsMulti, ids)
	if err != nil {
		return nil, fmt.Errorf("could not construct versions query: %w", err)

	}
	iter, err = s.db.Queryx(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not get versions: %w", err)
	}
	defer iter.Close()

	versions := make(map[int][]sqliteVersion)
	for iter.Next() {
		var v sqliteVersion
		err = iter.StructScan(&v)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve version: %w", err)
		}

		versions[v.Effect] = append(versions[v.Effect], v)
	}
	if iter.Err() != nil {
		return nil, fmt.Errorf("could not iterate versions: %w", err)
	}

	for _, e := range versions {
		sort.Slice(e, func(i, j int) bool {
			return e[i].Version < e[j].Version
		})
	}

	for i, e := range effects {
		s := make([]Version, 0, len(versions[e.ID]))
		for _, v := range versions[e.ID] {
			s = append(s, sqliteToVersion(v))
		}
		effects[i].Versions = s
	}

	return effects, nil
}

func (s *Effects) versions(id int) ([]Version, error) {
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

func (s *Effects) Effect(id int) (Effect, error) {
	var e sqliteEffect
	r := s.db.QueryRowx(sqlSelectEffect, id)
	err := r.StructScan(&e)
	if err != nil {
		return Effect{}, fmt.Errorf("could not get effect: %w", err)
	}

	versions, err := s.versions(id)
	if err != nil {
		return Effect{}, err
	}

	effect := sqliteToEffect(e)
	effect.Versions = versions

	return effect, nil
}

func (s *Effects) Hide(id int, hidden bool) error {
	r, err := s.db.Exec(sqlUpdateEffectHide, hidden, id)
	if err != nil {
		return fmt.Errorf("could not update effect: %w", err)
	}

	n, err := r.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not update effect: %w", err)

	}
	if n < 1 {
		return ErrNotFound
	}

	return nil
}

func (s *Effects) transaction(f func(*sqlx.Tx) error) error {
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
