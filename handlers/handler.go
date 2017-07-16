package handlers

import (
	"net/http"
)

type GobisHandler interface {
	GetServerAddr() string
	ServeHTTP(http.ResponseWriter, *http.Request)
}
