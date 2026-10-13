package updatesrouter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/sessions"
)

const testSessionKey = "test-session-key-0123456789"
const testAppName = "specz-test"

var setupOnce sync.Once

// setup gives the tests what the handlers read from the working directory: a .env (loaded
// once, as in production), the layout and the module's template. It chdirs into a temp dir.
func setup(t *testing.T) {
	t.Helper()
	setupOnce.Do(func() {
		dir, err := os.MkdirTemp("", "updatesui")
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_SESSION_KEY="+testSessionKey+"\nAPP_NAME="+testAppName+"\nAPP_PORT=1\nAPI_URL=http://127.0.0.1:1/\n"), 0o600)
		for _, p := range [][2]string{{"../../../layouts/newbase.html", "layouts/newbase.html"}, {"../templates/updates.html", "modules/updates/templates/updates.html"}} {
			b, err := os.ReadFile(p[0])
			if err != nil {
				t.Fatal(err)
			}
			os.MkdirAll(filepath.Dir(filepath.Join(dir, p[1])), 0o755)
			os.WriteFile(filepath.Join(dir, p[1]), b, 0o644)
		}
		os.Chdir(dir)
	})
}

// stubAPI stands in for srvcman: it records the calls and answers from a table.
type stubAPI struct {
	calls    []string
	bodies   map[string]string
	settings settings
	sw       software
	job      job
	status   map[string]int // "METHOD uri" -> error status (the call fails with it)
	installJ job
}

func (s *stubAPI) install(t *testing.T) {
	old := rester
	rester = func(uri, method string, data any, r *http.Request) (error, int) {
		key := method + " " + uri
		s.calls = append(s.calls, key)
		if method == "PUT" || method == "POST" {
			b, _ := json.Marshal(data)
			if s.bodies == nil {
				s.bodies = map[string]string{}
			}
			s.bodies[key] = string(b)
		}
		if code, ok := s.status[key]; ok {
			return fmt.Errorf("API error: %d", code), code
		}
		var v any
		switch key {
		case "GET updates":
			v = s.settings
		case "GET updates/software":
			v = s.sw
		case "GET updates/software/install":
			v = s.job
		case "PUT updates/software/install":
			v = s.installJ
		case "PUT updates":
			v = s.settings
		default:
			return fmt.Errorf("unexpected call %s", key), 500
		}
		b, _ := json.Marshal(v)
		json.Unmarshal(b, data)
		return nil, 200
	}
	t.Cleanup(func() { rester = old })
}

func (s *stubAPI) called(key string) bool {
	for _, c := range s.calls {
		if c == key {
			return true
		}
	}
	return false
}

