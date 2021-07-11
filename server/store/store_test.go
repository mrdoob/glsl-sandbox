package store

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type helper struct {
	name string
	test func(t *testing.T, s Store)
}

var tests = []helper{
	{"import", testImport},
	{"hidden", testHidden},
	{"add version", testAddVersion},
	{"hide", testHide},
}

func TestMemory(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewMemory()
			require.NoError(t, err)
			test.test(t, s)
		})
	}
}

func TestSqlite(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewSqlite(":memory:")
			require.NoError(t, err)

			err = s.Init()
			require.NoError(t, err)

			test.test(t, s)
		})
	}
}

func testImport(t *testing.T, s Store) {
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

func testHidden(t *testing.T, s Store) {
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

func testAddVersion(t *testing.T, s Store) {
	id, err := s.Add(10, 5, "user", "first")
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
	require.Equal(t, "user", e.User)
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

func testHide(t *testing.T, s Store) {
	id, err := s.Add(10, 5, "user", "first")
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
