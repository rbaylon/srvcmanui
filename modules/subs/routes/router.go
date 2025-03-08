package subrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	runtimemodel "github.com/rbaylon/srvcman/modules/runtimes/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
)

func New(db submodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records, err := db.GetAll()
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
		subid, _ := strconv.Atoi(r.PathValue("id"))
		record, err := db.GetById(uint(subid))
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
		dbconn := db.GetDB()
		rtdb := runtimemodel.New(dbconn)
		rt := &runtimemodel.Runtime{}
		rt.Action = "updatepf"
		rt.Status = "active"
		if id > 0 {
			s.ID = uint(id)
			err := db.Update(s)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
				return
			}
			err = rtdb.Add(rt)
			if err != nil {
				log.Println("update pf failed")
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
		err := db.Add(s)
		if err != nil {
			log.Println(err)
		}
		err = rtdb.Add(rt)
		if err != nil {
			log.Println("update pf failed")
		}
		http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
	})

	return Router
}
