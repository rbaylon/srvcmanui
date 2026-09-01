package collectionrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	collectionmodel "github.com/rbaylon/srvcman/modules/collections/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type Formdata struct {
	Record *collectionmodel.Collection
	Subs   []submodel.Sub
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /{year}", func(w http.ResponseWriter, r *http.Request) {
		y := r.PathValue("year")
		records := []collectionmodel.Collection{}
		err, status := client.Rester("collections/year/"+y, "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
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
		err, status := client.Rester("collections/"+collectionid, "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		subs := []submodel.Sub{}
		err, status = client.Rester("subs", "GET", &subs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/colls/templates/coll.html",
		}
		data := Formdata{Record: &record, Subs: subs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", data)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		subs := []submodel.Sub{}
		err, status := client.Rester("subs", "GET", &subs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/colls/templates/coll.html",
		}
		data := Formdata{Record: nil, Subs: subs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", data)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &collectionmodel.Collection{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		// SubName/SubDue/SubPlan are derived server-side from the Sub
		// relationship on every read, so there's no need to look the sub up
		// here just to stamp them onto the request body.
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("collections/"+strconv.FormatUint(uint64(s.ID), 10), "PUT", s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
			return
		}
		err, status := client.Rester("collections/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
	})

	return Router
}
