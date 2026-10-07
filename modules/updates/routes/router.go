// Package updatesrouter - srvcmanui UI for the updates module.
//
// One page: the update server setting, and an OS patch check. The check
// runs in the background on srvcman (syspatch can take minutes), so the
// button only starts it and sends you back to the page, which shows its
// progress and refreshes itself until it finishes. Checking only lists
// patches - nothing is installed from here.
//
// The settings and status structs are local rather than imported from
// srvcman's updates model: they are just the JSON fields the API
// exchanges, and keeping them here means this UI builds without waiting
// on a srvcman release that carries the model.
package updatesrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type settings struct {
	UpdateServer string `json:"update_server"`
}

// checkStatus mirrors srvcman's check state: State is idle, running, done
// or failed.
type checkStatus struct {
	State      string     `json:"state"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Available  []string   `json:"available"`
	Error      string     `json:"error"`
}

type viewdata struct {
	UpdateServer string
	Saved        bool
	Check        checkStatus
}

func page(w http.ResponseWriter, data viewdata) {
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/updates/templates/updates.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var s settings
		if err, status := client.Rester("updates", "GET", &s, r); err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		var c checkStatus
		if err, status := client.Rester("updates/check", "GET", &c, r); err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		page(w, viewdata{UpdateServer: s.UpdateServer, Saved: r.URL.Query().Get("saved") == "1", Check: c})
	})

	Router.HandleFunc("POST /save", func(w http.ResponseWriter, r *http.Request) {
		s := settings{UpdateServer: strings.TrimSpace(r.FormValue("UpdateServer"))}
		if err, status := client.Rester("updates", "PUT", &s, r); err != nil {
			log.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/updates?saved=1", http.StatusSeeOther)
	})

	// Starts the background check and returns to the page straight away.
	// Pressing it again while one is running is harmless: srvcman leaves
	// the running check alone.
	Router.HandleFunc("GET /check", func(w http.ResponseWriter, r *http.Request) {
		var c checkStatus
		if err, status := client.Rester("updates/check", "PUT", &c, r); err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/updates", http.StatusSeeOther)
	})

	return Router
}
