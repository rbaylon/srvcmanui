// Package tunnelrouter - srvcmanui UI for the tunnels module.
//
// The list page doubles as the status dashboard: srvcman decorates every
// tunnel with live status/pid/age read from the gateway's process list,
// so the table and the active/inactive counts above it come from the same
// single GET. Start/stop/re-establish are GET links here that proxy to
// srvcman's PUT endpoints and redirect back to the list, the same shape
// gatewaymonitor's "apply routes" action uses.
package tunnelrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	tunnelmodel "github.com/rbaylon/srvcman/modules/tunnels/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// listData is the tunnels list page's view model: the records plus the
// active/inactive tallies shown in the status dashboard above the table.
type listData struct {
	Records  []tunnelmodel.Tunnel
	Active   int
	Inactive int
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []tunnelmodel.Tunnel
		err, status := client.Rester("tunnels", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		data := listData{Records: records}
		for _, t := range records {
			if t.Status == tunnelmodel.StatusActive {
				data.Active++
				continue
			}
			data.Inactive++
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/tunnels/templates/tunnels.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /startinactive", func(w http.ResponseWriter, r *http.Request) {
		var result struct {
			Started int `json:"started"`
			Failed  int `json:"failed"`
		}
		err, status := client.Rester("tunnels/startinactive", "PUT", &result, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/tunnels", http.StatusSeeOther)
	})

	action := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result tunnelmodel.Tunnel
			err, status := client.Rester("tunnels/"+r.PathValue("id")+"/"+name, "PUT", &result, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/tunnels", http.StatusSeeOther)
		}
	}
	Router.HandleFunc("GET /start/{id}", action("start"))
	Router.HandleFunc("GET /stop/{id}", action("stop"))

	type formData struct {
		Record    *tunnelmodel.Tunnel
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/tunnels/templates/tunnelnew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", formData{nil, pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record tunnelmodel.Tunnel
		err, status := client.Rester("tunnels/"+r.PathValue("id"), "GET", &record, r)
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
			pwd + "/modules/tunnels/templates/tunnel.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", formData{&record, pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &tunnelmodel.Tunnel{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			err, status := client.Rester("tunnels/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/tunnels", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("tunnels/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/tunnels", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("tunnels/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/tunnels", http.StatusSeeOther)
	})

	return Router
}
