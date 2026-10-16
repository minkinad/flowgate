package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/minkinad/flowgate/internal/config"
	"github.com/minkinad/flowgate/internal/routing"
)

type Gateway struct {
	Config    config.Config
	Logger    *slog.Logger
	requests  atomic.Uint64
	errors    atomic.Uint64
	inflight  atomic.Int64
	requestID atomic.Uint64
	proxies   map[string]*httputil.ReverseProxy
}

// TODO(flowgate): Add health-aware selection before constructing a proxy request.
// Suggested approach: keep an immutable snapshot of upstream states per route,
// update it from bounded active probes, then select only healthy endpoints.
// Acceptance: deterministic selection tests, failure recovery tests, and a
// benchmark showing allocations per proxied request.

func New(c config.Config, logger *slog.Logger) *Gateway {
	g := &Gateway{Config: c, Logger: logger, proxies: make(map[string]*httputil.ReverseProxy, len(c.Routes))}
	for _, route := range c.Routes {
		target, _ := url.Parse(route.Upstreams[0].URL)
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, MaxIdleConns: 100, MaxIdleConnsPerHost: 100, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: route.Timeout, DialContext: (&net.Dialer{Timeout: route.Timeout, KeepAlive: 30 * time.Second}).DialContext}
		name := route.Name
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			g.errors.Add(1)
			status := http.StatusBadGateway
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				status = http.StatusGatewayTimeout
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				status = http.StatusGatewayTimeout
			}
			g.Logger.Warn("upstream request failed", "route", name, "request_id", r.Header.Get("X-Request-ID"), "error", err)
			http.Error(w, http.StatusText(status), status)
		}
		g.proxies[name] = proxy
	}
	return g
}

func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /health/ready", g.ready)
	mux.HandleFunc("GET /metrics", g.metrics)
	mux.HandleFunc("/", g.proxy)
	return mux
}

func (g *Gateway) ready(w http.ResponseWriter, r *http.Request) {
	for _, route := range g.Config.Routes {
		u, _ := url.Parse(route.Upstreams[0].URL)
		host := u.Host
		if u.Port() == "" {
			if u.Scheme == "https" {
				host = net.JoinHostPort(u.Hostname(), "443")
			} else {
				host = net.JoinHostPort(u.Hostname(), "80")
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host)
		cancel()
		if err != nil {
			http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
			return
		}
		_ = conn.Close()
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready\n"))
}

func (g *Gateway) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "# TYPE flowgate_requests_total counter\nflowgate_requests_total %d\n# TYPE flowgate_proxy_errors_total counter\nflowgate_proxy_errors_total %d\n# TYPE flowgate_inflight_requests gauge\nflowgate_inflight_requests %d\n", g.requests.Load(), g.errors.Load(), g.inflight.Load())
}

func (g *Gateway) proxy(w http.ResponseWriter, r *http.Request) {
	g.requests.Add(1)
	g.inflight.Add(1)
	defer g.inflight.Add(-1)
	start := time.Now()
	id := r.Header.Get("X-Request-ID")
	if id == "" || len(id) > 128 || strings.ContainsAny(id, "\r\n") {
		id = strconv.FormatUint(g.requestID.Add(1), 10)
	}
	w.Header().Set("X-Request-ID", id)
	route := routing.Match(g.Config.Routes, r)
	if route == nil {
		http.NotFound(w, r)
		return
	}
	r.Header.Set("X-Request-ID", id)
	// Never trust client supplied forwarding headers.
	r.Header.Del("Forwarded")
	r.Header.Del("X-Forwarded-For")
	r.Header.Del("X-Forwarded-Host")
	r.Header.Del("X-Forwarded-Proto")
	ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
	defer cancel()
	g.proxies[route.Name].ServeHTTP(w, r.WithContext(ctx))
	g.Logger.Info("request", "route", route.Name, "request_id", id, "duration", time.Since(start))
}
