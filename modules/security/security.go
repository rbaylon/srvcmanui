package usersecurity

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
	"github.com/rbaylon/srvcman/modules/security"
)

var (
	key   = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
	store = sessions.NewCookieStore(key)
)

func Login(db usermodel.Crud) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			user, err := db.GetByUsername(username)
			if err != nil {
				http.Error(w, "User invalid", http.StatusForbidden)
				return
			}

			if security.CheckPasswordHash(password, user.Password) {
				session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
				session.Values["authenticated"] = true
				session.Save(r, w)
				return
			}
			http.Error(w, "Password invalid", http.StatusForbidden)
			return
		}
		http.Error(w, "User or password invalid", http.StatusForbidden)
	})
}
