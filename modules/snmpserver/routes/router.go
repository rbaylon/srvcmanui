// Package snmpserverrouter - srvcmanui UI for the snmpserver module.
//
// Same shape as the dhcpserver UI: CRUD list carrying the snmpd service's
// live status and start/stop/restart controls, plus a view of the
// generated snmpd.conf.
//
// snmpd.conf is a single document rather than a list of blocks, so only
// one record is ever rendered (see snmpservermodel.GenerateConf) and the
// New button is hidden once one exists.
package snmpserverrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	snmpservermodel "github.com/rbaylon/srvcman/modules/snmpserver/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// listData is the snmpserver list page: the configured record, plus the
// live status of the snmpd service it describes.
type listData struct {
	Records     []snmpservermodel.SnmpServer
	Running     bool
	Output      string
	StatusError string
}

// serviceStatus fetches snmpd's current state. A failure to reach
// srvcman/arkgated is shown inline rather than as a full error page, since
// "can't tell right now" is a normal state for a status display - the
// start/stop/restart actions still get the error page, since those are the
// ones a user needs to notice failed.
func serviceStatus(r *http.Request) (running bool, output string, statusErr string) {
	var result struct {
		Running bool   `json:"running"`
		Output  string `json:"output"`
	}
	if err, _ := client.Rester("snmpserver/status", "GET", &result, r); err != nil {
		return false, "", err.Error()
	}
	return result.Running, result.Output, ""
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []snmpservermodel.SnmpServer
		err, status := client.Rester("snmpserver", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/snmpserver/templates/snmpservers.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		running, output, statusErr := serviceStatus(r)
		if err := tmpl.ExecuteTemplate(w, "htmx", listData{Records: records, Running: running, Output: output, StatusError: statusErr}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /conf", func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			Conf string `json:"conf"`
		}
		err, status := client.Rester("snmpserver/conf", "GET", &data, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/snmpserver/templates/snmpserverconf.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", data.Conf); err != nil {
			log.Println(err)
		}
	})

	action := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result map[string]string
			err, status := client.Rester("snmpserver/"+name, "PUT", &result, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/snmpserver", http.StatusSeeOther)
		}
	}
	Router.HandleFunc("GET /start", action("start"))
	Router.HandleFunc("GET /stop", action("stop"))
	Router.HandleFunc("GET /restart", action("restart"))

	type formData struct {
		Record    *snmpservermodel.SnmpServer
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
			pwd + "/modules/snmpserver/templates/snmpservernew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", formData{nil, pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record snmpservermodel.SnmpServer
		err, status := client.Rester("snmpserver/"+r.PathValue("id"), "GET", &record, r)
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
			pwd + "/modules/snmpserver/templates/snmpserver.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", formData{&record, pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &snmpservermodel.SnmpServer{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			err, status := client.Rester("snmpserver/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/snmpserver", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("snmpserver/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/snmpserver", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("snmpserver/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/snmpserver", http.StatusSeeOther)
	})

	return Router
}
