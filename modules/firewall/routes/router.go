package fwrouter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	fwmodel "github.com/rbaylon/srvcman/modules/firewall/model"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []fwmodel.Fw
		err, status := client.Rester("firewall/?option=rules", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/firewall/templates/fws.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	Router.HandleFunc("GET /pfconfig/{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pfconfigmodel.Pfconfig
		err, status := client.Rester("pfconfig/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(record)
	})

	Router.HandleFunc("GET /pfconfig", func(w http.ResponseWriter, r *http.Request) {
		var records []pfconfigmodel.Pfconfig
		err, status := client.Rester("pfconfig", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	})

	type fwFormData struct {
		Record    *fwmodel.Fw
		Plans     []planmodel.Plan
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	type subFormData struct {
		Record   *fwmodel.Fw
		Plans    []planmodel.Plan
		Pfconfig pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record fwmodel.Fw
		err, status := client.Rester("firewall/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/firewall/templates/fw.html",
		}
		plans := []planmodel.Plan{}
		pfconfig := pfconfigmodel.Pfconfig{}
		err, status = client.Rester("plans", "GET", &plans, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		err, status = client.Rester(fmt.Sprintf("pfconfig/%d", record.PfconfigID), "GET", &pfconfig, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		formdata := subFormData{&record, plans, pfconfig}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		plans := []planmodel.Plan{}
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status := client.Rester("plans", "GET", &plans, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		err, status = client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		formdata := fwFormData{nil, plans, pfconfigs}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/firewall/templates/fwnew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &fwmodel.Fw{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("firewall/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/firewall", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("firewall/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/firewall", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("firewall/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/firewall", http.StatusSeeOther)
	})

	return Router
}
