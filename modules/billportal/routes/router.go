// Package billportalrouter - srvcmanui UI for the billportal module.
//
// Same shape as captiveportalrouter: the page carries both the app's
// configuration (a single record, written out as its .env on save) and
// the service controls that proxy to srvcman's /api/v1/billportal/*
// endpoints.
//
// API_AUTH is deliberately absent from the form. srvcman tags it json:"-",
// so it never reaches this process at all - it is generated there
// alongside a read-only API user and only ever appears in the .env written
// to the gateway.
package billportalrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	billportalmodel "github.com/rbaylon/srvcman/modules/billportal/model"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type viewdata struct {
	Record *billportalmodel.BillPortal
	// Pfconfigs backs the Router select - ROUTER_ID is set from whichever
	// one is chosen, rather than entered as free text.
	Pfconfigs []pfconfigmodel.Pfconfig
	Running   bool
	Output    string
	Error     string
	EnvPath   string
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
	if err, _ := client.Rester("billportal/status", "GET", &result, r); err != nil {
		return false, "", err.Error()
	}
	return result.Running, result.Output, ""
}

func page(w http.ResponseWriter, data viewdata) {
	data.EnvPath = billportalmodel.EnvPath
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

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []billportalmodel.BillPortal
		err, code := client.Rester("billportal", "GET", &records, r)
		if err != nil {
			render500(w, code, err)
			return
		}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		if err, code := client.Rester("pfconfig", "GET", &pfconfigs, r); err != nil {
			render500(w, code, err)
			return
		}
		running, output, statusErr := status(r)
		data := viewdata{Running: running, Output: output, Error: statusErr, Pfconfigs: pfconfigs}
		if len(records) > 0 {
			data.Record = &records[0]
		}
		page(w, data)
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &billportalmodel.BillPortal{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			err, code := client.Rester("billportal/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				render500(w, code, err)
				return
			}
			http.Redirect(w, r, "/ui/billportal", http.StatusSeeOther)
			return
		}
		err, code := client.Rester("billportal/create", "POST", s, r)
		if err != nil {
			render500(w, code, err)
			return
		}
		http.Redirect(w, r, "/ui/billportal", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, code := client.Rester("billportal/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			render500(w, code, err)
			return
		}
		http.Redirect(w, r, "/ui/billportal", http.StatusSeeOther)
	})

	action := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result map[string]string
			err, code := client.Rester("billportal/"+name, "PUT", &result, r)
			if err != nil {
				render500(w, code, err)
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
