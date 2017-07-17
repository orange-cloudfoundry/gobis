package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	log "github.com/sirupsen/logrus"
	"github.com/goji/httpauth"
	"net/http"
)

type BasicAuthConfig struct {
	BasicAuth *BasicAuthOptions `mapstructure:"basic_auth" json:"basic_auth" yaml:"basic_auth"`
}
type BasicAuthOptions struct {
	User     string `mapstructure:"user" json:"user" yaml:"user"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}

func BasicAuth(proxyRoute models.ProxyRoute, handler http.Handler) http.Handler {
	var config BasicAuthConfig
	mapstructure.Decode(proxyRoute.ExtraParams, &config)
	options := config.BasicAuth
	if options == nil {
		return handler
	}
	log.Debug("orange-cloudfoundry/gobis/middlewares: Adding basic auth to route.")
	return httpauth.SimpleBasicAuth(options.User, options.Password)(handler)
}
