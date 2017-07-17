package casbin

import (
	"github.com/casbin/casbin"
	"github.com/casbin/casbin/persist"
	"github.com/orange-cloudfoundry/gobis/models"
	"net/http"
	"github.com/mitchellh/mapstructure"
	"github.com/gorilla/mux"
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
	user, _, _ := r.BasicAuth()
	method := r.Method
	path := ""
	vars := mux.Vars(r)
	if vars != nil {
		path = vars[models.MUX_REST_VAR_KEY]
	}
	if path == "" {
		path = "/"
	}
	return e.Enforce(user, path, method)
}

func newEnforcer(adapter persist.Adapter, modelConfText string) *casbin.Enforcer {
	if modelConfText == "" {
		modelConfText = MODEL_CONF
	}
	modelConf := casbin.NewModel()
	modelConf.LoadModelFromText(modelConfText)
	return casbin.NewEnforcer(modelConf, adapter, false)
}

func Casbin(proxyRoute models.ProxyRoute, handler http.Handler) (http.Handler, error) {
	var config CasbinConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		return handler, err
	}

	if config.Casbin == nil {
		return handler, nil
	}
	if len(config.Casbin.Policies) == 0 {
		return handler, nil
	}
	return NewCasbinHandler(handler, config.Casbin), nil
}