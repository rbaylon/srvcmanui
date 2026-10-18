package userrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
	groupmodel "github.com/rbaylon/srvcman/modules/groups/model"
	usermodel "github.com/rbaylon/srvcman/modules/users/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/loginguard"
	"github.com/rbaylon/srvcmanui/utils"
)

var (
	key   = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
	store = sessions.NewCookieStore(key)
)

// requestIsSecure reports whether r reached this app over HTTPS - directly
// (r.TLS set, if this process ever terminates TLS itself) or via a reverse
// proxy that terminated it and says so through X-Forwarded-Proto (what
// nginx-proxy, this app's actual deployment, sets). Only used to decide
// the session cookie's Secure flag: gorilla/sessions' CookieStore defaults
// to Secure=true, which is correct behind the proxy (the browser sees
// https:// and stores the cookie) but breaks direct http://<ip> access -
// the browser silently discards a Secure cookie offered over plain HTTP,
// so login "succeeds" server-side yet the browser never has anything to
// send back, and the next request looks signed-out again: an infinite
// redirect loop back to the login page. Hardcoding Secure=false instead
// would fix that but weaken the normal, proxied HTTPS case, so this
// follows the request instead of a fixed setting either way.
func requestIsSecure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// LoginData is what the sign-in page shows: where to go afterwards, what
// was typed, why the last attempt failed, and a fresh math challenge.
type LoginData struct {
	Prevurl  string
	Username string
	Error    string
	Question string
	Token    string
}

// showLogin draws the sign-in page with a new challenge - every failed
// submission has spent its old one. retryAfter, when set, is also sent as
// the Retry-After header.
func showLogin(w http.ResponseWriter, status int, prevurl, username, msg string, retryAfter time.Duration) {
	ch, err := loginguard.NewChallenge()
	if err != nil {
		log.Println("login: could not build challenge:", err)
		utils.RenderErrorPage(w, http.StatusInternalServerError, "Could not prepare the sign-in form.")
		return
	}
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	files := []string{
		pwd + "/layouts/base.html",
		pwd + "/modules/users/templates/sign-in.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	// The page embeds a one-time token; a cached copy would only ever
	// fail the challenge.
	w.Header().Set("Cache-Control", "no-store")
	if retryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprint(int(retryAfter.Seconds())+1))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	ld := LoginData{Prevurl: prevurl, Username: username, Error: msg, Question: ch.Question, Token: ch.Token}
	if err := tmpl.ExecuteTemplate(w, "base", &ld); err != nil {
		log.Println("template error:", err)
	}
}

func waitText(d time.Duration) string {
	if d < 90*time.Second {
		return fmt.Sprintf("%d seconds", int(d.Seconds())+1)
	}
	return fmt.Sprintf("%d minutes", int(d.Minutes())+1)
}

func login() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := strings.TrimSpace(r.PostFormValue("username"))
		password := r.PostFormValue("password")
		prevurl := r.PostFormValue("prevurl")
		ip := loginguard.ClientIP(r)

		if ok, wait := loginguard.Begin(ip, username); !ok {
			log.Println("login: rate limited", ip, username)
			showLogin(w, http.StatusTooManyRequests, prevurl, username,
				"Too many sign-in attempts. Try again in "+waitText(wait)+".", wait)
			return
		}
		if username == "" || password == "" {
			showLogin(w, http.StatusBadRequest, prevurl, username, "Username and password are required.", 0)
			return
		}
		// The challenge is checked before the password so a bot that cannot
		// answer it never learns whether a password was right - and never
		// costs the srvcman backend a login request.
		if !loginguard.VerifyChallenge(r.PostFormValue("captcha_token"), r.PostFormValue("captcha_answer")) {
			loginguard.Fail(ip, "")
			showLogin(w, http.StatusUnauthorized, prevurl, username,
				"The math answer was wrong or expired. Please answer the new question.", 0)
			return
		}
		token, err := client.GetToken(username, password)
		if err == nil {
			loginguard.Succeed(username)
			session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
			session.Values["authenticated"] = true
			session.Values["whoami"] = username
			session.Values["jwt"] = *token
			session.Options.Secure = requestIsSecure(r)
			session.Options.SameSite = http.SameSiteLaxMode
			session.Save(r, w)
			if prevurl == "" {
				prevurl = "/ui/dashboard"
			}
			http.Redirect(w, r, prevurl, http.StatusSeeOther)
			return
		}
		loginguard.Fail(ip, username)
		showLogin(w, http.StatusUnauthorized, prevurl, username, "Invalid username or password!", 0)
	})
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	session.Options.Secure = requestIsSecure(r)
	session.Options.SameSite = http.SameSiteLaxMode
	session.Save(r, w)
	http.Redirect(w, r, "/ui/users/login", http.StatusSeeOther)
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records := []usermodel.User{}
		err, status := client.Rester("users", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
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
				utils.RenderErrorPage(w, 501, "error converting ID to string: "+err.Error())
				return
			}
			s.ID = uint(rid)
		} else {
			s.ID = 0
		}

		if err := r.ParseForm(); err != nil {
			fmt.Println("Error parsing form")
			utils.RenderErrorPage(w, 501, "Error parsing form: "+err.Error())
			return
		}
		for _, v := range r.Form["Groups"] {
			g := groupmodel.Group{}
			err, _ := client.Rester("groups/getbyname/"+v, "GET", &g, r)
			if err != nil {
				log.Println("error getting group: ", err)
			} else {
				groups = append(groups, g)
			}
		}
		s.Groups = groups
		if err := s.Bind(r); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("users/"+strconv.FormatUint(uint64(id), 10), "PUT", &s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("users/create", "POST", &s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("users/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/users", http.StatusSeeOther)
	})

	type userdata struct {
		User      usermodel.User
		Groups    []groupmodel.Group
		IsProfile bool
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record usermodel.User
		groups := []groupmodel.Group{}
		err, status := client.Rester("groups", "GET", &groups, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		err, status = client.Rester("users/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		formdata := userdata{User: record, Groups: groups}
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
	Router.HandleFunc("GET /profile", func(w http.ResponseWriter, r *http.Request) {
		var record usermodel.User
		var (
			key   = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
			store = sessions.NewCookieStore(key)
		)
		session, _ := store.Get(r, database.GetEnvVariable("APP_NAME"))
		whoami, _ := session.Values["whoami"].(string)
		if admin := database.GetEnvVariable("APP_ADMIN"); admin != "" && whoami == admin {
			// The built-in administrator is configured in .env, not stored as a
			// user, so there is no record to show or edit.
			utils.RenderErrorPage(w, http.StatusNotFound,
				"The built-in administrator account has no profile. Its name and password are set in the server's configuration.")
			return
		}
		err, status := client.Rester("users/username/"+whoami, "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		formdata := userdata{User: record, Groups: record.Groups, IsProfile: true}
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
		showLogin(w, http.StatusOK, prevurl, "", "", 0)
	})

	Router.HandleFunc("POST /login", login())
	Router.HandleFunc("GET /logout", logout)

	return Router
}
