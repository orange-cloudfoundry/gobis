package gobis

import "net/http"

const (
	PathContextKey RouterContextKey = iota
	RouteNameContextKey
)

type RouterContextKey int


// Set the rest of the path from a request url to his context (used by router factory)
func SetPath(req *http.Request, path string) {
	pathPtr := pathPtr(req)
	if pathPtr == nil {
		AddContextValue(req, PathContextKey, &path)
		return
	}
	*pathPtr = path
}

// Retrieve rest of the path from a request url to his context
func Path(req *http.Request) string {
	pathPtr := pathPtr(req)
	if pathPtr == nil {
		return ""
	}
	return *pathPtr
}
func pathPtr(req *http.Request) *string {
	var path *string
	InjectContextValue(req, PathContextKey, &path)
	return path
}

func setRouteName(req *http.Request, routeName string) {
	ptr := routeNamePtr(req)
	if ptr == nil {
		AddContextValue(req, RouteNameContextKey, &routeName)
		return
	}
	*ptr = routeName
}

// Retrieve route name used in this request
func RouteName(req *http.Request) string {
	ptr := routeNamePtr(req)
	if ptr == nil {
		return ""
	}
	return *ptr
}
func routeNamePtr(req *http.Request) *string {
	var routeName *string
	InjectContextValue(req, RouteNameContextKey, &routeName)
	return routeName
}