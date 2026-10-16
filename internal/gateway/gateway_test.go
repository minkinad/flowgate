package gateway

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minkinad/flowgate/internal/config"
)

func TestProxyAndHealth(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()
	c := config.Config{}
	c.Routes = []config.Route{{Name: "test"}}
	c.Routes[0].Match.PathPrefix = "/"
	c.Routes[0].Upstreams = []struct {
		URL string `yaml:"url"`
	}{{URL: upstream.URL}}
	c.Routes[0].TimeoutText = "1s"
	c.Server.Listen = ":8080"
	c.Server.LogFormat = "text"
	c.Server.LogLevel = "info"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(c, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/hello")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 201 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	for _, path := range []string{"/health/live", "/health/ready", "/metrics"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("%s status=%d", path, resp.StatusCode)
		}
	}
}
