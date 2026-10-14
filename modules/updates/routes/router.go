// Package updatesrouter - srvcmanui UI for the updates module.
//
// One page: software updates from the update server (specz.net). It shows
// whether an update is available, lets the operator start it, shows its
// progress and result, and holds the two settings it needs: the update
// server address and the API token the operator created on the website.
//
// Installing an update first brings the operating system up to date, then
// installs the update. That is done by srvcman in one background job and is
// deliberately not shown as a feature: the page only says "preparing the
// system", there is no OS patch check or button, and nothing here asks the
// operator about it.
//
// Installing changes the running system, so it is a POST that needs the page's
// own token, like the Power page: a link, a prefetch or another site cannot
// start it. Saving the API token is protected the same way.
//
// The structs below are local rather than imported from srvcman's updates
// model: they are just the JSON fields the API exchanges, and keeping them
// here means this UI builds without waiting on a srvcman release.
package updatesrouter

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

// rester is the call to the srvcman API; tests replace it.
var rester = client.Rester

// tokenTTL is how long a page's token stays valid.
const tokenTTL = 15 * time.Minute

// settings is the update configuration. The API token is never sent back by
// srvcman: TokenSet only says whether one is saved.
type settings struct {
	UpdateServer string `json:"update_server"`
	Product      string `json:"product"`
	TokenSet     bool   `json:"token_set"`
}

// mirror is the OpenBSD mirror the gateway downloads system updates from.
type mirror struct {
	Mirror  string `json:"mirror"`
	Default string `json:"default"`
}

// mirrorChoices are well-known OpenBSD mirrors offered as suggestions; any
// other address can be typed. The first is OpenBSD's own default.
var mirrorChoices = []struct{ URL, Where string }{
	{"https://cdn.openbsd.org/pub/OpenBSD", "OpenBSD CDN (default, picks a nearby server)"},
	{"https://ftp.jaist.ac.jp/pub/OpenBSD", "Japan - JAIST"},
	{"https://mirror.freedif.org/OpenBSD", "Singapore / Indonesia - FreeDif"},
	{"https://mirror.aarnet.edu.au/pub/OpenBSD", "Australia - AARNet"},
	{"https://ftp.nluug.nl/pub/OpenBSD", "Netherlands - NLUUG"},
	{"https://ftp.hostserver.de/pub/OpenBSD", "Germany - hostserver.de"},
	{"https://mirrors.mit.edu/pub/OpenBSD", "United States - MIT"},
}

// settingsIn is what is sent to change them. ApiToken is a pointer so that
// leaving it out keeps the saved token.
type settingsIn struct {
	UpdateServer string  `json:"update_server"`
	ApiToken     *string `json:"api_token,omitempty"`
	ClearToken   bool    `json:"clear_token,omitempty"`
}

type file struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type update struct {
	Version       string   `json:"version"`
	Notes         string   `json:"notes"`
	Channels      []string `json:"channels"`
	PublishedAt   string   `json:"published_at"`
	AvailableFrom string   `json:"available_from"`
	Files         []file   `json:"files"`
}

type scheduled struct {
	Version       string `json:"version"`
	AvailableFrom string `json:"available_from"`
}

// software mirrors srvcman's answer to GET updates/software.
type software struct {
	Configured      bool       `json:"configured"`
	Product         string     `json:"product"`
	Release         string     `json:"release"`
	Installed       string     `json:"installed"`
	Tier            string     `json:"tier"`
	Latest          *update    `json:"latest"`
	Scheduled       *scheduled `json:"scheduled"`
	UpdateAvailable bool       `json:"update_available"`
	Error           string     `json:"error"`
}

