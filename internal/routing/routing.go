package routing

import (
	"net"
	"net/http"
	"strings"

	"github.com/minkinad/flowgate/internal/config"
)

// Match chooses the longest prefix, then an explicit host, then config order.
func Match(routes []config.Route, r *http.Request) *config.Route {
	var best *config.Route
	host := r.Host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	for i := range routes {
		candidate := &routes[i]
		if candidate.Match.Host != "" && !strings.EqualFold(candidate.Match.Host, host) {
			continue
		}
		if !strings.HasPrefix(r.URL.Path, candidate.Match.PathPrefix) {
			continue
		}
		if best == nil || len(candidate.Match.PathPrefix) > len(best.Match.PathPrefix) || (len(candidate.Match.PathPrefix) == len(best.Match.PathPrefix) && candidate.Match.Host != "" && best.Match.Host == "") {
			best = candidate
		}
	}
	return best
}
