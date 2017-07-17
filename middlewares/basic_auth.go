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
	entry := log.WithField("route_name", proxyRoute.Name)
	var config BasicAuthConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding basic auth middleware failed: " + err.Error())
		return handler
	}
	options := config.BasicAuth
	if options == nil {
		return handler
	}
	entry.Debug("orange-cloudfoundry/gobis/middlewares: Adding basic auth middleware.")
	return httpauth.SimpleBasicAuth(options.User, options.Password)(handler)
}
