package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbedFilesNil(t *testing.T) {
	for name, style := range map[string]FileEmbedStyle{"text": EmbedText, "reader": EmbedIOReader} {
		t.Run(name, func(t *testing.T) {
			var result any
			require.NotPanics(t, func() {
				var err error
				result, err = embedFiles(nil, style)
				require.NoError(t, err)
			})
			require.Nil(t, result)
		})
	}
}
