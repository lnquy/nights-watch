package router

import (
	"time"

	"github.com/alexedwards/scs"
)

var sm *scs.Manager

func init() {
	sm = scs.NewCookieManager("u46IpCV9y5Vlur8YvODJEhgOY8m9JVE4")
	sm.Lifetime(7 * 24 * time.Hour)
	sm.Name("nightswatch")
	sm.Persist(true)
	sm.Secure(false)
}

func (rt *Router) GetSessionManager() *scs.Manager {
	return sm
}
