package proxy

import (
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
)

type RouterMiddlewareFunc func(models.ProxyRoute, http.Handler) (http.Handler, error)
