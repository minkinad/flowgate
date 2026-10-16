package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Listen    string `yaml:"listen"`
		LogFormat string `yaml:"log_format"`
		LogLevel  string `yaml:"log_level"`
	} `yaml:"server"`
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Name  string `yaml:"name"`
	Match struct {
		Host       string `yaml:"host"`
		PathPrefix string `yaml:"path_prefix"`
	} `yaml:"match"`
	Upstreams []struct {
		URL string `yaml:"url"`
	} `yaml:"upstreams"`
	Timeout     time.Duration `yaml:"-"`
	TimeoutText string        `yaml:"timeout"`
}

func Load(path string) (Config, error) {
	c := Config{}
	c.Server.Listen = ":8080"
	c.Server.LogFormat = "json"
	c.Server.LogLevel = "info"
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return c, errors.New("multiple YAML documents are not supported")
	}
	if v := os.Getenv("FLOWGATE_LISTEN"); v != "" {
		c.Server.Listen = v
	}
	if v := os.Getenv("FLOWGATE_LOG_FORMAT"); v != "" {
		c.Server.LogFormat = v
	}
	if v := os.Getenv("FLOWGATE_LOG_LEVEL"); v != "" {
		c.Server.LogLevel = v
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func (c *Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.Server.Listen); err != nil {
		return fmt.Errorf("server.listen: %w", err)
	}
	if c.Server.LogFormat != "json" && c.Server.LogFormat != "text" {
		return errors.New("server.log_format must be json or text")
	}
	if !validLevel(c.Server.LogLevel) {
		return errors.New("server.log_level must be debug, info, warn or error")
	}
	if len(c.Routes) == 0 {
		return errors.New("at least one route is required")
	}
	names := map[string]bool{}
	for i := range c.Routes {
		r := &c.Routes[i]
		if r.Name == "" || names[r.Name] {
			return fmt.Errorf("route %d: name must be unique and nonempty", i)
		}
		names[r.Name] = true
		if !strings.HasPrefix(r.Match.PathPrefix, "/") {
			return fmt.Errorf("route %s: path_prefix must start with /", r.Name)
		}
		if len(r.Upstreams) != 1 {
			return fmt.Errorf("route %s: exactly one upstream is supported in this version", r.Name)
		}
		u, err := url.Parse(r.Upstreams[0].URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("route %s: upstream must be an HTTP(S) origin without credentials, query or fragment", r.Name)
		}
		if r.TimeoutText == "" {
			r.TimeoutText = "5s"
		}
		r.Timeout, err = time.ParseDuration(r.TimeoutText)
		if err != nil || r.Timeout <= 0 || r.Timeout > time.Minute {
			return fmt.Errorf("route %s: timeout must be within (0, 1m]", r.Name)
		}
	}
	return nil
}

func validLevel(s string) bool { return s == "debug" || s == "info" || s == "warn" || s == "error" }
