package gobis

import (
	"net/http"
)

type MiddlewareHandler interface {
	Handler(route ProxyRoute, params interface{}, next http.Handler) (http.Handler, error)
	Schema() interface{}
}
