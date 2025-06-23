package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_serviceModesValue_Set(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    serviceModesValue
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty",
			in:   "",
			want: serviceModesValue{},
		},
		{
			name: "no entries",
			in:   "[]",
			want: serviceModesValue{},
		},
		{
			name: "single",
			in:   `["foo"=debug]`,
			want: serviceModesValue{"foo": ModeDebug},
		},
		{
			name: "multiple",
			in:   `["foo"=debug,"bar"=local]`,
			want: serviceModesValue{"foo": ModeDebug, "bar": ModeLocal},
		},
		{
			name: "bad empty",
			in:   " ",
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorContains(t, err, "must start with [", msgAndArgs...)
			},
		},
		{
			name: "trailing comma",
			in:   `["foo"=debug,]`,
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorContains(t, err, "must have another entry", msgAndArgs...)
			},
		},
		{
			name: "bad mode",
			in:   `["foo"=bar]`,
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorContains(t, err, "invalid mode value", msgAndArgs...)
			},
		},
		{
			name: "bad quoting",
			in:   `["foo\"=bar]`,
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...any) bool {
				return assert.ErrorContains(t, err, "keys must be quoted", msgAndArgs...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := serviceModesValue{}
			errOK := false
			if tt.wantErr == nil {
				errOK = assert.NoError(t, v.Set(tt.in))
			} else {
				errOK = tt.wantErr(t, v.Set(tt.in))
			}
			if errOK && tt.want != nil {
				assert.Equal(t, tt.want, v)
			}
		})
	}
}