// job mirrors srvcman's update job: State is idle, running, success or
// failed; Step is downloading, preparing or installing while it runs.
type job struct {
	State         string     `json:"state"`
	Step          string     `json:"step"`
	Version       string     `json:"version"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	Message       string     `json:"message"`
	Log           string     `json:"log"`
	RebootPending bool       `json:"reboot_pending"`
}

// step is one line of the progress list.
type step struct {
	Label  string
	Status string // done, current or todo
}

type viewdata struct {
	Settings settings
	Software software
	Job      job
	Token    string // the page's CSRF token
	Mirror   mirror
	// HasMirror is false when srvcman does not have the mirror call (older).
	HasMirror bool
	Choices   []struct{ URL, Where string }
	Saved     bool
	Notice    string // a refusal to show above the page (e.g. nothing to install)
	Steps     []step
	// LegacyServer is true when srvcman does not know the software update calls
	// yet (an older srvcman): the page then explains instead of failing.
	LegacyServer bool
	// ShowResult is whether the last finished job is worth showing: a success or
	// failure from the last day.
	ShowResult bool
}

// stepLabels are the operator-facing steps, in order. The operating system
// patching is part of "Preparing the system"; it is never named.
var stepLabels = []struct{ key, label string }{
	{"downloading", "Downloading the update"},
	{"preparing", "Preparing the system"},
	{"installing", "Installing the update"},
}

// stepsFor lists the steps with the one in progress marked.
func stepsFor(current string) []step {
	var out []step
	known := false
	for _, s := range stepLabels {
		known = known || s.key == current
	}
	seen := !known // an unknown step shows everything as still to do
	for _, s := range stepLabels {
		st := "todo"
		switch {
		case s.key == current:
			st, seen = "current", true
		case !seen:
			st = "done"
		}
		out = append(out, step{Label: s.label, Status: st})
	}
	return out
}

func viewOf(s settings, sw software, j job, token string, now time.Time) viewdata {
	v := viewdata{Settings: s, Software: sw, Job: j, Token: token}
	if j.State == "running" {
		v.Steps = stepsFor(j.Step)
	}
	if (j.State == "success" || j.State == "failed") && (j.FinishedAt == nil || now.Sub(*j.FinishedAt) < 24*time.Hour) {
		v.ShowResult = true
	}
	return v
}

// ---- the page's token (same scheme as the Power page) ----

var (
	sessionOnce  sync.Once
	sessionKeyV  []byte
	sessionStore *sessions.CookieStore
)

// The session key and store are read from the environment when first needed,
// not at start-up, so that tests of this package do not need a .env file.
func sessionKey() []byte {
	sessionOnce.Do(func() {
		sessionKeyV = []byte(database.GetEnvVariable("APP_SESSION_KEY"))
		sessionStore = sessions.NewCookieStore(sessionKeyV)
	})
	return sessionKeyV
}

func sign(key []byte, user, ts string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte("updates-csrf|" + user + "|" + ts))
	return hex.EncodeToString(m.Sum(nil))
}

// makeToken returns "<unix time>.<HMAC of user and time>": nothing is stored,
// the signature made with the session key proves this server showed this page
// to this user a moment ago.
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

// ---- rendering ----

func page(w http.ResponseWriter, data viewdata) {
	w.Header().Set("Cache-Control", "no-store")
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/updates/templates/updates.html",
	}
	tmpl := template.Must(template.New("updates").Funcs(template.FuncMap{
		"size": humanSize,
		"when": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Local().Format("2006-01-02 15:04")
		},
	}).ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(n)/float64(div), 'f', 1, 64) + " " + string("KMGTPE"[exp]) + "B"
}

// isOldSrvcman reports whether status says the API does not have the call: a
// srvcman that predates software updates.
func isOldSrvcman(status int) bool { return status == http.StatusNotFound }

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var s settings
		if err, status := rester("updates", "GET", &s, r); err != nil {
			log.Println("updates:", err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		token := makeToken(sessionKey(), currentUser(r), time.Now())
		var sw software
		var j job
		legacy := false
		if err, status := rester("updates/software/install", "GET", &j, r); err != nil {
			if !isOldSrvcman(status) {
				log.Println("updates:", err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			legacy = true
		}
		// The check asks the update server, so it is skipped while an install runs:
		// the page only needs to show progress then, and refreshes every few seconds.
		if !legacy && j.State != "running" {
			if err, status := rester("updates/software", "GET", &sw, r); err != nil {
				log.Println("updates:", err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
		}
		v := viewOf(s, sw, j, token, time.Now())
		if err, _ := rester("updates/mirror", "GET", &v.Mirror, r); err == nil {
			v.HasMirror = true
			v.Choices = mirrorChoices
		}
		v.LegacyServer = legacy
		v.Saved = r.URL.Query().Get("saved") == "1"
		page(w, v)
	})

	Router.HandleFunc("POST /save", func(w http.ResponseWriter, r *http.Request) {
		if !validToken(sessionKey(), currentUser(r), r.PostFormValue("csrf"), time.Now()) {
			utils.RenderErrorPage(w, http.StatusForbidden,
				"This page has expired or was not opened from the Software Update page. Open it again and repeat the change.")
			return
		}
		in := settingsIn{UpdateServer: strings.TrimSpace(r.FormValue("UpdateServer"))}
		if tok := strings.TrimSpace(r.FormValue("ApiToken")); tok != "" {
			in.ApiToken = &tok
		}
		in.ClearToken = r.FormValue("ClearToken") == "1"
		var out settings
		if err, status := rester("updates", "PUT", &in, r); err != nil {
			log.Println("updates: save failed:", err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		_ = out
		http.Redirect(w, r, "/ui/updates?saved=1", http.StatusSeeOther)
	})

	// Changes the OpenBSD mirror. srvcman tests the mirror before it replaces the
	// old one, so a wrong address comes back as an error and nothing changes.
	Router.HandleFunc("POST /mirror", func(w http.ResponseWriter, r *http.Request) {
		if !validToken(sessionKey(), currentUser(r), r.PostFormValue("csrf"), time.Now()) {
			utils.RenderErrorPage(w, http.StatusForbidden,
				"This page has expired or was not opened from the Software Update page. Open it again and repeat the change.")
			return
		}
		in := mirror{Mirror: strings.TrimSpace(r.FormValue("Mirror"))}
		var out mirror
		if err, status := rester("updates/mirror", "PUT", &in, r); err != nil {
			log.Println("updates: mirror not changed:", err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		_ = out
		http.Redirect(w, r, "/ui/updates?saved=1", http.StatusSeeOther)
	})

	// Starts the installation in the background and returns to the page, which
	// shows its progress. Pressing it again while one runs is harmless: srvcman
	// leaves the running job alone.
	Router.HandleFunc("POST /install", func(w http.ResponseWriter, r *http.Request) {
		if !validToken(sessionKey(), currentUser(r), r.PostFormValue("csrf"), time.Now()) {
			utils.RenderErrorPage(w, http.StatusForbidden,
				"This page has expired or was not opened from the Software Update page. Open it again and press Update again.")
			return
		}
		var j job
		if err, status := rester("updates/software/install", "PUT", &j, r); err != nil {
			log.Println("updates: install refused:", err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/updates", http.StatusSeeOther)
	})

	return Router
}
