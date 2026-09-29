// Package acmerouter - srvcmanui UI for the acme module.
//
// Single-entry config, so this is one form rather than a list/edit pair:
// the page shows the record if there is one and an empty form if not, and
// the Request button appears only once a record is enabled.
package acmerouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	acmemodel "github.com/rbaylon/srvcman/modules/acme/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// viewdata backs the page. Result/ResultError carry the outcome of a
// certificate request so it can be shown inline - acme-client's own output
// is the useful part when issuance fails (an unresolvable name, a
// challenge that never came back), so it is rendered rather than reduced
// to a status code.
type viewdata struct {
	Record      *acmemodel.Acme
	Conf        string
	Result      string
	ResultError string
}

// load fetches the single record plus the rendered config.
func load(r *http.Request) (*acmemodel.Acme, string, error, int) {
	var records []acmemodel.Acme
	if err, status := client.Rester("acme", "GET", &records, r); err != nil {
		return nil, "", err, status
	}
	var conf struct {
		Conf string `json:"conf"`
	}
	if err, status := client.Rester("acme/conf", "GET", &conf, r); err != nil {
		return nil, "", err, status
	}
	if len(records) == 0 {
		return nil, conf.Conf, nil, http.StatusOK
	}
	return &records[0], conf.Conf, nil, http.StatusOK
}

func page(w http.ResponseWriter, data viewdata) {
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/acme/templates/acme.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		record, conf, err, status := load(r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		page(w, viewdata{Record: record, Conf: conf})
	})

	// Apply reaches out to Let's Encrypt and restarts nginx, so its result
	// is rendered back onto the page rather than redirecting - the output
	// is the whole point, and a failure here is something the operator
	// needs to read, not a flash message.
	Router.HandleFunc("GET /apply", func(w http.ResponseWriter, r *http.Request) {
		record, conf, err, status := load(r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		var result struct {
			Result string `json:"result"`
		}
		data := viewdata{Record: record, Conf: conf}
		if err, _ := client.Rester("acme/apply", "PUT", &result, r); err != nil {
			data.ResultError = err.Error()
		} else {
			data.Result = result.Result
		}
		page(w, data)
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &acmemodel.Acme{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			err, status := client.Rester("acme/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/acme", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("acme/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/acme", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("acme/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/acme", http.StatusSeeOther)
	})

	return Router
}
