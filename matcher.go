package gobis

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/glob"
	"regexp"
)

const (
	PathRegex = "(?i)^((/[^/\\*]*)*)(/((\\*){1,2}))?$"
)

type PathMatcher struct {
	pathMatcher *regexp.Regexp
	expr        string
	appPath     string
}

func NewPathMatcher(path string) *PathMatcher {
	err := checkPathMatcher(path)
	if err != nil {
		panic(err)
	}

	return &PathMatcher{
		pathMatcher: generatePathMatcher(path),
		expr:        path,
		appPath:     generateRawPath(path),
	}
}

func (re PathMatcher) String() string {
	return re.expr
}

func (re PathMatcher) AppPath() string {
	return re.appPath
}

func (re PathMatcher) CreateRoutePath(finalPath string) string {
	return re.appPath + finalPath
}

func (re *PathMatcher) UnmarshalCloud(data interface{}) error {
	return re.load(data.(string))
}

func (re *PathMatcher) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	return re.load(s)
}

func (re *PathMatcher) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	return re.load(s)
}

func (re *PathMatcher) load(s string) error {
	err := checkPathMatcher(s)
	if err != nil {
		return err
	}
	re.pathMatcher = generatePathMatcher(s)
	re.expr = s
	re.appPath = generateRawPath(s)
	return nil
}

func checkPathMatcher(path string) error {
	reg := regexp.MustCompile(PathRegex)
	if !reg.MatchString(path) {
		return fmt.Errorf("Invalid path, e.g.: /api/** to match everything, /api/* to match first level or /api to only match this")
	}
	return nil
}

func generateRawPath(path string) string {
	reg := regexp.MustCompile(PathRegex)
	sub := reg.FindStringSubmatch(path)
	return sub[1]
}

func generatePathMatcher(path string) *regexp.Regexp {
	var pathMatcher *regexp.Regexp
	reg := regexp.MustCompile(PathRegex)
	sub := reg.FindStringSubmatch(path)
	muxRoute := regexp.QuoteMeta(sub[1])
	globSub := sub[4]
	switch globSub {
	case "*":
		pathMatcher = regexp.MustCompile(fmt.Sprintf("^%s(/[^/]*)?$", muxRoute))
	case "**":
		pathMatcher = regexp.MustCompile(fmt.Sprintf("^%s(/.*)?$", muxRoute))
	default:
		pathMatcher = regexp.MustCompile(fmt.Sprintf("^%s$", muxRoute))
	}
	return pathMatcher
}

type HostMatchers []*HostMatcher

func (m HostMatchers) Match(s string) bool {
	for _, matcher := range m {
		if matcher.Match(s) {
			return true
		}
	}
	return false
}

type HostMatcher struct {
	glob.Glob
	raw string
}

func NewHostMatcher(hostOrWildcard string) *HostMatcher {
	return &HostMatcher{
		Glob: glob.MustCompile(hostOrWildcard, '.'),
		raw:  hostOrWildcard,
	}
}

func (re *HostMatcher) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	re.Glob, err = glob.Compile(s, '.')
	re.raw = s
	return err
}

func (re *HostMatcher) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	re.Glob, err = glob.Compile(s, '.')
	re.raw = s
	return err
}

func (re HostMatcher) String() string {
	return re.raw
}
