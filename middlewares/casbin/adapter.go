package casbin

import (
	"github.com/casbin/casbin/model"
	"net/http"
	"github.com/orange-cloudfoundry/gobis/middlewares"
)

type GobisAdapter struct {
	policies []CasbinPolicy
}

func NewGobisAdapter() *GobisAdapter {
	return &GobisAdapter{policies: make([]CasbinPolicy, 0)}
}

func (a *GobisAdapter) LoadPolicy(model model.Model) error {
	for _, policy := range a.policies {
		err := a.loadPolicy(policy, model)
		if err != nil {
			return err
		}
	}
	return nil
}
func (a *GobisAdapter) loadPolicy(policy CasbinPolicy, model model.Model) error {
	tokens := []string{policy.Sub, policy.Obj}
	if policy.Act != "" {
		tokens = append(tokens, policy.Act)
	}
	key := policy.Type
	sec := key[:1]
	model[sec][key].Policy = append(
		model[sec][key].Policy,
		tokens,
	)
	return nil
}
func (a GobisAdapter) SavePolicy(model model.Model) error {
	return nil
}
func (a *GobisAdapter) AddPolicies(policies ...CasbinPolicy) {
	a.policies = append(a.policies, policies...)
}
func (a *GobisAdapter) AddPoliciesFromRequest(req *http.Request) {
	user, _, _ := req.BasicAuth()
	if user == "" {
		return
	}
	ctxGroupVal := req.Context().Value(middlewares.GroupContextKey)
	if ctxGroupVal == nil {
		return
	}
	groups := ctxGroupVal.([]string)
	for _, group := range groups {
		a.AddPolicies(CasbinPolicy{
			Type: "g",
			Sub: user,
			Obj: group,
		})
	}

	ctxPolicyVal := req.Context().Value(PolicyContextKey)
	if ctxPolicyVal == nil {
		return
	}
	ctxPolicies := ctxPolicyVal.([]CasbinPolicy)
	for _, ctxPolicy := range ctxPolicies {
		a.AddPolicies(ctxPolicy)
	}
}