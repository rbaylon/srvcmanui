package diagnosticsrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type viewdata struct {
	Ip     string
	Action string
	Output string
}

func render(w http.ResponseWriter, data viewdata) {
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/diagnostics/templates/diagnostics.html",
	}
	tmpl := template.Must(template.ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		render(w, viewdata{})
	})

	Router.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		var result struct {
			Output string `json:"output"`
		}
		err, status := client.Rester("diagnostics/ping/"+url.PathEscape(ip), "GET", &result, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		render(w, viewdata{Ip: ip, Action: "ping", Output: result.Output})
	})

	Router.HandleFunc("GET /traceroute", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		var result struct {
			Output string `json:"output"`
		}
		err, status := client.Rester("diagnostics/traceroute/"+url.PathEscape(ip), "GET", &result, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		render(w, viewdata{Ip: ip, Action: "traceroute", Output: result.Output})
	})

	return Router
}
