package pfifacerouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	pfifacemodel "github.com/rbaylon/srvcman/modules/pfifaces/model"
	"github.com/rbaylon/srvcmanui/client"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pfifacemodel.Pfiface
		err := client.Rester("pfiface", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pfifaces/templates/pfifaces.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", records); err != nil {
			log.Println("template error:", err)
		}
	})

	type Formdata struct {
		Record    pfifacemodel.Pfiface
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pfifacemodel.Pfiface
		err := client.Rester("pfiface/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		data := Formdata{Record: record, Pfconfigs: pfconfigs}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pfifaces/templates/pfiface.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", data)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			log.Println(err)
		}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pfifaces/templates/pfiface.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", pfconfigs)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pfifacemodel.Pfiface{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err := client.Rester("pfiface/"+strconv.FormatUint(uint64(id), 10), "PUT", &s, r)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
			return
		}
		err := client.Rester("pfiface/create", "POST", &s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := client.Rester("pfiface/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
	})
	return Router
}
