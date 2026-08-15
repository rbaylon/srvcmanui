package subrouter

import (
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

var letters = []byte("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []submodel.Sub
		err, status := client.Rester("subs", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/subs/templates/subs.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", records)
	})

	type subFormData struct {
		Record   *submodel.Sub
		Plans    []planmodel.Plan
		Pfconfig pfconfigmodel.Pfconfig
	}

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record submodel.Sub
		var pfc pfconfigmodel.Pfconfig
		err, status := client.Rester("subs/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		err, status = client.Rester(fmt.Sprintf("pfconfig/%d", record.PfconfigID), "GET", &pfc, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/subs/templates/sub.html",
		}
		plans := []planmodel.Plan{}
		err, status = client.Rester("plans", "GET", &plans, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		formdata := subFormData{&record, plans, pfc}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	type fwFormData struct {
		Record    *submodel.Sub
		Plans     []planmodel.Plan
		Pfconfigs []pfconfigmodel.Pfconfig
	}

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
			pwd + "/modules/subs/templates/subnew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
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
			err, status := client.Rester("subs/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
			return
		}
		code := []byte("000000000000")
		maxlen := len(letters) - 1
		for j := 0; j < 12; j++ {
			c := letters[rand.IntN(maxlen)]
			code[j] = c
		}
		s.Mac = "subid" + string(code)
		s.Burstspeed = 60
		s.Downspeed = 15
		s.Upspeed = 10
		s.Duration = 3000
		err, status := client.Rester("subs/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("subs/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
	})

	return Router
}
