package gobis

import (
	"net/http"
)

type RouterMiddlewareFunc func(ProxyRoute, http.Handler) (http.Handler, error)
