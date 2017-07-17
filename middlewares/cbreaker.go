package middlewares

import (
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	log "github.com/sirupsen/logrus"
	"github.com/vulcand/oxy/cbreaker"
	"github.com/orange-cloudfoundry/gobis/proxy"
	"time"
)

type CircuitBreakerConfig struct {
	CircuitBreaker *CircuitBreakerOptions `mapstructure:"circuit_breaker" json:"circuit_breaker" yaml:"circuit_breaker"`
}
type CircuitBreakerOptions struct {
	// enable conn limit middleware
	Enable           bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// Limit number of simultaneous connection (default to 20)
	Expression       string `mapstructure:"expression" json:"expression" yaml:"expression"`
	// Identify request source to limit the source
	// possible value are 'client.ip', 'request.host' or 'request.header.X-My-Header-Name'
	// (default: client.ip)
	FallbackUrl      string `mapstructure:"fallback_url" json:"fallback_url" yaml:"fallback_url"`
	// FallbackDuration is how long the CircuitBreaker will remain in the Tripped in second
	// state before trying to recover.
	FallbackDuration int64 `mapstructure:"fallback_duration" json:"fallback_duration" yaml:"fallback_duration"`
	// RecoveryDuration is how long the CircuitBreaker will take to ramp up in second
	// requests during the Recovering state.
	RecoveryDuration int64 `mapstructure:"recovery_duration" json:"recovery_duration" yaml:"recovery_duration"`
	// CheckPeriod is how long the CircuitBreaker will wait between successive in second
	// checks of the breaker condition.
	CheckPeriod      int64 `mapstructure:"check_period" json:"check_period" yaml:"check_period"`
}

func CircuitBreaker(proxyRoute models.ProxyRoute, handler http.Handler) http.Handler {
	entry := log.WithField("route_name", proxyRoute.Name)
	var config CircuitBreakerConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding circuit breaker middleware failed: " + err.Error())
		return handler
	}
	options := config.CircuitBreaker
	if options == nil || !options.Enable {
		return handler
	}
	if options.Expression == "" {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding circuit breaker middleware failed: expression can't be empty")
		return handler
	}
	if options.FallbackUrl == "" {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding circuit breaker middleware failed: fallback url can't be empty")
		return handler
	}
	routerFactory := proxy.NewRouterFactory()
	proxyRoute.Url = options.FallbackUrl
	proxyRoute.Methods = []string{}
	proxyRoute.RemoveProxyHeaders = false
	proxyRoute.Name = proxyRoute.Name + " fallback"
	fallbackHandler, err := routerFactory.CreateForwardHandler(proxyRoute)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding circuit breaker middleware failed: " + err.Error())
		return handler
	}
	cbreakerOptions := []cbreaker.CircuitBreakerOption{cbreaker.Fallback(fallbackHandler)}
	if options.FallbackDuration > 0 {
		cbreakerOptions = append(
			cbreakerOptions,
			cbreaker.FallbackDuration(time.Second * time.Duration(options.FallbackDuration)),
		)
	}
	if options.RecoveryDuration > 0 {
		cbreakerOptions = append(
			cbreakerOptions,
			cbreaker.RecoveryDuration(time.Second * time.Duration(options.RecoveryDuration)),
		)
	}
	if options.CheckPeriod > 0 {
		cbreakerOptions = append(
			cbreakerOptions,
			cbreaker.CheckPeriod(time.Second * time.Duration(options.CheckPeriod)),
		)
	}
	finalHandler, err := cbreaker.New(handler, options.Expression, cbreakerOptions...)
	if err != nil {
		entry.Errorf("orange-cloudfoundry/gobis/middlewares: Adding circuit breaker middleware failed: " + err.Error())
		return handler
	}

	entry.Debug("orange-cloudfoundry/gobis/middlewares:: Adding circuit breaker middleware.")
	return finalHandler
}
