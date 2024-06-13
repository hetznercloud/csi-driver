package mocked

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Request struct {
	Method     string
	Path       string
	PathRegexp string

	Status int
	Body   string
	JSON   any
}

func Handler(t *testing.T, requests []Request) http.HandlerFunc {
	t.Helper()

	index := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if index >= len(requests) {
			t.Fatalf("received unknown request %d", index)
		}

		response := requests[index]
		assert.Equal(t, response.Method, r.Method)
		if response.PathRegexp != "" {
			require.Regexp(t, response.Path, r.RequestURI, "request %d", index)
		} else {
			require.Equal(t, response.Path, r.RequestURI, "request %d", index)
		}

		w.WriteHeader(response.Status)
		w.Header().Set("Content-Type", "application/json")
		if response.Body != "" {
			_, err := w.Write([]byte(response.Body))
			if err != nil {
				t.Fatal(err)
			}
		} else if response.JSON != nil {
			if err := json.NewEncoder(w).Encode(response.JSON); err != nil {
				t.Fatal(err)
			}
		}

		index++
	})
}
