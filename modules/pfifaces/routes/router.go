package pfifacerouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	pfifacemodel "github.com/rbaylon/srvcman/modules/pfifaces/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pfifacemodel.Pfiface
		err, status := client.Rester("pfiface", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
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
		Record    *pfifacemodel.Pfiface
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pfifacemodel.Pfiface
		err, status := client.Rester("pfiface/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		data := Formdata{Record: &record, Pfconfigs: pfconfigs}
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
		err, status := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pfifaces/templates/pfiface.html",
		}
		data := Formdata{nil, pfconfigs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", data)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pfifacemodel.Pfiface{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("pfiface/"+strconv.FormatUint(uint64(id), 10), "PUT", &s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("pfiface/create", "POST", &s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("pfiface/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/pfifaces", http.StatusSeeOther)
	})
	return Router
}