// request makes a request with a signed-in session (user "admin") cookie.
func request(t *testing.T, method, path string, form url.Values) *http.Request {
	t.Helper()
	var r *http.Request
	if form != nil {
		r = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	store := sessions.NewCookieStore([]byte(testSessionKey))
	sess, _ := store.Get(r, testAppName)
	sess.Values["whoami"] = "admin"
	rec := httptest.NewRecorder()
	if err := store.Save(r, rec, sess); err != nil {
		t.Fatal(err)
	}
	for _, c := range rec.Result().Cookies() {
		r.AddCookie(c)
	}
	return r
}

func serve(t *testing.T, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	// As main.go mounts it.
	mux.Handle("/ui/updates/", http.StripPrefix("/ui/updates", New()))
	mux.ServeHTTP(rec, r)
	return rec
}

func get(t *testing.T, s *stubAPI, path string) string {
	t.Helper()
	setup(t)
	s.install(t)
	rec := serve(t, request(t, "GET", path, nil))
	if rec.Code != 200 {
		t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func contains(t *testing.T, body string, subs ...string) {
	t.Helper()
	for _, s := range subs {
		if !strings.Contains(body, s) {
			t.Errorf("page lacks %q", s)
		}
	}
}

func lacks(t *testing.T, body string, subs ...string) {
	t.Helper()
	for _, s := range subs {
		if strings.Contains(body, s) {
			t.Errorf("page unexpectedly contains %q", s)
		}
	}
}

var osWords = regexp.MustCompile(`(?i)syspatch|operating system|os patch|os update|patches`)

func TestPageStates(t *testing.T) {
	finished := time.Now().Add(-time.Hour)
	latest := &update{Version: "1.0.1", Notes: "Fixes the voucher bug", Files: []file{{"u.tgz", 30_000_000}, {"n.txt", 5}}}
	cases := []struct {
		name  string
		stub  *stubAPI
		has   []string
		not   []string
		skips string // an API call that must not be made
	}{
		{"not configured", &stubAPI{sw: software{Configured: false}},
			[]string{"Updates are not set up yet", "API tokens", `name="ApiToken"`, `name="UpdateServer"`, `placeholder="spz_..."`},
			[]string{"Update now", "Could not check", "up to date"}, ""},
		{"token saved", &stubAPI{settings: settings{UpdateServer: "https://www.specz.net", TokenSet: true}, sw: software{Configured: true}},
			[]string{`value="https://www.specz.net"`, "A token is saved. Enter a new one to replace it.", `name="ClearToken"`, "Remove the saved token"},
			nil, ""},
		{"update available", &stubAPI{sw: software{Configured: true, UpdateAvailable: true, Installed: "1.0.0", Latest: latest}},
			[]string{"Update 1.0.1 is available", "Fixes the voucher bug", "Update now", `action="/ui/updates/install"`, `name="csrf"`, "2 files", "28.6 MB, 5 B", "You have 1.0.0"},
			[]string{"Could not check", "up to date"}, ""},
		{"up to date", &stubAPI{sw: software{Configured: true, Installed: "1.0.1", Latest: latest}},
			[]string{"The software is up to date", "Installed update: 1.0.1."}, []string{"Update now"}, ""},
		{"up to date with a scheduled update", &stubAPI{sw: software{Configured: true, Latest: nil, Scheduled: &scheduled{Version: "1.0.2", AvailableFrom: "2026-11-03T09:00:00Z"}}},
			[]string{"The software is up to date", "No update has been needed so far.", "next update, 1.0.2, is scheduled", "2026-11-03T09:00:00Z"}, nil, ""},
		{"check failed", &stubAPI{sw: software{Configured: true, Error: "The update server did not accept the API token"}},
			[]string{"Could not check for updates", "did not accept the API token", "Try again"}, []string{"Update now", "up to date"}, ""},
		{"running: download", &stubAPI{job: job{State: "running", Step: "downloading", Version: "1.0.1", StartedAt: &finished}},
			[]string{"Installing update 1.0.1", "Downloading the update", "Preparing the system", "Installing the update", "window.location.reload()", "This page refreshes by itself"},
			[]string{"Update now"}, "GET updates/software"},
		{"running: preparing", &stubAPI{job: job{State: "running", Step: "preparing", Version: "1.0.1"}},
			[]string{`data-feather="check-circle" class="text-success me-2"`, "fw-bold"}, nil, "GET updates/software"},
		{"failed", &stubAPI{sw: software{Configured: true, UpdateAvailable: true, Latest: latest}, job: job{State: "failed", Version: "1.0.1", Message: "The update's install script failed.", Log: "2026 FAILED (install)\ndisk full", FinishedAt: &finished}},
			[]string{"The update did not complete", "install script failed", "Technical details", "disk full", "press Update again to retry", "Update now"}, nil, ""},
		{"failed long ago is not shown", &stubAPI{sw: software{Configured: true}, job: job{State: "failed", Message: "old failure", FinishedAt: ptr(time.Now().Add(-48 * time.Hour))}},
			nil, []string{"old failure", "did not complete"}, ""},
		{"success with a restart pending", &stubAPI{sw: software{Configured: true, Installed: "1.0.1"}, job: job{State: "success", Version: "1.0.1", RebootPending: true, FinishedAt: &finished}},
			[]string{"Update 1.0.1 was installed", "Some changes take full effect after the next restart", `href="/ui/power/"`}, nil, ""},
		{"success without a restart", &stubAPI{sw: software{Configured: true, Installed: "1.0.1"}, job: job{State: "success", Version: "1.0.1", FinishedAt: &finished}},
			[]string{"Update 1.0.1 was installed"}, []string{"next restart", "You can restart from"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := c.stub
			body := get(t, s, "/ui/updates/")
			contains(t, body, c.has...)
			lacks(t, body, c.not...)
			// The operating system update is not a feature of this page: in no state is it named.
			if m := osWords.FindString(body); m != "" {
				t.Errorf("the page mentions the OS update: %q", m)
			}
			if c.skips != "" && s.called(c.skips) {
				t.Errorf("%s was called", c.skips)
			}
			if s.called("PUT updates/check") || s.called("GET updates/check") {
				t.Error("the page uses the OS patch check")
			}
			// The page's own token is always there, for the settings form at least.
			if !strings.Contains(body, `name="csrf" value="`) {
				t.Error("no token in the page")
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestOldSrvcmanGetsAnExplanation(t *testing.T) {
	s := &stubAPI{status: map[string]int{"GET updates/software/install": 404}, settings: settings{UpdateServer: "https://www.specz.net"}}
	body := get(t, s, "/ui/updates/")
	contains(t, body, "does not support software updates yet", `name="UpdateServer"`)
	lacks(t, body, "Update now")
	if s.called("GET updates/software") {
		t.Error("asked for a software check that the old service cannot answer")
	}
}

func TestAPIErrorsShowTheErrorPage(t *testing.T) {
	setup(t)
	// Error pages are rendered from <cwd>/error.html, like everywhere in this UI.
	b, _ := os.ReadFile("../../../utils/error.html")
	os.WriteFile("error.html", b, 0o644)
	s := &stubAPI{status: map[string]int{"GET updates": 500}}
	s.install(t)
	rec := serve(t, request(t, "GET", "/ui/updates/", nil))
	if rec.Code != 500 {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestRemovedOSCheckRoutes(t *testing.T) {
	setup(t)
	s := &stubAPI{}
	s.install(t)
	// Like any unknown GET under this module it shows the page (the module's "GET /"), but it
	// no longer starts anything: the OS patch check is gone.
	serve(t, request(t, "GET", "/ui/updates/check", nil))
	if s.called("PUT updates/check") || s.called("GET updates/check") {
		t.Fatal("/ui/updates/check still starts the OS patch check")
	}
}

func tokenFor(user string, at time.Time) string { return makeToken([]byte(testSessionKey), user, at) }

func TestSaveNeedsTheTokenAndSendsOnlyWhatWasEntered(t *testing.T) {
	setup(t)
	s := &stubAPI{}
	s.install(t)
	// No token, a token for another user, an old token: nothing is saved.
	for name, tok := range map[string]string{"none": "", "other user": tokenFor("someone", time.Now()), "expired": tokenFor("admin", time.Now().Add(-2*tokenTTL)), "garbage": "x.y"} {
		rec := serve(t, request(t, "POST", "/ui/updates/save", url.Values{"csrf": {tok}, "UpdateServer": {"https://evil.example"}, "ApiToken": {"spz_x"}}))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: %d, want 403", name, rec.Code)
		}
	}
	if s.called("PUT updates") {
		t.Fatal("settings were saved without a valid page token")
	}
	// A new server and token.
	rec := serve(t, request(t, "POST", "/ui/updates/save", url.Values{"csrf": {tokenFor("admin", time.Now())}, "UpdateServer": {" https://www.specz.net "}, "ApiToken": {" spz_abcdef123456 "}}))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/ui/updates?saved=1" {
		t.Fatalf("save: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	var in map[string]any
	json.Unmarshal([]byte(s.bodies["PUT updates"]), &in)
	if in["update_server"] != "https://www.specz.net" || in["api_token"] != "spz_abcdef123456" || in["clear_token"] != nil {
		t.Fatalf("sent %v", in)
	}
	// Server only: the saved token is left alone (the field is omitted, not sent empty).
	serve(t, request(t, "POST", "/ui/updates/save", url.Values{"csrf": {tokenFor("admin", time.Now())}, "UpdateServer": {"https://www.specz.net"}, "ApiToken": {""}}))
	if strings.Contains(s.bodies["PUT updates"], "api_token") || strings.Contains(s.bodies["PUT updates"], "clear_token") {
		t.Fatalf("sent %s: an empty token field must leave the saved token alone", s.bodies["PUT updates"])
	}
	// Remove the token.
	serve(t, request(t, "POST", "/ui/updates/save", url.Values{"csrf": {tokenFor("admin", time.Now())}, "UpdateServer": {"https://www.specz.net"}, "ClearToken": {"1"}}))
	if !strings.Contains(s.bodies["PUT updates"], `"clear_token":true`) {
		t.Fatalf("sent %s", s.bodies["PUT updates"])
	}
	// srvcman refuses (a bad address): its reason reaches the operator.
	s.status = map[string]int{"PUT updates": 400}
	b, _ := os.ReadFile("../../../utils/error.html")
	os.WriteFile("error.html", b, 0o644)
	if rec := serve(t, request(t, "POST", "/ui/updates/save", url.Values{"csrf": {tokenFor("admin", time.Now())}, "UpdateServer": {"nonsense"}})); rec.Code != 400 {
		t.Fatalf("refusal: %d", rec.Code)
	}
	// Never over GET.
	before := len(s.calls)
	serve(t, request(t, "GET", "/ui/updates/save?UpdateServer=x&ApiToken=spz_abcdef123456", nil))
	for _, c := range s.calls[before:] {
		if strings.HasPrefix(c, "PUT") {
			t.Fatalf("save over GET changed something: %s", c)
		}
	}
}

func TestInstallNeedsTheTokenAndIsAPost(t *testing.T) {
	setup(t)
	s := &stubAPI{installJ: job{State: "running", Step: "downloading"}}
	s.install(t)
	for name, tok := range map[string]string{"none": "", "other user": tokenFor("someone", time.Now()), "expired": tokenFor("admin", time.Now().Add(-2*tokenTTL)), "from the future": tokenFor("admin", time.Now().Add(time.Hour))} {
		rec := serve(t, request(t, "POST", "/ui/updates/install", url.Values{"csrf": {tok}}))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: %d, want 403", name, rec.Code)
		}
	}
	for _, m := range []string{"GET", "PUT", "DELETE"} {
		serve(t, request(t, m, "/ui/updates/install", nil))
	}
	if s.called("PUT updates/software/install") {
		t.Fatal("an install was started without a valid page token")
	}
	// The page's own token starts it, and the browser is sent back to the page to watch it.
	rec := serve(t, request(t, "POST", "/ui/updates/install", url.Values{"csrf": {tokenFor("admin", time.Now())}}))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/ui/updates" || !s.called("PUT updates/software/install") {
		t.Fatalf("install: %d %q called=%v", rec.Code, rec.Header().Get("Location"), s.called("PUT updates/software/install"))
	}
	// A refusal (nothing to install, no token saved) is shown with its reason.
	b, _ := os.ReadFile("../../../utils/error.html")
	os.WriteFile("error.html", b, 0o644)
	s.status = map[string]int{"PUT updates/software/install": 400}
	if rec := serve(t, request(t, "POST", "/ui/updates/install", url.Values{"csrf": {tokenFor("admin", time.Now())}})); rec.Code != 400 {
		t.Fatalf("refusal: %d", rec.Code)
	}
}

func TestPageTokenIsInTheForms(t *testing.T) {
	s := &stubAPI{sw: software{Configured: true, UpdateAvailable: true, Latest: &update{Version: "1.0.1", Files: []file{{"u.tgz", 10}}}}}
	body := get(t, s, "/ui/updates/")
	m := regexp.MustCompile(`name="csrf" value="([0-9]+\.[0-9a-f]{64})"`).FindAllStringSubmatch(body, -1)
	if len(m) != 2 { // the install form and the settings form
		t.Fatalf("%d tokens in the page, want 2", len(m))
	}
	for _, x := range m {
		if !validToken([]byte(testSessionKey), "admin", x[1], time.Now()) {
			t.Errorf("token %q is not valid for the signed-in user", x[1])
		}
	}
	// The page is not cached: the token and the progress change.
	setup(t)
	s.install(t)
	rec := serve(t, request(t, "GET", "/ui/updates/", nil))
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("cache control %q", rec.Header().Get("Cache-Control"))
	}
}

func TestStepsFor(t *testing.T) {
	got := func(cur string) string {
		var s []string
		for _, st := range stepsFor(cur) {
			s = append(s, st.Status)
		}
		return strings.Join(s, ",")
	}
	for cur, want := range map[string]string{"downloading": "current,todo,todo", "preparing": "done,current,todo", "installing": "done,done,current", "": "todo,todo,todo", "unknown": "todo,todo,todo"} {
		if g := got(cur); g != want {
			t.Errorf("stepsFor(%q) = %s, want %s", cur, g, want)
		}
	}
	// The labels never name the operating system.
	for _, st := range stepsFor("preparing") {
		if osWords.MatchString(st.Label) {
			t.Errorf("step label %q names the OS update", st.Label)
		}
	}
}

func TestTokenRejectsEverythingElse(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	key := []byte("k")
	tok := makeToken(key, "admin", now)
	if !validToken(key, "admin", tok, now.Add(time.Minute)) || !validToken(key, "admin", tok, now.Add(tokenTTL)) {
		t.Error("a fresh token was rejected")
	}
	ts, sig, _ := strings.Cut(tok, ".")
	for name, c := range map[string]struct {
		user, tok string
		at        time.Time
	}{
		"empty":             {"admin", "", now},
		"another user":      {"operator", tok, now},
		"signed out":        {"", makeToken(key, "", now), now},
		"expired":           {"admin", tok, now.Add(tokenTTL + time.Second)},
		"from the future":   {"admin", tok, now.Add(-time.Minute)},
		"signature changed": {"admin", ts + "." + strings.Repeat("0", len(sig)), now},
		"time changed":      {"admin", "1800000500." + sig, now.Add(time.Minute)},
	} {
		if validToken(key, c.user, c.tok, c.at) {
			t.Errorf("%s: accepted", name)
		}
	}
	// A token made by the Power page's scheme (another purpose string) is not valid here.
	if validToken(key, "admin", "1800000000."+strings.Repeat("a", 64), now) {
		t.Error("a forged signature was accepted")
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 5: "5 B", 1023: "1023 B", 1024: "1.0 KB", 1536: "1.5 KB", 30_000_000: "28.6 MB", 5 << 30: "5.0 GB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}
