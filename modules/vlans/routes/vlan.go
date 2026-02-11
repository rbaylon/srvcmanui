package vlanrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	pfifacemodel "github.com/rbaylon/srvcman/modules/pfifaces/model"
	vlanmodel "github.com/rbaylon/srvcman/modules/vlans/model"
	"github.com/rbaylon/srvcmanui/client"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []vlanmodel.Vlan
		err := client.Rester("vlan", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vlans/templates/vlans.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	type subFormData struct {
		Record    *vlanmodel.Vlan
		Pfifaces  []pfifacemodel.Pfiface
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record vlanmodel.Vlan
		err := client.Rester("vlan/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vlans/templates/vlan.html",
		}
		ifs := []pfifacemodel.Pfiface{}
		err = client.Rester("pfiface", "GET", &ifs, r)
		if err != nil {
			log.Println(err)
		}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			log.Println(err)
		}
		formdata := subFormData{&record, ifs, pfconfigs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vlans/templates/vlannew.html",
		}
		ifs := []pfifacemodel.Pfiface{}
		err := client.Rester("pfiface", "GET", &ifs, r)
		if err != nil {
			log.Println(err)
		}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			log.Println(err)
		}
		formdata := subFormData{nil, ifs, pfconfigs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &vlanmodel.Vlan{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/vlan", http.StatusSeeOther)
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err := client.Rester("vlan/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/vlan", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/ui/vlan", http.StatusSeeOther)
			return
		}
		err := client.Rester("vlan/create", "POST", s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/vlan", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := client.Rester("vlan/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/vlan", http.StatusSeeOther)
	})

	return Router
}
