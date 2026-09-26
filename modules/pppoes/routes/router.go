package pppoerouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/render"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
	pppoemodel "github.com/rbaylon/srvcman/modules/pppoes/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// listData is the pppoe list page: the configured records, plus the live
// status of the npppd service they describe - same status/start/stop/
// restart shape as the billportal/captiveportal modules, just folded into
// this module's existing CRUD list instead of a standalone page.
type listData struct {
	Records     []pppoemodel.Pppoe
	Running     bool
	Output      string
	StatusError string
}

// serviceStatus fetches npppd's current state. A failure to reach
// srvcman/arkgated is shown inline (StatusError) rather than a full error
// page, since "can't tell right now" is a normal, recoverable state for a
// status display - the start/stop/restart actions still get the full
// error page treatment, since those are the ones a user needs to notice
// failed.
func serviceStatus(r *http.Request) (running bool, output string, statusErr string) {
	var result struct {
		Running bool   `json:"running"`
		Output  string `json:"output"`
	}
	if err, _ := client.Rester("pppoe/status", "GET", &result, r); err != nil {
		return false, "", err.Error()
	}
	return result.Running, result.Output, ""
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []pppoemodel.Pppoe
		err, status := client.Rester("pppoe", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pppoes/templates/pppoes.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		running, output, statusErr := serviceStatus(r)
		tmpl.ExecuteTemplate(w, "htmx", listData{Records: records, Running: running, Output: output, StatusError: statusErr})
	})

	type pppoeFormData struct {
		Record    *pppoemodel.Pppoe
		Plans     []planmodel.Plan
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	type subFormData struct {
		Record   *pppoemodel.Pppoe
		Plans    []planmodel.Plan
		Pfconfig pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record pppoemodel.Pppoe
		err, status := client.Rester("pppoe/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pppoes/templates/pppoe.html",
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
		formdata := pppoeFormData{nil, plans, pfconfigs}
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/pppoes/templates/pppoenew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &pppoemodel.Pppoe{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("pppoe/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("pppoe/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("pppoe/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
	})

	serviceAction := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result map[string]string
			err, httpStatus := client.Rester("pppoe/"+name, "PUT", &result, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, httpStatus, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/pppoes", http.StatusSeeOther)
		}
	}
	Router.HandleFunc("GET /start", serviceAction("start"))
	Router.HandleFunc("GET /stop", serviceAction("stop"))
	Router.HandleFunc("GET /restart", serviceAction("restart"))

	return Router
}
