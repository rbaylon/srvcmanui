package userrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
	"github.com/rbaylon/srvcmanui/client"
)

var (
	key   = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
	store = sessions.NewCookieStore(key)
)

func login() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")
		if username == "" || password == "" {
			http.Error(w, "User or password empty", http.StatusForbidden)
			return
		}
		token, err := client.GetToken(username, password)
		if err == nil {
			session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
			session.Values["authenticated"] = true
			session.Values["whoami"] = username
			session.Values["jwt"] = *token
			session.Save(r, w)
			prevurl := r.PostFormValue("prevurl")
			if prevurl == "" {
				prevurl = "/ui/subs"
			}
			http.Redirect(w, r, prevurl, http.StatusSeeOther)
			return
		}
		http.Error(w, "User or password invalid", http.StatusForbidden)
	})
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/ui/users/login", http.StatusSeeOther)
}

type LoginData struct {
	Prevurl string
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		prevurl := r.URL.Query().Get("prevurl")
		if prevurl != "" {
			rawprevurl, err := url.PathUnescape(prevurl)
			if err != nil {
				log.Println(err)
				prevurl = ""
			}
			prevurl = rawprevurl
		}
		pwd, err := os.Getwd()
		ld := LoginData{Prevurl: prevurl}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		files := []string{
			pwd + "/layouts/base.html",
			pwd + "/modules/users/templates/sign-in.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "base", &ld)
	})

	Router.HandleFunc("POST /login", login())
	Router.HandleFunc("GET /logout", logout)

	return Router
}
