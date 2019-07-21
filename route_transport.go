package gobis

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var protectedHeaders map[string]bool = map[string]bool{}

type RouteTransport struct {
	route         ProxyRoute
	httpTransport *http.Transport
}

const (
	XForwardedProto  = "X-Forwarded-Proto"
	XForwardedFor    = "X-Forwarded-For"
	XForwardedHost   = "X-Forwarded-Host"
	XForwardedServer = "X-Forwarded-Server"
)

func NewRouteTransport(route ProxyRoute) http.RoundTripper {
	return NewRouteTransportWithHttpTransport(route, NewDefaultTransport())
}

func NewRouteTransportWithHttpTransport(route ProxyRoute, httpTransport *http.Transport) http.RoundTripper {
	routeTransport := &RouteTransport{
		route:         route,
		httpTransport: httpTransport,
	}
	routeTransport.InitHttpTransport()
	return routeTransport
}

func (r *RouteTransport) InitHttpTransport() {
	r.httpTransport.Proxy = r.ProxyFromRouteOrEnv
	if r.httpTransport.TLSClientConfig == nil {
		r.httpTransport.TLSClientConfig = &tls.Config{}
	}
	r.httpTransport.TLSClientConfig.InsecureSkipVerify = r.route.InsecureSkipVerify
}

func (r *RouteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.TransformRequest(req)
	return r.httpTransport.RoundTrip(req)
}

func (r *RouteTransport) TransformRequest(req *http.Request) {
	sensitiveHeaders := r.route.SensitiveHeaders
	if r.route.RemoveProxyHeaders {
		sensitiveHeaders = append(sensitiveHeaders,
			[]string{
				XForwardedProto,
				XForwardedFor,
				XForwardedHost,
				XForwardedServer,
			}...)
	}
	for _, sensitiveHeader := range sensitiveHeaders {
		sensitiveHeader = strings.TrimSpace(sensitiveHeader)
		if _, ok := protectedHeaders[strings.ToLower(sensitiveHeader)]; ok {
			continue
		}
		req.Header.Del(sensitiveHeader)
	}
}

func (r *RouteTransport) ProxyFromRouteOrEnv(req *http.Request) (*url.URL, error) {
	if r.route.NoProxy {
		return nil, nil
	}
	if (req.URL.Scheme == "https" && r.route.HttpsProxy == "") ||
		(req.URL.Scheme == "http" && r.route.HttpProxy == "") {
		return http.ProxyFromEnvironment(req)
	}
	var proxy string
	if req.URL.Scheme == "https" {
		proxy = r.route.HttpsProxy
	} else {
		proxy = r.route.HttpProxy
	}
	proxyURL, err := url.Parse(proxy)
	if err != nil || !strings.HasPrefix(proxyURL.Scheme, "http") {
		if proxyURL, err := url.Parse("http://" + proxy); err == nil {
			return proxyURL, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("invalid proxy address %q: %v", proxy, err)
	}
	return proxyURL, nil
}

func NewDefaultTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func SetProtectedHeaders(protectHeaders []string) {
	protectedHeaders = make(map[string]bool)
	for _, protectHeader := range protectHeaders {
		protectedHeaders[strings.ToLower(protectHeader)] = true
	}
}
