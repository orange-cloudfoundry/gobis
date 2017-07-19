package casbin

import (
	"github.com/casbin/casbin"
	"github.com/casbin/casbin/persist"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	"github.com/mitchellh/mapstructure"
	"strings"
	"github.com/orange-cloudfoundry/gobis/proxy/ctx"
	log "github.com/sirupsen/logrus"
)

type CasbinHandler struct {
	next         http.Handler
	casbinOption *CasbinOption
}

func NewCasbinHandler(next http.Handler, casbinOption *CasbinOption) http.Handler {
	return &CasbinHandler{next, casbinOption}
}

func (h CasbinHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	gobisAdapter := NewGobisAdapter()
	gobisAdapter.AddPolicies(h.casbinOption.Policies...)
	gobisAdapter.AddPoliciesFromRequest(req)
	enforcer := newEnforcer(gobisAdapter, h.casbinOption.PermConf)
	if !h.CheckPermission(enforcer, req) {
		http.Error(w, http.StatusText(403), 403)
		return
	}

	h.next.ServeHTTP(w, req)
}



// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (h CasbinHandler) CheckPermission(e *casbin.Enforcer, r *http.Request) bool {
	user := ctx.Username(r)
	method := r.Method
	path := ctx.Path(r)
	path = strings.TrimSuffix(path, "/") + "/"
	return e.Enforce(user, path, method)
}

func newEnforcer(adapter persist.Adapter, modelConfText string) *casbin.Enforcer {
	if modelConfText == "" {
		modelConfText = MODEL_CONF
	}
	modelConf := casbin.NewModel()
	modelConf.LoadModelFromText(modelConfText)
	enableLog := log.GetLevel() == log.DebugLevel
	return casbin.NewEnforcer(modelConf, adapter, enableLog)
}

func Casbin(proxyRoute models.ProxyRoute, handler http.Handler) (http.Handler, error) {
	var config CasbinConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		return handler, err
	}
	if config.Casbin == nil || !config.Casbin.Enable {
		return handler, nil
	}
	return NewCasbinHandler(handler, config.Casbin), nil
}