package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"fastcat.org/go/gdev/examples/full-stack/ent-blog/ent/enttest"
)

func TestIndex(t *testing.T) {
	// Initialize an Ent client that uses an in memory SQLite db.
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	// seed the database with our "Hello, world" post and user.
	err := seed(context.Background(), client)
	require.NoError(t, err)

	// Initialize a server and router.
	srv := newServer(client)
	r := newRouter(srv)

	// Create a test server using the `httptest` package.
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Make a GET request to the server root path.
	resp, err := ts.Client().Get(ts.URL)

	// Assert we get a 200 OK status code.
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Read the response body and assert it contains "Hello, world!"
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "Hello, World!")
}

func TestAdd(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	err := seed(context.Background(), client)
	require.NoError(t, err)

	srv := newServer(client)
	r := newRouter(srv)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Post the form.
	resp, err := ts.Client().PostForm(ts.URL+"/add", map[string][]string{
		"title": {"Testing, one, two."},
		"body":  {"This is a test"},
	})
	require.NoError(t, err)
	// We should be redirected to the index page and receive 200 OK.
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// The home page should contain our new post.
	require.Contains(t, string(body), "This is a test")
}
