package gobis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type ProxyRoute struct {
	// Name of your routes
	Name string `json:"name" yaml:"name"`
	// Path which gobis handler should listen to
	// You can use globs:
	//   - appending /* will only make requests available in first level of upstream
	//   - appending /** will pass everything to upstream
	// e.g.: /app/**
	Path *PathMatcher `json:"path" yaml:"path"`
	// Url Upstream url where all request will be redirected (if ForwardedHeader option not set)
	// Query parameters can be passed, e.g.: http://localhost?param=1
	// User and password are given as basic auth too (this is not recommended to use it), e.g.: http://user:password@localhost
	// Can be empty if ForwardedHeader is set
	// This is ignored if ForwardHandler is set
	Url string `json:"url" yaml:"url"`
	// ForwardedHeader If set upstream url will be taken from the value of this header inside the received request
	// Url option will be used for the router to match host and path (if not empty) found in value of this header and host and path found in url (If NoUrlMatch is false)
	// this useful, for example, to create a cloud foundry routes service: https://docs.cloudfoundry.org/services/route-services.html
	ForwardedHeader string `json:"forwarded_header" yaml:"forwarded_header"`
	// SensitiveHeaders List of headers which should not be sent to upstream
	SensitiveHeaders []string `json:"sensitive_headers" yaml:"sensitive_headers"`
	// Methods List of http methods allowed (Default: all methods are accepted)
	Methods []string `json:"methods" yaml:"methods"`
	// HttpProxy An url to a http proxy to make requests to upstream pass to this
	HttpProxy string `json:"http_proxy" yaml:"http_proxy"`
	// HttpsProxy An url to a https proxy to make requests to upstream pass to this
	HttpsProxy string `json:"https_proxy" yaml:"https_proxy"`
	// NoProxy Force to never use proxy even proxy from environment variables
	NoProxy bool `json:"no_proxy" yaml:"no_proxy"`
	// NoBuffer Responses from upstream are buffered by default, it can be issue when sending big files
	// Set to true to stream response
	NoBuffer bool `json:"no_buffer" yaml:"no_buffer"`
	// RemoveProxyHeaders Set to true to not send X-Forwarded-* headers to upstream
	RemoveProxyHeaders bool `json:"remove_proxy_headers" yaml:"remove_proxy_headers"`
	// InsecureSkipVerify Set to true to not check ssl certificates from upstream (not really recommended)
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	// MiddlewareParams It was made to pass arbitrary params to use it after in gobis middlewares
	// This can be a structure (to set them programmatically) or a map[string]interface{} (to set them from a config file)
	MiddlewareParams interface{} `json:"middleware_params" yaml:"middleware_params"`
	// ShowError Set to true to see errors on web page when there is a panic error on gobis
	ShowError bool `json:"show_error" yaml:"show_error"`
	// UseFullPath Set to true to use full path
	// e.g.: path=/metrics/** and request=/metrics/foo this will be redirected to /metrics/foo on upstream instead of /foo
	UseFullPath bool `json:"use_full_path" yaml:"use_full_path"`
	// Routes Chain others routes in a routes
	Routes []ProxyRoute `json:"routes" yaml:"routes"`
	// ForwardHandler Set a handler to use to forward request to this handler when using gobis programmatically
	ForwardHandler http.Handler `json:"-" yaml:"-"`
	// OptionsPassthrough Will forward directly to proxied route OPTIONS method without using middlewares
	OptionsPassthrough bool `json:"options_passthrough" yaml:"options_passthrough"`
	// HostsPassthrough Will forward directly to proxied route without using middlewares when http header host given match one of host in this list
	// Wildcard are allowed
	// E.g.: - *.my.passthroughurl.com -> this will allow all routes matching this wildcard to passthrough middleware
	// **Warning**: host header can be forged by user, this may be a security issue if not used properly.
	HostsPassthrough HostMatchers `json:"hosts_passthrough" yaml:"hosts_passthrough"`
}

func (r *ProxyRoute) UnmarshalJSON(data []byte) error {
	type plain ProxyRoute
	err := json.Unmarshal(data, (*plain)(r))
	if err != nil {
		return err
	}
	return r.Check()
}

func (r *ProxyRoute) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ProxyRoute
	var err error
	if err = unmarshal((*plain)(r)); err != nil {
		return err
	}
	return r.Check()
}

func (r ProxyRoute) Check() error {
	if r.Name == "" {
		return fmt.Errorf("you must provide a name to your routes")
	}
	if r.Path == nil {
		return fmt.Errorf("you must provide a path to your routes")
	}
	if r.Url == "" && r.ForwardedHeader == "" {
		return fmt.Errorf("you must provide an URL or forwarded header to your routes")
	}

	_, err := url.Parse(r.HttpProxy)
	if err != nil && r.HttpProxy != "" {
		return fmt.Errorf("invalid http_proxy : %s", err.Error())
	}
	_, err = url.Parse(r.HttpsProxy)
	if err != nil && r.HttpsProxy != "" {
		return fmt.Errorf("invalid https_proxy : %s", err.Error())
	}
	if r.Url == "" {
		return nil
	}
	routeUrl, err := url.Parse(r.Url)
	if err != nil {
		return fmt.Errorf("Invalid url : %s", err.Error())
	}
	if routeUrl.Host == "localhost" || routeUrl.Host == "127.0.0.1" {
		return fmt.Errorf("invalid URL: host couldn't be localhost or 127.0.0.1")
	}
	if routeUrl.Scheme == "" {
		return fmt.Errorf("invalid URL: scheme is missing")
	}
	return nil
}

func (r ProxyRoute) PathAsStartPath() string {
	startPath := strings.TrimSuffix(r.Path.String(), "/**")
	startPath = strings.TrimSuffix(startPath, "/*")
	return startPath
}

func (r ProxyRoute) CreateRoutePath(finalPath string) string {
	return r.Path.CreateRoutePath(finalPath)
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
	origPath := ""
	if r.UseFullPath {
		origPath = r.PathAsStartPath() + "/"
	}
	if r.ForwardHandler != nil {
		req.URL.Path = origPath
		return req.URL
	}
	var upstreamUrl *url.URL
	if r.ForwardedHeader == "" {
		upstreamUrl, _ = url.Parse(r.Url)
		upstreamUrl.Path = origPath + upstreamUrl.Path
		return upstreamUrl
	}
	upstream := req.Header.Get(r.ForwardedHeader)
	if upstream == "" {
		upstreamUrl, _ = url.Parse(r.Url)
		upstreamUrl.Path = origPath + upstreamUrl.Path
		return upstreamUrl
	}
	upstreamUrl, _ = url.Parse(upstream)
	upstreamUrl.Path = origPath
	return upstreamUrl
}

func (r ProxyRoute) RouteMatcher() *regexp.Regexp {
	return r.Path.pathMatcher
}
