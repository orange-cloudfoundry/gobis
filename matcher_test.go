package gobis_test

import (
	"encoding/json"
	"testing"

	"github.com/orange-cloudfoundry/gobis"
	"gopkg.in/yaml.v2"
)

func TestHostMatcher(t *testing.T) {
	tests := []struct {
		pattern string
		host    string
		match   bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "www.example.com", false},
		{"*.example.com", "www.example.com", true},
		{"*.example.com", "api.www.example.com", false},
		{"**.example.com", "api.www.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.host, func(t *testing.T) {
			matcher := gobis.NewHostMatcher(tt.pattern)
			if got := matcher.Match(tt.host); got != tt.match {
				t.Errorf("Match(%q) = %t, want %t", tt.host, got, tt.match)
			}
			if got := matcher.Glob.Match(tt.host); got != tt.match {
				t.Errorf("Glob.Match(%q) = %t, want %t", tt.host, got, tt.match)
			}
			if got := matcher.String(); got != tt.pattern {
				t.Errorf("String() = %q, want %q", got, tt.pattern)
			}
		})
	}

	matchers := gobis.HostMatchers{
		gobis.NewHostMatcher("example.com"),
		gobis.NewHostMatcher("*.example.org"),
	}
	if !matchers.Match("www.example.org") || matchers.Match("www.example.com") {
		t.Error("HostMatchers.Match did not match the expected hosts")
	}
}

func TestHostMatcherUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		unmarshal func([]byte, interface{}) error
	}{
		{"JSON", json.Unmarshal},
		{"YAML", yaml.Unmarshal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matcher gobis.HostMatcher
			if err := tt.unmarshal([]byte(`"*.example.com"`), &matcher); err != nil {
				t.Fatal(err)
			}
			if matcher.String() != "*.example.com" || !matcher.Match("www.example.com") || matcher.Match("api.www.example.com") {
				t.Error("unmarshaled matcher did not preserve the pattern and separator behavior")
			}
			if err := tt.unmarshal([]byte(`"["`), &matcher); err == nil {
				t.Error("expected an error for an invalid pattern")
			}
		})
	}
}
