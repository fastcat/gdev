package gocache

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseActionEntry(t *testing.T) {
	tests := []struct {
		input string
		want  ActionEntry
		err   error
	}{
		{
			input: `v1 d405b479e410c7a2bc276897d858fb8ee480eb4e2ca089845ddd3d5e1cad195c 8a3bc0f228cc3894fa61d1c181043e1b2cec92c4ef8ac1f4445d082c3cd37132               797818  1750435547055716553` + "\n",
			want: ActionEntry{
				ID: []byte{
					0xd4, 0x05, 0xb4, 0x79, 0xe4, 0x10, 0xc7, 0xa2, 0xbc, 0x27, 0x68, 0x97, 0xd8, 0x58, 0xfb, 0x8e,
					0xe4, 0x80, 0xeb, 0x4e, 0x2c, 0xa0, 0x89, 0x84, 0x5d, 0xdd, 0x3d, 0x5e, 0x1c, 0xad, 0x19, 0x5c,
				},
				OutputID: []byte{
					0x8a, 0x3b, 0xc0, 0xf2, 0x28, 0xcc, 0x38, 0x94, 0xfa, 0x61, 0xd1, 0xc1, 0x81, 0x04, 0x3e, 0x1b,
					0x2c, 0xec, 0x92, 0xc4, 0xef, 0x8a, 0xc1, 0xf4, 0x44, 0x5d, 0x08, 0x2c, 0x3c, 0xd3, 0x71, 0x32,
				},
				Size: 797818,
				Time: time.Unix(0, 1750435547055716553),
			},
		},
	}
	for idx, tt := range tests {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			got, err := parseActionEntry([]byte(tt.input))
			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
				return
			}
			if !assert.NoError(t, err) ||
				!assert.NotNil(t, got) {
				return
			}
			require.Equal(t, tt.want, *got)

			// test loop back to string, technically tests a different function
			var loop strings.Builder
			_, err = got.WriteTo(&loop)
			require.NoError(t, err)
			assert.Equal(t, tt.input, loop.String())
		})
	}
}
