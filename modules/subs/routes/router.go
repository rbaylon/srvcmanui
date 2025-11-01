package subrouter

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
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	"github.com/rbaylon/srvcmanui/client"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []submodel.Sub
		err := client.Rester("subs", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/subs/templates/subs.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	type subFormData struct {
		Record   *submodel.Sub
		Plans    []planmodel.Plan
		Pfconfig pfconfigmodel.Pfconfig
	}

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record submodel.Sub
		var pfc pfconfigmodel.Pfconfig
		err := client.Rester("subs/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		err = client.Rester(fmt.Sprintf("pfconfig/%d", record.PfconfigID), "GET", &pfc, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/subs/templates/sub.html",
		}
		plans := []planmodel.Plan{}
		err = client.Rester("plans", "GET", &plans, r)
		if err != nil {
			log.Println(err)
		}
		formdata := subFormData{&record, plans, pfc}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	type fwFormData struct {
		Record    *submodel.Sub
		Plans     []planmodel.Plan
		Pfconfigs []pfconfigmodel.Pfconfig
	}

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
		formdata := fwFormData{nil, plans, pfconfigs}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/subs/templates/subnew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &submodel.Sub{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err := client.Rester("subs/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
			return
		}
		s.Burstspeed = 60
		s.Downspeed = 15
		s.Upspeed = 10
		s.Duration = 3000
		err := client.Rester("subs/create", "POST", s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := client.Rester("subs/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
	})

	return Router
}
