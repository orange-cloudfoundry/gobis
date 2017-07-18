// Mark header as dirty to not forward those headers in the upstream url
// Useful for middleware when they ask for authorization header fox example
package ctx

import (
	"net/http"
	"context"
	"strings"
)

const (
	dirtyHeadersKey GobisContextKey = iota
)

type GobisContextKey int

func DirtHeader(req *http.Request, header string, oldValues ...string) {
	header = sanitizeHeaderName(header)
	dirtyHeaders := GetDirtyHeaders(req)
	oldValue := ""
	if len(oldValues) > 0 {
		oldValue = oldValues[0]
	}
	dirtyHeaders[header] = oldValue
	*req = *req.WithContext(context.WithValue(req.Context(), dirtyHeadersKey, dirtyHeaders))
}
func IsDirtyHeader(req *http.Request, header string) bool {
	header = sanitizeHeaderName(header)
	dirtyHeaders := GetDirtyHeaders(req)
	_, ok := dirtyHeaders[header]
	return ok
}
func UndirtHeader(req *http.Request, header string) {
	header = sanitizeHeaderName(header)
	dirtyHeaders := GetDirtyHeaders(req)
	delete(dirtyHeaders, header)
	*req = *req.WithContext(context.WithValue(req.Context(), dirtyHeadersKey, dirtyHeaders))
}
func GetDirtyHeaders(req *http.Request) map[string]string {
	values := req.Context().Value(dirtyHeadersKey)
	if values == nil {
		return make(map[string]string)
	}
	dirtyHeaders, ok := values.(map[string]string)
	if !ok {
		return make(map[string]string)
	}
	return dirtyHeaders
}
func sanitizeHeaderName(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}