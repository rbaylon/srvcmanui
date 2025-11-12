package pppoerouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
	pppoemodel "github.com/rbaylon/srvcman/modules/pppoes/model"
	"github.com/rbaylon/srvcmanui/client"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pppoemodel.Pppoe
		err := client.Rester("pfconfig", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pppoes/templates/instances.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	type pppoeFormData struct {
		Record    *pppoemodel.Pppoe
		Plans     []planmodel.Plan
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	type subFormData struct {
		Record   *pppoemodel.Pppoe
		Plans    []planmodel.Plan
		Pfconfig pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pppoemodel.Pppoe
		err := client.Rester("pppoe/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pppoes/templates/pppoe.html",
		}
		plans := []planmodel.Plan{}
		pfconfig := pfconfigmodel.Pfconfig{}
		err = client.Rester("plans", "GET", &plans, r)
		if err != nil {
			log.Println(err)
		}
		err = client.Rester(fmt.Sprintf("pfconfig/%d", record.PfconfigID), "GET", &pfconfig, r)
		if err != nil {
			log.Println(err)
		}
		formdata := subFormData{&record, plans, pfconfig}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		plans := []planmodel.Plan{}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err := client.Rester("plans", "GET", &plans, r)
		if err != nil {
			log.Println(err)
		}
		err = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			log.Println(err)
		}
		formdata := pppoeFormData{nil, plans, pfconfigs}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pppoes/templates/pppoenew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pppoemodel.Pppoe{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err := client.Rester("pppoe/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
			return
		}
		err := client.Rester("pppoe/create", "POST", s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := client.Rester("pppoe/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
	})

	return Router
}
