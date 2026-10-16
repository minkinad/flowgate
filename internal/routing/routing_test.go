package routing

import (
	"net/http/httptest"
	"testing"

	"github.com/minkinad/flowgate/internal/config"
)

func TestMatchLongestPrefixAndHost(t *testing.T) {
	routes := []config.Route{{Name: "all"}, {Name: "orders"}, {Name: "host"}}
	routes[0].Match.PathPrefix = "/"
	routes[1].Match.PathPrefix = "/api/orders"
	routes[2].Match.PathPrefix = "/api/orders"
	routes[2].Match.Host = "other.example"
	for _, tc := range []struct{ path, host, want string }{{"/api/orders/1", "example.com", "orders"}, {"/other", "example.com", "all"}, {"/api/orders/1", "other.example", "host"}} {
		r := httptest.NewRequest("GET", "http://"+tc.host+tc.path, nil)
		got := Match(routes, r)
		if got == nil || got.Name != tc.want {
			t.Fatalf("Match(%s,%s)=%v, want %s", tc.path, tc.host, got, tc.want)
		}
	}
}

func BenchmarkMatch(b *testing.B) {
	routes := []config.Route{{Name: "all"}}
	routes[0].Match.PathPrefix = "/"
	r := httptest.NewRequest("GET", "http://example.com/path", nil)
	for i := 0; i < b.N; i++ {
		_ = Match(routes, r)
	}
}
