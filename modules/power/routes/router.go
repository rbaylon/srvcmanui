// Package powerrouter - srvcmanui UI for restarting or shutting down the
// gateway.
//
// Restarting stops every service on the box, and a shutdown stays down until
// someone with physical (or hypervisor) access starts it, so nothing here can
// be triggered by a plain link or by a request from another site:
//   - both actions are POSTs, never GETs - a link, a prefetch or a crawler
//     cannot send one;
//   - the page carries a token tied to the signed-in user and good for a few
//     minutes (see makeToken), which the POST must return - a page on another
//     site cannot read it, so cannot forge the request;
//   - the page makes the operator confirm in a dialog first, and for a
//     shutdown type the word;
//   - srvcman asks for the confirmation again in the API request itself.
package powerrouter

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/rbaylon/srvcman/database"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// tokenTTL is how long a page's token stays valid.
const tokenTTL = 10 * time.Minute

// The session key and store are read from the environment when first needed,
// not at start-up: reading it aborts the process if .env is missing, which
// would take any test of this package down with it.
var (
	sessionOnce  sync.Once
	sessionKeyV  []byte
	sessionStore *sessions.CookieStore
)

func sessionKey() []byte {
	sessionOnce.Do(func() {
		sessionKeyV = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
		sessionStore = sessions.NewCookieStore(sessionKeyV)
	})
	return sessionKeyV
}

func sign(key []byte, user, ts string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte("power-csrf|" + user + "|" + ts))
	return hex.EncodeToString(m.Sum(nil))
}

// makeToken returns "<unix time>.<HMAC of user and time>". Nothing is stored:
// the signature, made with the session key, is what proves the page was
// served to this user by this server a moment ago. It is not written to the
// session, so this page does not touch the cookie the login set up.
func makeToken(key []byte, user string, now time.Time) string {
	ts := strconv.FormatInt(now.Unix(), 10)
	return ts + "." + sign(key, user, ts)
}

// validToken reports whether tok was made for user, by this server's key,
// within tokenTTL of now (and not in the future).
func validToken(key []byte, user, tok string, now time.Time) bool {
	if user == "" {
		return false
	}
	ts, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return false
	}
	issued, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	age := now.Sub(time.Unix(issued, 0))
	if age < 0 || age > tokenTTL {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(sig), []byte(sign(key, user, ts))) == 1
}

func currentUser(r *http.Request) string {
	sessionKey()
	session, _ := sessionStore.Get(r, database.GetEnvVariable("APP_NAME"))
	user, _ := session.Values["whoami"].(string)
	return user
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Cache-Control", "no-store")
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/power/templates/" + name,
	}
	tmpl := template.Must(template.ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

type confirmation struct {
	Confirm string `json:"confirm"`
}

type result struct {
	Action string // "restart" or "shutdown"
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		render(w, "power.html", struct{ Token string }{makeToken(sessionKey(), currentUser(r), time.Now())})
	})

	act := func(action string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := currentUser(r)
			if !validToken(sessionKey(), user, r.PostFormValue("csrf"), time.Now()) {
				utils.RenderErrorPage(w, http.StatusForbidden,
					"This page has expired or was not opened from the Power page. Open System Power again and repeat the action.")
				return
			}
			body := confirmation{Confirm: action}
			if err, status := client.Rester("power/"+action, "PUT", &body, r); err != nil {
				log.Println("power:", action, "failed:", err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			log.Printf("power: %s requested by %s from %s", action, user, r.RemoteAddr)
			render(w, "powerdone.html", result{Action: action})
		}
	}
	Router.HandleFunc("POST /restart", act("restart"))
	Router.HandleFunc("POST /shutdown", act("shutdown"))

	return Router
}
