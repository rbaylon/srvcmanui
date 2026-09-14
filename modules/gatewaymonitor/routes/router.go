package gatewaymonitorrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	gatewaymonitormodel "github.com/rbaylon/srvcman/modules/gatewaymonitor/model"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	pfifacemodel "github.com/rbaylon/srvcman/modules/pfifaces/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	type listData struct {
		Records      []gatewaymonitormodel.GatewayMonitor
		ActiveRoutes string
	}
	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []gatewaymonitormodel.GatewayMonitor
		err, status := client.Rester("gatewaymonitor", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		var activeRoutes struct {
			Output string `json:"output"`
			Error  string `json:"error,omitempty"`
		}
		if err, _ := client.Rester("gatewaymonitor/activeroutes", "GET", &activeRoutes, r); err != nil {
			fmt.Println(err)
		} else if activeRoutes.Error != "" {
			fmt.Println(activeRoutes.Error)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/gatewaymonitor/templates/gatewaymonitors.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "htmx", listData{records, activeRoutes.Output})
	})

	Router.HandleFunc("GET /applyroutes", func(w http.ResponseWriter, r *http.Request) {
		var result struct {
			Applied int `json:"applied"`
			Failed  int `json:"failed"`
		}
		err, status := client.Rester("gatewaymonitor/applyroutes", "PUT", &result, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/gatewaymonitor", http.StatusSeeOther)
	})

	type formData struct {
		Record    *gatewaymonitormodel.GatewayMonitor
		Pfifaces  []pfifacemodel.Pfiface
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record gatewaymonitormodel.GatewayMonitor
		err, status := client.Rester("gatewaymonitor/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		ifs := []pfifacemodel.Pfiface{}
		err, status = client.Rester("pfiface", "GET", &ifs, r)
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
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/gatewaymonitor/templates/gatewaymonitor.html",
		}
		formdata := formData{&record, ifs, pfconfigs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		ifs := []pfifacemodel.Pfiface{}
		err, status := client.Rester("pfiface", "GET", &ifs, r)
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
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/gatewaymonitor/templates/gatewaymonitornew.html",
		}
		formdata := formData{nil, ifs, pfconfigs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &gatewaymonitormodel.GatewayMonitor{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("gatewaymonitor/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/gatewaymonitor", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("gatewaymonitor/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/gatewaymonitor", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("gatewaymonitor/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/gatewaymonitor", http.StatusSeeOther)
	})

	return Router
}
