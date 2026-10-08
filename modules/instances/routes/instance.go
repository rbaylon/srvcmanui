// Package instancerouter - srvcmanui UI for the pfconfig (router) module.
//
// There is exactly one router record per gateway, so this is one form
// rather than a list/edit pair: the page shows the record when there is one
// and an empty form to create it when there is not. srvcman refuses a second
// record, and the router select is hidden on every other form because it
// can only ever have one choice.
package instancerouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pfconfigmodel.Pfconfig
		err, status := client.Rester("pfconfig", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		// An empty record (ID 0) makes the form post as a create. If
		// older data left more than one, the first is shown.
		var record pfconfigmodel.Pfconfig
		if len(records) > 0 {
			record = records[0]
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/instances/templates/instance.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", record); err != nil {
			log.Println(err)
		}
	})

	// Old list/edit/new links all lead to the single page.
	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
	})
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pfconfigmodel.Pfconfig{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("pfconfig/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("pfconfig/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("pfconfig/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
	})

	return Router
}
