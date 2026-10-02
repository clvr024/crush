package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionsGetLoopMaxRepeats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		options  *Options
		expected int
	}{
		{
			name:     "nil options",
			options:  nil,
			expected: DefaultLoopMaxRepeats,
		},
		{
			name:     "unset field",
			options:  &Options{},
			expected: DefaultLoopMaxRepeats,
		},
		{
			name:     "default is 4",
			options:  &Options{},
			expected: 4,
		},
		{
			name:     "non-positive treated as unset",
			options:  &Options{LoopMaxRepeats: ptr(0)},
			expected: DefaultLoopMaxRepeats,
		},
		{
			name:     "custom value",
			options:  &Options{LoopMaxRepeats: ptr(6)},
			expected: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.options.GetLoopMaxRepeats())
		})
	}
}

func TestOptionsGetLoopHistorySize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		options  *Options
		expected int
	}{
		{
			name:     "nil options",
			options:  nil,
			expected: DefaultLoopHistorySize,
		},
		{
			name:     "unset field",
			options:  &Options{},
			expected: DefaultLoopHistorySize,
		},
		{
			name:     "custom value",
			options:  &Options{LoopHistorySize: ptr(20)},
			expected: 20,
		},
		{
			name:     "clamped up to threshold",
			options:  &Options{LoopMaxRepeats: ptr(6), LoopHistorySize: ptr(4)},
			expected: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.options.GetLoopHistorySize())
		})
	}
}

func TestOptionsLoopDetectorFromJSON(t *testing.T) {
	t.Parallel()

	var cfg Config
	require.NoError(t, json.Unmarshal([]byte(`{"options":{"loop_max_repeats":5,"loop_history_size":20}}`), &cfg))
	require.Equal(t, 5, *cfg.Options.LoopMaxRepeats)
	require.Equal(t, 20, *cfg.Options.LoopHistorySize)
	require.Equal(t, 5, cfg.Options.GetLoopMaxRepeats())
	require.Equal(t, 20, cfg.Options.GetLoopHistorySize())
}
