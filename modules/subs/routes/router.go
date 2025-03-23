package subrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
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
			pwd + "/layouts/darknav.html",
			pwd + "/modules/subs/templates/subs.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "darktheme", records)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record submodel.Sub
		err := client.Rester("subs/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/subs/templates/sub.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "darktheme", record)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/subs/templates/sub.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "darktheme", nil)
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

	return Router
}
