package gobis

import "net/http"

const (
	pathContextKey RouterContextKey = iota
	routeNameContextKey
)

type RouterContextKey int


// Set the rest of the path from a request url to his context (used by router factory)
func SetPath(req *http.Request, path string) {
	AddContextValue(req, pathContextKey, path)
}

// Retrieve rest of the path from a request url to his context
func Path(req *http.Request) string {
	var path string
	InjectContextValue(req, pathContextKey, &path)
	return path
}

func setRouteName(req *http.Request, routeName string) {
	AddContextValue(req, routeNameContextKey, routeName)
}

// Retrieve routes name used in this request
func RouteName(req *http.Request) string {
	var routeName string
	InjectContextValue(req, routeNameContextKey, &routeName)
	return routeName
}
