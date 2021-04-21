package store

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	effects, err := NewMemory()
	require.NoError(t, err)

	buf := bytes.NewBufferString(importData)
	err = Import(buf, effects)
	require.NoError(t, err)

	es, err := effects.Page(1, 10, false)
	require.NoError(t, err)
	require.Len(t, es, 0)

	es, err = effects.Page(0, 10, false)
	require.NoError(t, err)
	require.Len(t, es, 4)

	ids := []int{10140, 10141, 10142, 10143}

	for i, e := range es {
		require.Equal(t, ids[i], e.ID)
		require.False(t, e.CreatedAt.IsZero())
		require.False(t, e.ModifiedAt.IsZero())
		img := fmt.Sprintf("%d.png", e.ID)
		require.Equal(t, img, e.ImageURL)
		require.False(t, e.User == "")
		require.False(t, e.Hidden)
		require.True(t, len(e.Versions) > 0)
		require.False(t, e.Parent == 0)
		require.Equal(t, 0, e.ParentVersion)
	}
}

func TestHidden(t *testing.T) {
	effects, err := NewMemory()
	require.NoError(t, err)

	for _, e := range testEffects {
		err = effects.AddEffect(e)
		require.NoError(t, err)
	}

	es, err := effects.Page(0, 10, false)
	require.NoError(t, err)
	require.Len(t, es, 1)
	require.Equal(t, 1, es[0].ID)

	es, err = effects.Page(0, 10, true)
	require.NoError(t, err)
	require.Len(t, es, 2)
	require.Equal(t, 2, es[0].ID)
	require.Equal(t, 1, es[1].ID)
}
