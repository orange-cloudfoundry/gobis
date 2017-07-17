package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"encoding/json"
)

const (
	PATH_REGEX = "(?i)^((/[^/\\*]*)*)(/((\\*){1,2}))?$"
	MUX_REST_VAR_KEY = "rest"
)

type ProxyRoute struct {
	// Name of your route
	Name               string `json:"name" yaml:"name"`
	// Path which gobis handler should listen to
	// You can use globs:
	//   - appending /* will only make requests available in first level of upstream
	//   - appending /** will pass everything to upstream
	// e.g.: /app/**
	Path             string `json:"path" yaml:"path"`
	// Upstream url where all request will be redirected
	// Query parameters can be passed, e.g.: http://localhost?param=1
	// User and password are given as basic auth too (this is not recommended to use it), e.g.: http://user:password@localhost
	Url              string `json:"url" yaml:"url"`
	// List of headers which should not be sent to upstream
	SensitiveHeaders []string `json:"sensitive_headers" yaml:"sensitive_headers"`
	// List of http methods allowed (Default: all methods are accepted)
	Methods          []string `json:"methods" yaml:"methods"`
	// An url to an http proxy to make requests to upstream pass to this
	HttpProxy        string `json:"http_proxy" yaml:"http_proxy"`
	// An url to an https proxy to make requests to upstream pass to this
	HttpsProxy       string `json:"https_proxy" yaml:"https_proxy"`
	// Force to never use proxy even proxy from environment variables
	NoProxy            bool `json:"no_proxy" yaml:"no_proxy"`
	// By default response from upstream are buffered, it can be issue when sending big files
	// Set to true to stream response
	NoBuffer           bool `json:"no_buffer" yaml:"no_buffer"`
	// Set to true to not send X-Forwarded-* headers to upstream
	RemoveProxyHeaders bool `json:"remove_proxy_headers" yaml:"remove_proxy_headers"`
	// Set to true to not check ssl certificates from upstream (not really recommended)
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	// It was made to pass arbitrary params to use it after in gobis middlewares
	ExtraParams        map[string]interface{} `json:"extra_params" yaml:"extra_params"`
	// This is the path without glob variables
	// Filled when unmarshal json or yaml or when running LoadParams on route
	AppPath            string `json:"-" yaml:"-"`
}

func (r *ProxyRoute) UnmarshalJSON(data []byte) (error) {
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
func (r *ProxyRoute) UnmarshalYAML(unmarshal func(interface{}) error) (error) {
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
	if r.Url == "" {
		return fmt.Errorf("You must provide an url to your route")
	}

	reg := regexp.MustCompile(PATH_REGEX)
	if !reg.MatchString(r.Path) {
		return fmt.Errorf("Invalid path, e.g.: /api/** to match everything, /api/* to match first level or /api to only match this")
	}

	routeUrl, err := url.Parse(r.Url)
	if err != nil {
		return fmt.Errorf("Invalid url : " + err.Error())
	}
	if routeUrl.Host == "localhost" || routeUrl.Host == "127.0.0.1" {
		return fmt.Errorf("Invalid url : host couldn't be localhost or 127.0.0.1")
	}
	_, err = url.Parse(r.HttpProxy)
	if err != nil && r.HttpProxy != "" {
		return fmt.Errorf("Invalid http_proxy : " + err.Error())
	}
	_, err = url.Parse(r.HttpsProxy)
	if err != nil && r.HttpsProxy != "" {
		return fmt.Errorf("Invalid https_proxy : " + err.Error())
	}
	return nil
}

func (r *ProxyRoute) LoadParams() {
	reg := regexp.MustCompile(PATH_REGEX)
	r.AppPath = reg.FindStringSubmatch(r.Path)[1]
	r.Url = strings.TrimSuffix(r.Url, "/")
}

func (r ProxyRoute) MuxRoute() string {
	reg := regexp.MustCompile(PATH_REGEX)
	sub := reg.FindStringSubmatch(r.Path)
	muxRoute := sub[1]
	glob := sub[4]
	if glob == "" {
		return muxRoute
	}
	if glob == "*" {
		return fmt.Sprintf("%s{%s:(?:/[^/]*)?}", muxRoute, MUX_REST_VAR_KEY)
	}
	return fmt.Sprintf("%s{%s:(?:/.*)?}", muxRoute, MUX_REST_VAR_KEY)
}