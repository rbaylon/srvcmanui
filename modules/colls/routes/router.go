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
)

func New(db collectionmodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /{year}", func(w http.ResponseWriter, r *http.Request) {
		y := r.PathValue("year")
		records, err := db.GetByYear(y)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/colls/templates/colls.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "darktheme", records)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /{year}/{id}", func(w http.ResponseWriter, r *http.Request) {
		collectionid, _ := strconv.Atoi(r.PathValue("id"))
		record, err := db.GetById(uint(collectionid))
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/colls/templates/coll.html",
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
			pwd + "/modules/colls/templates/coll.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "darktheme", nil)
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
		dbconn := db.GetDB()
		subdb := submodel.New(dbconn)
		sub, _ := subdb.GetById(uint(s.SubID))
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			s.SubName = sub.FirstName + " " + sub.LastName
			s.SubDue = sub.DueDay
			s.SubPlan = sub.Plan
			err := db.Update(s)
			if err != nil {
				log.Println(err)
			}
			http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
			return
		}
		s.SubName = sub.FirstName + " " + sub.LastName
		s.SubDue = sub.DueDay
		err := db.Add(s)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
	})

	return Router
}
