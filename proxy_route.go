package gobis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	PATH_REGEX = "(?i)^((/[^/\\*]*)*)(/((\\*){1,2}))?$"
)

type ProxyRoute struct {
	// Name of your route
	Name string `json:"name" yaml:"name"`
	// Path which gobis handler should listen to
	// You can use globs:
	//   - appending /* will only make requests available in first level of upstream
	//   - appending /** will pass everything to upstream
	// e.g.: /app/**
	Path string `json:"path" yaml:"path"`
	// Upstream url where all request will be redirected (if ForwardedHeader option not set)
	// Query parameters can be passed, e.g.: http://localhost?param=1
	// User and password are given as basic auth too (this is not recommended to use it), e.g.: http://user:password@localhost
	// Can be empty if ForwardedHeader is set
	// This is ignored if ForwardHandler is set
	Url string `json:"url" yaml:"url"`
	// If set upstream url will be took from the value of this header inside the received request
	// Url option will be used for the router to match host and path (if not empty) found in value of this header and host and path found in url (If NoUrlMatch is false)
	// this useful, for example, to create a cloud foundry route service: https://docs.cloudfoundry.org/services/route-services.html
	ForwardedHeader string `json:"forwarded_header" yaml:"forwarded_header"`
	// List of headers which should not be sent to upstream
	SensitiveHeaders []string `json:"sensitive_headers" yaml:"sensitive_headers"`
	// List of http methods allowed (Default: all methods are accepted)
	Methods []string `json:"methods" yaml:"methods"`
	// An url to an http proxy to make requests to upstream pass to this
	HttpProxy string `json:"http_proxy" yaml:"http_proxy"`
	// An url to an https proxy to make requests to upstream pass to this
	HttpsProxy string `json:"https_proxy" yaml:"https_proxy"`
	// Force to never use proxy even proxy from environment variables
	NoProxy bool `json:"no_proxy" yaml:"no_proxy"`
	// By default response from upstream are buffered, it can be issue when sending big files
	// Set to true to stream response
	NoBuffer bool `json:"no_buffer" yaml:"no_buffer"`
	// Set to true to not send X-Forwarded-* headers to upstream
	RemoveProxyHeaders bool `json:"remove_proxy_headers" yaml:"remove_proxy_headers"`
	// Set to true to not check ssl certificates from upstream (not really recommended)
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	// It was made to pass arbitrary params to use it after in gobis middlewares
	MiddlewareParams interface{} `json:"middleware_params" yaml:"middleware_params"`
	// This is the path without glob variables
	// Filled when unmarshal json or yaml or when running LoadParams on route
	AppPath string `json:"-" yaml:"-"`
	// Set to true to see errors on web page when there is a panic error on gobis
	ShowError bool `json:"show_error" yaml:"show_error"`
	// Chain others routes in a route
	Routes []ProxyRoute `json:"routes" yaml:"routes"`
	// Set an handler to use to forward request to this handler when using gobis programmatically
	ForwardHandler http.Handler `json:"-" yaml:"-"`
}

func (r *ProxyRoute) UnmarshalJSON(data []byte) error {
	type plain ProxyRoute
	err := json.Unmarshal(data, (*plain)(r))
	if err != nil {
		return err
	}
	err = r.Check()
	if err != nil {
		return err
	}
	r.LoadParams()
	return nil
}
func (r *ProxyRoute) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ProxyRoute
	var err error
	if err = unmarshal((*plain)(r)); err != nil {
		return err
	}
	err = r.Check()
	if err != nil {
		return err
	}
	r.LoadParams()
	return nil
}
func (r ProxyRoute) Check() error {
	if r.Name == "" {
		return fmt.Errorf("You must provide a name to your route")
	}
	if r.Path == "" {
		return fmt.Errorf("You must provide a path to your route")
	}
	if r.Url == "" && r.ForwardedHeader == "" {
		return fmt.Errorf("You must provide an url or forwarded header to your route")
	}

	reg := regexp.MustCompile(PATH_REGEX)
	if !reg.MatchString(r.Path) {
		return fmt.Errorf("Invalid path, e.g.: /api/** to match everything, /api/* to match first level or /api to only match this")
	}

	_, err := url.Parse(r.HttpProxy)
	if err != nil && r.HttpProxy != "" {
		return fmt.Errorf("Invalid http_proxy : " + err.Error())
	}
	_, err = url.Parse(r.HttpsProxy)
	if err != nil && r.HttpsProxy != "" {
		return fmt.Errorf("Invalid https_proxy : " + err.Error())
	}
	if r.Url == "" {
		return nil
	}
	routeUrl, err := url.Parse(r.Url)
	if err != nil {
		return fmt.Errorf("Invalid url : " + err.Error())
	}
	if routeUrl.Host == "localhost" || routeUrl.Host == "127.0.0.1" {
		return fmt.Errorf("Invalid url : host couldn't be localhost or 127.0.0.1")
	}
	if routeUrl.Scheme == "" {
		return fmt.Errorf("Invalid url : scheme is missing")
	}
	return nil
}
func (r ProxyRoute) PathAsStartPath() string {
	startPath := strings.TrimSuffix(r.Path, "/**")
	startPath = strings.TrimSuffix(startPath, "/*")
	return startPath
}
func (r *ProxyRoute) LoadParams() {
	reg := regexp.MustCompile(PATH_REGEX)
	r.AppPath = reg.FindStringSubmatch(r.Path)[1]
	r.Url = strings.TrimSuffix(r.Url, "/")
}
func (r ProxyRoute) CreateRoutePath(finalPath string) string {
	reg := regexp.MustCompile(PATH_REGEX)
	sub := reg.FindStringSubmatch(r.Path)
	return sub[1] + finalPath
}
func (r ProxyRoute) RequestPath(req *http.Request) string {
	if r.ForwardedHeader == "" {
		return req.URL.Path
	}
	upstream := req.Header.Get(r.ForwardedHeader)
	if upstream == "" {
		return req.URL.Path
	}
	upstreamUrl, _ := url.Parse(upstream)
	return upstreamUrl.Path
}
func (r ProxyRoute) UpstreamUrl(req *http.Request) *url.URL {
	if r.ForwardHandler != nil {
		req.URL.Path = ""
		return req.URL
	}
	var upstreamUrl *url.URL
	if r.ForwardedHeader == "" {
		upstreamUrl, _ = url.Parse(r.Url)
		return upstreamUrl
	}
	upstream := req.Header.Get(r.ForwardedHeader)
	if upstream == "" {
		upstreamUrl, _ = url.Parse(r.Url)
		return upstreamUrl
	}
	upstreamUrl, _ = url.Parse(upstream)
	upstreamUrl.Path = ""
	return upstreamUrl
}
func (r ProxyRoute) RouteMatcher() *regexp.Regexp {
	return createPathMatcher(r.Path)
}

func createPathMatcher(path string) *regexp.Regexp {
	reg := regexp.MustCompile(PATH_REGEX)
	sub := reg.FindStringSubmatch(path)
	muxRoute := regexp.QuoteMeta(sub[1])
	glob := sub[4]
	if glob == "" {
		return regexp.MustCompile(muxRoute)
	}
	if glob == "*" {
		return regexp.MustCompile(fmt.Sprintf("^%s(/[^/]*)?$", muxRoute))
	}
	return regexp.MustCompile(fmt.Sprintf("^%s(/.*)?$", muxRoute))
}
