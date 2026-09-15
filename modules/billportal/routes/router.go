// Package billportalrouter - srvcmanui UI for the billportal module.
//
// Same shape as captiveportalrouter: a status display plus start/stop/
// restart buttons that proxy to srvcman's /api/v1/billportal/* endpoints,
// which in turn just ask arkgated to rcctl the separate billportal app's
// "billportal" rc.d service.
package billportalrouter

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
		pwd + "/modules/billportal/templates/billportal.html",
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
	err, _ := client.Rester("billportal/status", "GET", &result, r)
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
			err, httpStatus := client.Rester("billportal/"+name, "PUT", &result, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, httpStatus, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/billportal", http.StatusSeeOther)
		}
	}
	Router.HandleFunc("GET /start", action("start"))
	Router.HandleFunc("GET /stop", action("stop"))
	Router.HandleFunc("GET /restart", action("restart"))

	return Router
}
