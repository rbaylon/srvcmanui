// Package pflowrouter - srvcmanui UI for the pflows module.
//
// Standard CRUD, same shape as staticroutes. Note the API path is
// "pflow" (singular) while the UI lives at /ui/pflows - that mismatch is
// srvcman's, where the module's registry key and mount point are
// "pflow" but its package and table are pflows.
//
// Edits here only reach the box once the pending change is applied:
// modules/changes' apply flow sends ApplyIfaces, and arkgated's
// ConfigCreate renders each pflow device's hostname.<if> and promotes it.
package pflowrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	pflowmodel "github.com/rbaylon/srvcman/modules/pflows/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pflowmodel.Pflow
		err, status := client.Rester("pflow", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pflows/templates/pflows.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", records); err != nil {
			log.Println(err)
		}
	})

	type formData struct {
		Record    *pflowmodel.Pflow
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pflows/templates/pflownew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", formData{nil, pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pflowmodel.Pflow
		err, status := client.Rester("pflow/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pflows/templates/pflow.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", formData{&record, pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pflowmodel.Pflow{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			err, status := client.Rester("pflow/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/pflows", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("pflow/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/pflows", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("pflow/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/pflows", http.StatusSeeOther)
	})

	return Router
}
