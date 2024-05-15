package gobis

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

const (
	pathContextKey RouterContextKey = iota
	routeNameContextKey
)

type RouterContextKey int

// SetPath Set the rest of the path from a request url to his context (used by router factory)
func SetPath(req *http.Request, path string) {
	AddContextValue(req, pathContextKey, path)
}

// Path Retrieve rest of the path from a request url to his context
func Path(req *http.Request) string {
	var path string
	if err := InjectContextValue(req, pathContextKey, &path); err != nil {
		log.Errorf("got error when injecting context value: %s", err)
	}
	return path
}

func setRouteName(req *http.Request, routeName string) {
	AddContextValue(req, routeNameContextKey, routeName)
}

// RouteName Retrieve routes name used in this request
func RouteName(req *http.Request) string {
	var routeName string
	if err := InjectContextValue(req, routeNameContextKey, &routeName); err != nil {
		log.Errorf("got error when injecting context value: %s", err)
	}
	return routeName
}
