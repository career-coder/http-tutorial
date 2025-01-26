package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	repo := "bazelbuild/rules_swift"
	scheme := "http"

	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("/repos/%s/pulls", repo), func(w http.ResponseWriter, r *http.Request) {
		body, err := os.ReadFile("./testdata/pulls.json")
		require.NoError(t, err)

		u := url.URL{
			Scheme: scheme,
			Host:   r.Host,
			Path:   "/pulldiff",
		}

		bodyString := strings.Replace(string(body), "DIFF_URL_REPLACE", u.String(), -1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(bodyString))
	})

	mux.HandleFunc("/pulldiff", func(w http.ResponseWriter, r *http.Request) {
		u := url.URL{
			Scheme: scheme,
			Host:   r.Host,
			Path:   "/redirect",
		}
		http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
	})

	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test diff"))
	})

	srv := httptest.NewServer(mux)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	stdout := bytes.NewBuffer(nil)
	require.NoError(t, run(params{
		client: srv.Client(),
		scheme: scheme,
		host:   u.Host,
		repo:   repo,
		stdout: stdout,
	}))

	assert.Contains(t, stdout.String(), "test diff")
}
