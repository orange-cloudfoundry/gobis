package gobis

import "net/http"

const (
	GroupContextKey MiddlewareContextKey = iota
	UsernameContextKey
)

type MiddlewareContextKey int

// Set the username to a request context
func SetUsername(req *http.Request, username string) {
	userPtr := usernamePtr(req)
	if userPtr == nil {
		user := username
		AddContextValue(req, UsernameContextKey, &user)
		return
	}
	*userPtr = username
}

// Retrieve username from a request context
func Username(req *http.Request) string {
	userPtr := usernamePtr(req)
	if userPtr == nil {
		return ""
	}
	return *userPtr
}
func usernamePtr(req *http.Request) *string {
	var username *string
	InjectContextValue(req, UsernameContextKey, &username)
	return username
}

// add groups to a request context
// this call AddGroups, it's simply for UX
func SetGroups(req *http.Request, groups ...string) {
	AddGroups(req, groups...)
}

// add groups to a request context
func AddGroups(req *http.Request, groups ...string) {
	if len(groups) == 0 {
		return
	}
	groupsPtr := groupsPtr(req)
	if groupsPtr == nil {
		m := sliceToMap(groups)
		AddContextValue(req, GroupContextKey, &m)
		return
	}
	origGroups := *groupsPtr
	for _, group := range groups {
		origGroups[group] = true
	}
	*groupsPtr = origGroups
}

// retrieve groups from request context
func Groups(req *http.Request) []string {
	groupsPtr := groupsPtr(req)
	if groupsPtr == nil {
		return make([]string, 0)
	}
	return mapToSlice(*groupsPtr)
}
func groupsPtr(req *http.Request) *map[string]bool {
	var groups *map[string]bool
	InjectContextValue(req, GroupContextKey, &groups)
	return groups
}
func mapToSlice(m map[string]bool) []string {
	s := make([]string, 0)
	for key, _ := range m {
		s = append(s, key)
	}
	return s
}
func sliceToMap(s []string) map[string]bool {
	m := make(map[string]bool)
	for _, v := range s {
		m[v] = true
	}
	return m
}
