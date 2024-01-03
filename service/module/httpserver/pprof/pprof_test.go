package pprof_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/elisasre/go-common/service/module/httpserver"
	"github.com/elisasre/go-common/service/module/httpserver/pprof"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	srv := httpserver.New(
		httpserver.WithServer(&http.Server{ReadHeaderTimeout: time.Second}),
		httpserver.WithAddr("127.0.0.1:0"),
		pprof.WithProfiling(),
	)

	require.NotEmpty(t, srv.Name())
	require.NoError(t, srv.Init())
	wg := &multierror.Group{}
	wg.Go(srv.Run)

	assertOK(t, srv.URL()+"/debug/pprof/heap")

	assert.NoError(t, srv.Stop())
	err := wg.Wait().ErrorOrNil()
	require.NoError(t, err)
}

func assertOK(t testing.TB, url string) {
	resp, err := http.Get(url) //nolint:gosec
	if !assert.NoError(t, err) {
		return
	}

	data, err := io.ReadAll(resp.Body)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "200 OK", resp.Status)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, resp.Body.Close())
	assert.NotEmpty(t, data)
}
