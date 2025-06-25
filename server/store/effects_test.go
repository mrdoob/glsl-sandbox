package store

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type helper struct {
	name string
	test func(t *testing.T, s *Effects)
}

var tests = []helper{
	{"import", testImport},
	{"hidden", testHidden},
	{"add version", testAddVersion},
	{"hide", testHide},
	{"siblings", testSiblings},
}

func TestEffects(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := sqlx.Open(sqliteshim.ShimName, testDatabase)
			require.NoError(t, err)

			s, err := NewEffects(db)
			require.NoError(t, err)

			test.test(t, s)
		})
	}
}

func testImport(t *testing.T, s *Effects) {
	buf := bytes.NewBufferString(importData)
	err := Import(buf, s)
	require.NoError(t, err)

	es, err := s.Page(1, 10, false)
	require.NoError(t, err)
	require.Len(t, es, 0)

	es, err = s.Page(0, 10, false)
	require.NoError(t, err)
	require.Len(t, es, 4)

	ids := []int{10143, 10142, 10141, 10140}

	for i, e := range es {
		require.Equal(t, ids[i], e.ID)
		require.False(t, e.CreatedAt.IsZero())
		require.False(t, e.ModifiedAt.IsZero())
		img := fmt.Sprintf("%d.png", e.ID)
		require.Equal(t, img, e.ImageName())
		require.False(t, e.User == "")
		require.False(t, e.Hidden)
		require.True(t, len(e.Versions) > 0)
		require.False(t, e.Parent == 0)
		require.Equal(t, 0, e.ParentVersion)
	}
}

func testHidden(t *testing.T, s *Effects) {
	for _, e := range testEffects {
		err := s.AddEffect(e)
		require.NoError(t, err)
	}

	es, err := s.Page(0, 10, false)
	require.NoError(t, err)
	require.Len(t, es, 1)
	require.Equal(t, 1, es[0].ID)

	es, err = s.Page(0, 10, true)
	require.NoError(t, err)
	require.Len(t, es, 2)
	require.Equal(t, 1, es[0].ID)
	require.Equal(t, 2, es[1].ID)
}

func testAddVersion(t *testing.T, s *Effects) {
	id, err := s.Add(10, 5, 42, "first")
	require.NoError(t, err)
	require.Equal(t, 1, id)

	e, err := s.Effect(1)
	require.NoError(t, err)
	require.Equal(t, 1, e.ID)
	require.Equal(t, false, e.Hidden)
	require.True(t, e.CreatedAt.Before(time.Now()))
	require.False(t, e.CreatedAt.IsZero())
	require.True(t, e.ModifiedAt.Before(time.Now()))
	require.False(t, e.ModifiedAt.IsZero())
	require.Equal(t, "1.png", e.ImageName())
	require.Equal(t, 10, e.Parent)
	require.Equal(t, 5, e.ParentVersion)
	require.Equal(t, 42, e.UserID)
	require.Len(t, e.Versions, 1)

	v := e.Versions[0]
	require.Equal(t, "first", v.Code)
	require.True(t, v.CreatedAt.Before(time.Now()))
	require.False(t, v.CreatedAt.IsZero())

	vid, err := s.AddVersion(1, "second")
	require.NoError(t, err)
	require.Equal(t, 1, vid)

	ti := e.ModifiedAt
	e, err = s.Effect(1)
	require.NoError(t, err)
	require.True(t, ti.Before(e.ModifiedAt))
	require.Len(t, e.Versions, 2)
	require.Equal(t, "second", e.Versions[1].Code)

	_, err = s.AddVersion(2, "invalid")
	require.Error(t, err)
	require.Equal(t, err, ErrNotFound)
}

func testHide(t *testing.T, s *Effects) {
	id, err := s.Add(10, 5, 0, "first")
	require.NoError(t, err)
	require.Equal(t, 1, id)

	e, err := s.Effect(1)
	require.NoError(t, err)
	require.False(t, e.Hidden)

	err = s.Hide(1, true)
	require.NoError(t, err)
	e, err = s.Effect(1)
	require.NoError(t, err)
	require.True(t, e.Hidden)

	err = s.Hide(1, false)
	require.NoError(t, err)
	e, err = s.Effect(1)
	require.NoError(t, err)
	require.False(t, e.Hidden)

	err = s.Hide(2, true)
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func testSiblings(t *testing.T, s *Effects) {
	pid, err := s.Add(-1, -1, 0, "parent")
	require.NoError(t, err)

	expected := []int{pid}
	for i := 0; i < 10; i++ {
		id, err := s.Add(pid, 0, 0, "child")
		require.NoError(t, err)
		expected = append(expected, id)
	}

	for i := 0; i < 10; i++ {
		_, err = s.Add(-1, -1, 0, "no")
		require.NoError(t, err)
	}

	effects, err := s.PageSiblings(0, 50, pid)
	require.NoError(t, err)

	var ids []int
	for _, e := range effects {
		require.Len(t, e.Versions, 1)
		if e.ID == pid {
			require.Equal(t, "parent", e.Versions[0].Code)
		} else {
			require.Equal(t, "child", e.Versions[0].Code)
		}
		ids = append(ids, e.ID)
	}

	require.ElementsMatch(t, expected, ids)
}
