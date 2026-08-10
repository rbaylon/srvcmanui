package userrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
	usermodel "github.com/rbaylon/srvcman/modules/users/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
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
			utils.RenderErrorPage(w, http.StatusUnauthorized, "username and password required!")
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
		utils.RenderErrorPage(w, http.StatusUnauthorized, "Invalid username or password!")
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

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records := []usermodel.User{}
		err := client.Rester("users", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/users/templates/users.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/users/templates/usernew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "htmx", nil)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &usermodel.User{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err := client.Rester("users/"+strconv.FormatUint(uint64(id), 10), "PUT", &s, r)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
			return
		}
		err := client.Rester("users/create", "POST", &s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
	})

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
