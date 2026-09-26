package unbounddnsrouter

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
	unbounddnsmodel "github.com/rbaylon/srvcman/modules/unbounddns/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// listData is the unbounddns list page: the configured records, plus the
// live status of the unbound service they describe - same status/start/
// stop/restart shape as the billportal/captiveportal modules, just folded
// into this module's existing CRUD list instead of a standalone page.
type listData struct {
	Records     []unbounddnsmodel.UnboundDns
	Running     bool
	Output      string
	StatusError string
}

// serviceStatus fetches unbound's current state. A failure to reach
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
	if err, _ := client.Rester("unbounddns/status", "GET", &result, r); err != nil {
		return false, "", err.Error()
	}
	return result.Running, result.Output, ""
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []unbounddnsmodel.UnboundDns
		err, status := client.Rester("unbounddns", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/unbounddns/templates/unbounddnss.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		running, output, statusErr := serviceStatus(r)
		tmpl.ExecuteTemplate(w, "htmx", listData{Records: records, Running: running, Output: output, StatusError: statusErr})
	})

	Router.HandleFunc("GET /conf", func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			Conf string `json:"conf"`
		}
		err, status := client.Rester("unbounddns/conf", "GET", &data, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/unbounddns/templates/unbounddnsconf.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", data.Conf)
		if err != nil {
			log.Println(err)
		}
	})

	type formData struct {
		Record    *unbounddnsmodel.UnboundDns
		Pfifaces  []pfifacemodel.Pfiface
		Pfconfigs []pfconfigmodel.Pfconfig
	}
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record unbounddnsmodel.UnboundDns
		err, status := client.Rester("unbounddns/"+r.PathValue("id"), "GET", &record, r)
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
			pwd + "/modules/unbounddns/templates/unbounddns.html",
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
			pwd + "/modules/unbounddns/templates/unbounddnsnew.html",
		}
		formdata := formData{nil, ifs, pfconfigs}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", formdata)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &unbounddnsmodel.UnboundDns{}
		if err := render.Bind(r, s); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		id := s.ID
		if id > 0 {
			s.ID = uint(id)
			err, status := client.Rester("unbounddns/"+strconv.FormatUint(uint64(id), 10), "PUT", s, r)
			if err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/unbounddns", http.StatusSeeOther)
			return
		}
		err, status := client.Rester("unbounddns/create", "POST", s, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/unbounddns", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("unbounddns/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/unbounddns", http.StatusSeeOther)
	})

	serviceAction := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var result map[string]string
			err, httpStatus := client.Rester("unbounddns/"+name, "PUT", &result, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, httpStatus, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/unbounddns", http.StatusSeeOther)
		}
	}
	Router.HandleFunc("GET /start", serviceAction("start"))
	Router.HandleFunc("GET /stop", serviceAction("stop"))
	Router.HandleFunc("GET /restart", serviceAction("restart"))

	return Router
}
