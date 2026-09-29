// Package captiveportalrouter - srvcmanui UI for the captiveportal module.
//
// The page carries both the app's configuration (a single record, written
// out as its .env on save) and the service controls that proxy to
// srvcman's /api/v1/captiveportal/* endpoints.
//
// API_AUTH is deliberately absent from the form. srvcman tags it json:"-",
// so it never reaches this process at all - it is generated there
// alongside a read-only API user and only ever appears in the .env written
// to the gateway.
package captiveportalrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	captiveportalmodel "github.com/rbaylon/srvcman/modules/captiveportal/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type viewdata struct {
	Record  *captiveportalmodel.CaptivePortal
	Running bool
	Output  string
	Error   string
	EnvPath string
}

func render500(w http.ResponseWriter, status int, err error) {
	fmt.Println(err)
	utils.RenderErrorPage(w, status, err.Error())
}

// status fetches current state for the index page. A failure to reach
// srvcman/arkgated is shown inline (Error) rather than a full error page,
// since "can't tell right now" is a normal, recoverable state for a status
// display - the start/stop/restart actions below still get the full error
// page treatment, since those are the ones a user needs to notice failed.
func status(r *http.Request) (bool, string, string) {
	var result struct {
		Running bool   `json:"running"`
		Output  string `json:"output"`
	}
	if err, _ := client.Rester("captiveportal/status", "GET", &result, r); err != nil {
		return false, "", err.Error()
	}
	return result.Running, result.Output, ""
}

func page(w http.ResponseWriter, data viewdata) {
	data.EnvPath = captiveportalmodel.EnvPath
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

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []captiveportalmodel.CaptivePortal
		err, code := client.Rester("captiveportal", "GET", &records, r)
		if err != nil {
			render500(w, code, err)
			return
		}
		running, output, statusErr := status(r)
		data := viewdata{Running: running, Output: output, Error: statusErr}
		if len(records) > 0 {
			data.Record = &records[0]
		}
		page(w, data)
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &captiveportalmodel.CaptivePortal{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			err, code := client.Rester("captiveportal/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				render500(w, code, err)
				return
			}
			http.Redirect(w, r, "/ui/captiveportal", http.StatusSeeOther)
			return
		}
		err, code := client.Rester("captiveportal/create", "POST", s, r)
		if err != nil {
			render500(w, code, err)
			return
		}
		http.Redirect(w, r, "/ui/captiveportal", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, code := client.Rester("captiveportal/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			render500(w, code, err)
			return
		}
		http.Redirect(w, r, "/ui/captiveportal", http.StatusSeeOther)
	})

	action := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result map[string]string
			err, code := client.Rester("captiveportal/"+name, "PUT", &result, r)
			if err != nil {
				render500(w, code, err)
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
