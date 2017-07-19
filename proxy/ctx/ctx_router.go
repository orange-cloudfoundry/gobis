package ctx

import "net/http"

const (
	PathContextKey RouterContextKey = iota
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
	var username *string
	InjectContextValue(req, PathContextKey, &username)
	return username
}