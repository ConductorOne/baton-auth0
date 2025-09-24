package client

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessFile(t *testing.T) {
	cases := []struct {
		file  string
		count int
	}{
		{
			file:  "asset/test.json.gz",
			count: 2,
		},
		{
			file:  "asset/test2.json.gz",
			count: 1,
		},
		{
			file:  "asset/empty.json.gz",
			count: 0,
		},
	}

	for _, s := range cases {
		t.Run(s.file, func(t *testing.T) {
			file, err := os.OpenFile(s.file, os.O_RDONLY, 0600)
			require.NoError(t, err)
			defer file.Close()

			users, err := process(file)
			require.NoError(t, err)
			require.Len(t, users, s.count)
		})
	}
}
