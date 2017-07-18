package middlewares

import (
	"gopkg.in/ldap.v2"
	"net/http"
	"fmt"
	log "github.com/sirupsen/logrus"
	"crypto/tls"
	"github.com/mitchellh/mapstructure"
	"github.com/orange-cloudfoundry/gobis/models"
	"os"
	"github.com/goji/httpauth"
	"github.com/orange-cloudfoundry/gobis/proxy/ctx"
)

const (
	LDAP_BIND_DN_KEY = "LDAP_BIND_DN"
	LDAP_BIND_PASSWORD_KEY = "LDAP_BIND_PASSWORD"
	LDAP_BIND_ADDRESS = "LDAP_BIND_ADDRESS"
)

type LdapConfig struct {
	Ldap *LdapOptions `mapstructure:"ldap" json:"ldap" yaml:"ldap"`
}
type LdapOptions struct {
	// enable ldap basic auth middleware
	Enable             bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// Search user bind dn (Can be set by env var `LDAP_BIND_DN`)
	BindDn             string `mapstructure:"bind_dn" json:"bind_dn" yaml:"bind_dn"`
	// Search user bind password (Can be set by env var `LDAP_BIND_PASSWORD`)
	BindPassword       string `mapstructure:"bind_password" json:"bind_password" yaml:"bind_password"`
	// Ldap server address in the form of host:port (Can be set by env var `LDAP_BIND_ADDRESS`)
	Address            string `mapstructure:"address" json:"address" yaml:"address"`
	// Set to true if ldap server supports TLS
	UseSsl             bool `mapstructure:"use_ssl" json:"use_ssl" yaml:"use_ssl"`
	// Set to true to skip certificate check
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	// base dns to search through (Default: `dc=com`)
	SearchBaseDns      string `mapstructure:"search_base_dns" json:"search_base_dns" yaml:"search_base_dns"`
	// User search filter, for example "(cn=%s)" or "(sAMAccountName=%s)" or "(uid=%s)" (default: `(objectClass=organizationalPerson)&(uid=%s)`)
	SearchFilter       string `mapstructure:"search_filter" json:"search_filter" yaml:"search_filter"`
	// Group search filter, to retrieve the groups of which the user is a member
	// Groups will be passed in request context as a list of strings, how to retrieve: ctx.Groups(*http.Request)
	// if GroupSearchFilter or GroupSearchBaseDns or MemberOf are empty it will not search for groups
	GroupSearchFilter  string `mapstructure:"group_search_filter" json:"group_search_filter" yaml:"group_search_filter"`
	// base DNs to search through for groups
	GroupSearchBaseDns string `mapstructure:"group_search_base_dns" json:"group_search_base_dns" yaml:"group_search_base_dns"`
	// Search group name by this value (default: `memberOf`)
	MemberOf           string `mapstructure:"member_of" json:"member_of" yaml:"member_of"`
}
type LdapAuth struct {
	LdapOptions
}

func NewLdapAuth(opt LdapOptions) *LdapAuth {
	return &LdapAuth{opt}
}
func (l LdapAuth) CreateConn() (conn *ldap.Conn, err error) {
	if l.UseSsl {
		conn, err = ldap.DialTLS("tcp", l.Address, &tls.Config{InsecureSkipVerify: l.InsecureSkipVerify})
	} else {
		conn, err = ldap.Dial("tcp", l.Address, )
	}
	if err != nil {
		return
	}
	err = conn.Bind(l.BindDn, l.BindPassword)
	if err != nil {
		return
	}
	return
}
func (l LdapAuth) LdapAuth(user, password string, req *http.Request) bool {
	ctx.DirtHeader(req, "Authorization")
	conn, err := l.CreateConn()
	if err != nil {
		log.Errorf("orange-cloudfoundry/gobis/middlewares: invalid ldap for '%s': %s", l.Address, err.Error())
		return false
	}
	defer conn.Close()
	searchRequest := ldap.NewSearchRequest(
		l.SearchBaseDns,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&" + l.SearchFilter + ")", user),
		[]string{"dn"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Errorf("orange-cloudfoundry/gobis/middlewares: invalid ldap search for '%s': %s", l.Address, err.Error())
		return false
	}

	if len(sr.Entries) != 1 {
		return false
	}

	userdn := sr.Entries[0].DN

	// Bind as the user to verify their password
	err = conn.Bind(userdn, password)
	if err != nil {
		return false
	}
	err = l.LoadLdapGroup(user, conn, req)
	if err != nil {
		log.Errorf("orange-cloudfoundry/gobis/middlewares: invalid ldap group search for '%s': %s", l.Address, err.Error())
		return false
	}
	ctx.SetUsername(req, user)
	return true
}
func (l LdapAuth) LoadLdapGroup(user string, conn *ldap.Conn, req *http.Request) error {
	if l.GroupSearchBaseDns == "" || l.GroupSearchFilter == "" {
		return nil
	}
	searchRequest := ldap.NewSearchRequest(
		l.GroupSearchBaseDns,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&" + l.GroupSearchFilter + ")", user),
		[]string{l.MemberOf},
		nil,
	)
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return err
	}
	groups := make([]string, 0)

	for _, entry := range sr.Entries {
		groups = append(groups, entry.GetAttributeValue(l.MemberOf))
	}
	ctx.AddGroups(req, groups...)
	return nil
}

func Ldap(proxyRoute models.ProxyRoute, handler http.Handler) (http.Handler, error) {
	var config LdapConfig
	err := mapstructure.Decode(proxyRoute.ExtraParams, &config)
	if err != nil {
		return handler, err
	}
	options := config.Ldap
	if options == nil || !options.Enable {
		return handler, nil
	}
	if options.BindDn == "" {
		options.BindDn = os.Getenv(LDAP_BIND_DN_KEY)
	}
	if options.BindPassword == "" {
		options.BindPassword = os.Getenv(LDAP_BIND_PASSWORD_KEY)
	}
	if options.Address == "" {
		options.Address = os.Getenv(LDAP_BIND_ADDRESS)
	}
	if options.BindDn == "" || options.BindPassword == "" {
		return handler, fmt.Errorf("bind dn and bind password can't be empty")
	}
	if options.Address == "" {
		return handler, fmt.Errorf("address can't be empty")
	}
	if options.SearchBaseDns == "" {
		options.SearchBaseDns = "dc=com"
	}
	if options.SearchFilter == "" {
		options.SearchFilter = "(objectClass=organizationalPerson)&(uid=%s)"
	}
	if options.MemberOf == "" {
		options.MemberOf = "memberOf"
	}
	ldapAuth := NewLdapAuth(*options)
	return httpauth.BasicAuth(httpauth.AuthOptions{
		AuthFunc: ldapAuth.LdapAuth,
	})(handler), nil
}
