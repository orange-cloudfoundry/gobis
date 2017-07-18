package casbin

const (
	PolicyContextKey CasbinContextKey = iota
)

type CasbinContextKey int

type CasbinConfig struct {
	Casbin *CasbinOption `mapstructure:"casbin" json:"casbin" yaml:"casbin"`
}
type CasbinOption struct {
	// Enable casbin access control
	Enable   bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// List of policies to load
	// middleware will load as role policies all group found by using `ctx.Groups(*http.Request)`
	// It will also load policies found in context `casbin.PolicyContextKey`
	Policies []CasbinPolicy `mapstructure:"policies" json:"policies" yaml:"policies"`
	// This is a perm conf in casbin format (see: https://github.com/casbin/casbin#examples )
	// by default this will be loaded:
	/*
	[request_definition]
	r = sub, obj, act

	[policy_definition]
	p = sub, obj, act

	[role_definition]
	g = _, _

	[policy_effect]
	e = some(where (p.eft == allow))

	[matchers]
	m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
	 */
	PermConf string `mapstructure:"perm_conf" json:"perm_conf" yaml:"perm_conf"`
}
type CasbinPolicy struct {
	// Type of policy, with default config it can be p (target) or g (role)
	Type string `mapstructure:"type" json:"type" yaml:"type"`
	// Subject of the policy, this can be a username retrieve basic auth or a role name
	// For example if use ldap middleware you can use username or a group where the user is member of
	Sub  string `mapstructure:"sub" json:"sub" yaml:"sub"`
	// Object of the policy, with default perm config it will be the following path set in your route
	// e.g.: with path = "/app/**" object will be /* to allow everything after /app
	Obj  string `mapstructure:"obj" json:"obj" yaml:"obj"`
	// Operation of the policy, with default config it will be an http method like GET, POST, ... or * for evything
	Act  string `mapstructure:"act" json:"act" yaml:"act"`
}

const MODEL_CONF = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")`