package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseChapters(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("testdata", "hello-world.txt"))
	require.NoError(t, err)
	chapters, err := ParseChapters(bytes.NewReader(data))
	require.NoError(t, err)
	require.Len(t, chapters, 12)
	require.Equal(t, "1", chapters[1].Title)
	require.InDelta(t, 2.23, chapters[1].Start.Seconds(), 0.1)
	require.Equal(t, "11", chapters[11].Title)
	require.InDelta(t, 13.13, chapters[11].Start.Seconds(), 0.1)
}
