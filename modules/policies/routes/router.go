package policyrouter

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	groupmodel "github.com/rbaylon/srvcman/modules/groups/model"
	policymodel "github.com/rbaylon/srvcman/modules/policies/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux()

	type formdata struct {
		Policy *policymodel.Policy
		Groups []groupmodel.Group
	}

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records := []policymodel.Policy{}
		err, status := client.Rester("policies", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/policies/templates/policies.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		groups := []groupmodel.Group{}
		err, status := client.Rester("groups", "GET", &groups, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		fd := formdata{nil, groups}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/policies/templates/policynew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", fd)
		if err != nil {
			fmt.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &policymodel.Policy{}
		s.Name = r.FormValue("Name")
		s.Type = r.FormValue("Module")
		s.Action = r.FormValue("Action")
		groups := []groupmodel.Group{}
		for _, v := range r.Form["Groups"] {
			g := groupmodel.Group{}
			err, _ := client.Rester("groups/getbyname/"+v, "GET", &g, r)
			if err != nil {
				fmt.Println("error getting group: ", err)
			} else {
				groups = append(groups, g)
			}
		}
		s.Groups = groups
		fid := r.FormValue("ID")
		if fid != "" {
			rid, err := strconv.Atoi(fid)
			if err != nil {
				fmt.Println("error converting ID to string: ", rid)
				http.Redirect(w, r, "/ui/policies", http.StatusSeeOther)
				return
			}
			s.ID = uint(rid)
		} else {
			s.ID = 0
		}

		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("policies/"+strconv.FormatUint(uint64(id), 10), "PUT", &s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/policies", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("policies/create", "POST", &s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/policies", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("policies/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/policies", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record policymodel.Policy
		err, status := client.Rester("policies/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		groups := []groupmodel.Group{}
		err, status = client.Rester("groups", "GET", &groups, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		fd := formdata{&record, groups}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/policies/templates/policy.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", fd)
		if err != nil {
			fmt.Println(err)
		}
	})

	return Router
}
