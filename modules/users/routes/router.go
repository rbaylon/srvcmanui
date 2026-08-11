package userrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
	groupmodel "github.com/rbaylon/srvcman/modules/groups/model"
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
		groups := []groupmodel.Group{}
		s.Firstname = r.FormValue("Firstname")
		s.Lastname = r.FormValue("Lastname")
		s.Username = r.FormValue("Username")
		s.Password = r.FormValue("Password")
		fid := r.FormValue("ID")
		if fid != "" {
			rid, err := strconv.Atoi(fid)
			if err != nil {
				log.Println("error converting ID to string: ", rid)
				http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
				return
			}
			s.ID = uint(rid)
		} else {
			s.ID = 0
		}

		if err := r.ParseForm(); err != nil {
			log.Println("Error parsing form")
			http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
			return
		}
		for _, v := range r.Form["Groups"] {
			g := groupmodel.Group{}
			err := client.Rester("groups/getbyname/"+v, "GET", &g, r)
			if err != nil {
				log.Println("error getting group: ", err)
			} else {
				groups = append(groups, g)
			}
		}
		s.Groups = groups
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

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := client.Rester("users/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
	})

	type userdata struct {
		User   usermodel.User
		Groups []groupmodel.Group
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record usermodel.User
		groups := []groupmodel.Group{}
		err := client.Rester("groups", "GET", &groups, r)
		if err != nil {
			log.Println(err)
		}
		err = client.Rester("users/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		formdata := userdata{record, groups}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/users/templates/user.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
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
