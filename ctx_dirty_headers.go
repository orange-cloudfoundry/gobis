// Mark header as dirty to not forward those headers in the upstream url
// Useful for middleware when they ask for authorization header fox example
package gobis

import (
	"net/http"
	"strings"
)

const (
	dirtyHeadersKey GobisContextKey = iota
)

type GobisContextKey int

// Mark an http header as dirty
// Useful to prevent some headers added and used by middleware to not be sent to upstream
// if oldValue is not empty it will make proxy rewrite header with this value
func DirtHeader(req *http.Request, header string, oldValue ...string) {
	var dirtyHeaders map[string]string = make(map[string]string)
	header = sanitizeHeaderName(header)
	oldVal := ""
	if len(oldVal) > 0 {
		oldVal = oldValue[0]
	}
	dirtyHeadersPtr := DirtyHeaders(req)
	if dirtyHeadersPtr == nil {
		dirtyHeaders[header] = oldVal
		AddContextValue(req, dirtyHeadersKey, &dirtyHeaders)
		return
	}
	dirtyHeaders = *dirtyHeadersPtr
	dirtyHeaders[header] = oldVal
	*dirtyHeadersPtr = dirtyHeaders
}

// Return true if an http header is marked as dirty
func IsDirtyHeader(req *http.Request, header string) bool {
	header = sanitizeHeaderName(header)
	dirtyHeadersPtr := DirtyHeaders(req)
	if dirtyHeadersPtr == nil {
		return false
	}
	dirtyHeaders := *dirtyHeadersPtr
	_, ok := dirtyHeaders[header]
	return ok
}

// Remove an http header from the list of dirty header
func UndirtHeader(req *http.Request, header string) {
	header = sanitizeHeaderName(header)
	dirtyHeadersPtr := DirtyHeaders(req)
	if dirtyHeadersPtr == nil {
		return
	}
	dirtyHeaders := *dirtyHeadersPtr
	delete(dirtyHeaders, header)
	*dirtyHeadersPtr = dirtyHeaders
}

// Retrieve all http headers marked as dirty
func DirtyHeaders(req *http.Request) *map[string]string {
	var dirtyHeaders *map[string]string
	InjectContextValue(req, dirtyHeadersKey, &dirtyHeaders)
	return dirtyHeaders
}

func sanitizeHeaderName(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}