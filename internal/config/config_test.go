package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidation(t *testing.T) {
	cases := []struct {
		name, body string
		valid      bool
	}{
		{"valid", "server:\n  listen: ':8080'\nroutes:\n  - name: app\n    match:\n      path_prefix: /\n    upstreams:\n      - url: http://127.0.0.1:9000\n    timeout: 2s\n", true},
		{"unknown", "bogus: true\nroutes: []\n", false},
		{"bad timeout", "routes:\n  - name: app\n    match:\n      path_prefix: /\n    upstreams:\n      - url: http://127.0.0.1:9000\n    timeout: -1s\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "flowgate.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if (err == nil) != tc.valid {
				t.Fatalf("err=%v valid=%v", err, tc.valid)
			}
		})
	}
}
