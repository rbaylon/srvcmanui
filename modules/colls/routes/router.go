package collectionrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	collectionmodel "github.com/rbaylon/srvcman/modules/collections/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	"github.com/rbaylon/srvcmanui/client"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /{year}", func(w http.ResponseWriter, r *http.Request) {
		y := r.PathValue("year")
		records := []collectionmodel.Collection{}
		err := client.Rester("collections/year/"+y, "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/colls/templates/colls.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", records)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /{year}/{id}", func(w http.ResponseWriter, r *http.Request) {
		collectionid := r.PathValue("id")
		record := collectionmodel.Collection{}
		err := client.Rester("collections/"+collectionid, "GET", &record, r)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/colls/templates/coll.html",
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
			pwd + "/modules/colls/templates/coll.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "htmx", nil)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &collectionmodel.Collection{}
		if err := render.Bind(r, s); err != nil {
			log.Println(err)
			http.Redirect(w, r, "/ui/collection/"+s.Year, http.StatusSeeOther)
			return
		}
		sub := submodel.Sub{}
		err := client.Rester("subs/"+strconv.FormatUint(uint64(s.SubID), 10), "GET", &sub, r)
		if err != nil {
			log.Println(err)
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			s.SubName = sub.FirstName + " " + sub.LastName
			s.SubDue = sub.DueDay
			s.SubPlan = sub.Plan
			err = client.Rester("collections/"+strconv.FormatUint(uint64(s.ID), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
			}
			http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
			return
		}
		s.SubName = sub.FirstName + " " + sub.LastName
		s.SubDue = sub.DueDay
		err = client.Rester("collections/create", "POST", s, r)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
	})

	return Router
}
