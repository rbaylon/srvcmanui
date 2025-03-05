package accessvalidator

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
)

var (
	key   = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
	store = sessions.NewCookieStore(key)
)

func Check(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))

		// Check if user is authenticated
		if !strings.Contains(r.URL.Path, "/ui/users/login") && !strings.Contains(r.URL.Path, "/ui/static") {
			if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
				http.Redirect(w, r, "/ui/users/login?prevurl="+url.PathEscape(r.URL.Path), http.StatusSeeOther)
				return
			}
			me := session.Values["whoami"]
			if me != "admin" && !strings.Contains(r.URL.Path, "/ui/subs") {
				http.Redirect(w, r, "/ui/users/login?prevurl="+url.PathEscape(r.URL.Path), http.StatusSeeOther)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
