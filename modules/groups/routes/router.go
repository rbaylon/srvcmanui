package grouprouter

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	groupmodel "github.com/rbaylon/srvcman/modules/groups/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records := []groupmodel.Group{}
		err, status := client.Rester("groups", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/groups/templates/groups.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/groups/templates/groupnew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "htmx", nil)
		if err != nil {
			fmt.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &groupmodel.Group{}
		s.Groupname = r.FormValue("Groupname")
		fid := r.FormValue("ID")
		if fid != "" {
			rid, err := strconv.Atoi(fid)
			if err != nil {
				fmt.Println("error converting ID to string: ", rid)
				http.Redirect(w, r, "/ui/groups", http.StatusSeeOther)
				return
			}
			s.ID = uint(rid)
		} else {
			s.ID = 0
		}

		s.Groupname = r.FormValue("Groupname")
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("groups/"+strconv.FormatUint(uint64(id), 10), "PUT", &s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/groups", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("groups/create", "POST", &s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/groups", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("groups/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/groups", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record groupmodel.Group
		err, status := client.Rester("groups/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/groups/templates/group.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", record)
		if err != nil {
			fmt.Println(err)
		}
	})

	return Router
}
