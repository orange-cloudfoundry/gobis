package casbin

const (
	PolicyContextKey CasbinContextKey = iota
)

type CasbinContextKey int

type CasbinConfig struct {
	Casbin *CasbinOption `mapstructure:"casbin" json:"casbin" yaml:"casbin"`
}
type CasbinOption struct {
	Policies []CasbinPolicy `mapstructure:"policies" json:"policies" yaml:"policies"`
	PermConf string `mapstructure:"perm_conf" json:"perm_conf" yaml:"perm_conf"`
}
type CasbinPolicy struct {
	Type string `mapstructure:"type" json:"type" yaml:"type"`
	Sub  string `mapstructure:"sub" json:"sub" yaml:"sub"`
	Obj  string `mapstructure:"obj" json:"obj" yaml:"obj"`
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