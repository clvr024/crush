package agent

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
)

type cassetteStreamModel struct {
	fantasy.LanguageModel
	response fantasy.StreamResponse
	err      error
}

func (m cassetteStreamModel) Stream(context.Context, fantasy.Call) (fantasy.StreamResponse, error) {
	return m.response, m.err
}

func TestCassetteModelMissingInteraction(t *testing.T) {
	t.Parallel()

	for _, deferred := range []bool{false, true} {
		name := "stream creation"
		if deferred {
			name = "stream iteration"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			original := &url.Error{Op: "Post", URL: "https://example.com", Err: cassette.ErrInteractionNotFound}
			inner := cassetteStreamModel{err: original}
			if deferred {
				inner.err = nil
				inner.response = func(yield func(fantasy.StreamPart) bool) {
					yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeError, Error: original})
				}
			}
			stream, err := (cassetteModel{inner}).Stream(t.Context(), fantasy.Call{})
			if deferred {
				require.NoError(t, err)
				for part := range stream {
					err = part.Error
				}
			}
			var providerErr *fantasy.ProviderError
			require.ErrorAs(t, err, &providerErr)
			require.ErrorIs(t, err, cassette.ErrInteractionNotFound)
			require.False(t, providerErr.IsRetryable())
		})
	}
}

func TestCassetteModelPreservesOtherResults(t *testing.T) {
	t.Parallel()

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		original := errors.New("other failure")
		_, err := (cassetteModel{cassetteStreamModel{err: original}}).Stream(t.Context(), fantasy.Call{})
		require.Same(t, original, err)
	})

	t.Run("successful stream and early stop", func(t *testing.T) {
		t.Parallel()
		want := fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, ID: "1", Delta: "hello"}
		inner := cassetteStreamModel{response: func(yield func(fantasy.StreamPart) bool) {
			if !yield(want) {
				return
			}
			t.Fatal("consumer stopped the stream")
		}}
		stream, err := (cassetteModel{inner}).Stream(t.Context(), fantasy.Call{})
		require.NoError(t, err)
		var parts []fantasy.StreamPart
		for part := range stream {
			parts = append(parts, part)
			break
		}
		require.Equal(t, []fantasy.StreamPart{want}, parts)
	})
}
