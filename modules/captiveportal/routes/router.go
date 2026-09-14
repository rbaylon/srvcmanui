// Package captiveportalrouter - srvcmanui UI for the captiveportal module.
//
// Like the diagnostics UI module, there's no CRUD form here: this is a
// status display plus start/stop/restart buttons that proxy to srvcman's
// /api/v1/captiveportal/* endpoints, which in turn just ask arkgated to
// rcctl the separate subsportal app's "captiveportal" rc.d service.
package captiveportalrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type viewdata struct {
	Running bool
	Output  string
	Error   string
}

func render(w http.ResponseWriter, data viewdata) {
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/captiveportal/templates/captiveportal.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

// status fetches current state for the index page. A failure to reach
// srvcman/arkgated is shown inline (Error) rather than a full error page,
// since "can't tell right now" is a normal, recoverable state for a status
// display - the start/stop/restart actions below still get the full error
// page treatment, since those are the ones a user needs to notice failed.
func status(r *http.Request) viewdata {
	var result struct {
		Running bool   `json:"running"`
		Output  string `json:"output"`
	}
	err, _ := client.Rester("captiveportal/status", "GET", &result, r)
	if err != nil {
		return viewdata{Error: err.Error()}
	}
	return viewdata{Running: result.Running, Output: result.Output}
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		render(w, status(r))
	})

	action := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result map[string]string
			err, httpStatus := client.Rester("captiveportal/"+name, "PUT", &result, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, httpStatus, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/captiveportal", http.StatusSeeOther)
		}
	}
	Router.HandleFunc("GET /start", action("start"))
	Router.HandleFunc("GET /stop", action("stop"))
	Router.HandleFunc("GET /restart", action("restart"))

	return Router
}
