// Mark header as dirty to not forward those headers in the upstream url
// Useful for middleware when they ask for authorization header fox example
package ctx

import (
	"net/http"
	"strings"
)

const (
	dirtyHeadersKey GobisContextKey = iota
)

type GobisContextKey int

func DirtHeader(req *http.Request, header string, oldValues ...string) {
	var dirtyHeaders map[string]string = make(map[string]string)
	header = sanitizeHeaderName(header)
	oldValue := ""
	if len(oldValues) > 0 {
		oldValue = oldValues[0]
	}
	dirtyHeadersPtr := DirtyHeaders(req)
	if dirtyHeadersPtr == nil {
		dirtyHeaders[header] = oldValue
		AddContextValue(req, dirtyHeadersKey, &dirtyHeaders)
		return
	}
	dirtyHeaders = *dirtyHeadersPtr
	dirtyHeaders[header] = oldValue
	*dirtyHeadersPtr = dirtyHeaders
}
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
func DirtyHeaders(req *http.Request) *map[string]string {
	var dirtyHeaders *map[string]string
	InjectContextValue(req, dirtyHeadersKey, &dirtyHeaders)
	return dirtyHeaders
}
func sanitizeHeaderName(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}