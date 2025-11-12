package instancerouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	"github.com/rbaylon/srvcmanui/client"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pfconfigmodel.Pfconfig
		err := client.Rester("pfconfig", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		log.Printf("%v\n", records[0])
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/instances/templates/instances.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pfconfigmodel.Pfconfig
		err := client.Rester("pfconfig/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/instances/templates/instance.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", record)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/instances/templates/instance.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "htmx", nil)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pfconfigmodel.Pfconfig{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err := client.Rester("pfconfig/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
			return
		}
		err := client.Rester("pfconfig/create", "POST", s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := client.Rester("pfconfig/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/instances", http.StatusSeeOther)
	})

	return Router
}
