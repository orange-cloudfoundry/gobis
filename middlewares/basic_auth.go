package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	log "github.com/sirupsen/logrus"
	"github.com/goji/httpauth"
	"net/http"
	"crypto/sha256"
	"crypto/subtle"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/blowfish"
)

type BasicAuthOptions []BasicAuthOption
type BasicAuthConfig struct {
	BasicAuth BasicAuthOptions `mapstructure:"basic_auth" json:"basic_auth" yaml:"basic_auth"`
}
type BasicAuthOption struct {
	User     string `mapstructure:"user" json:"user" yaml:"user"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	Crypted  bool `mapstructure:"crypted" json:"crypted" yaml:"crypted"`
}

func (b BasicAuthOptions) Auth(user string, password string, req *http.Request) bool {
	foundUser := b.findByUser(user)
	if foundUser.User == "" {
		return false
	}
	// Compare the supplied credentials to those set in our options
	if foundUser.Crypted {
		err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(password))
		if err == nil {
			return true
		}
		if _, ok := err.(blowfish.KeySizeError); ok {
			log.Errorf(
				"orange-cloudfoundry/gobis/middlewares: Basic auth middleware, invalid crypted password for user '%s': %s",
				foundUser.User,
				err.Error(),
			)
		}
		return false
	}
	// Equalize lengths of supplied and required credentials
	// by hashing them
	givenUser := sha256.Sum256([]byte(user))
	givenPass := sha256.Sum256([]byte(password))
	requiredUser := sha256.Sum256([]byte(foundUser.User))
	requiredPass := sha256.Sum256([]byte(foundUser.Password))
	return subtle.ConstantTimeCompare(givenUser[:], requiredUser[:]) == 1 &&
		subtle.ConstantTimeCompare(givenPass[:], requiredPass[:]) == 1
}
func (b BasicAuthOptions) findByUser(user string) BasicAuthOption {
	for _, basicAuthConfig := range b {
		if basicAuthConfig.User == user {
			return basicAuthConfig
		}
	}
	return BasicAuthOption{}
}
func BasicAuth(proxyRoute models.ProxyRoute, handler http.Handler) http.Handler {
	entry := log.WithField("route_name", proxyRoute.Name)
	var config BasicAuthConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding basic auth middleware failed: " + err.Error())
		return handler
	}
	if len(config.BasicAuth) == 0 {
		return handler
	}
	entry.Debug("orange-cloudfoundry/gobis/middlewares: Adding basic auth middleware.")
	return httpauth.BasicAuth(httpauth.AuthOptions{
		AuthFunc: config.BasicAuth.Auth,
	})(handler)
}
