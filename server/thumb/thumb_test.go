package thumb

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThumbs(t *testing.T) {
	d, err := os.MkdirTemp("", "glsl")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	thumbs, err := NewThumbs(d)
	require.NoError(t, err)
	require.Equal(t, d, thumbs.Path())

	err = thumbs.Save("../secret", []byte{0})
	require.Error(t, err)

	err = thumbs.Save("1.png", []byte("data"))
	require.NoError(t, err)

	data, err := os.ReadFile(path.Join(d, "1.png"))
	require.NoError(t, err)

	require.Equal(t, []byte("data"), data)
}
