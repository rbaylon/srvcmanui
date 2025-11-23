package dashrouter

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		stats, _ := os.ReadFile("/tmp/mystats")
		var data interface{}
		err := json.Unmarshal(stats, &data)
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	Router.HandleFunc("GET /usertraffic", func(w http.ResponseWriter, r *http.Request) {
		stats, _ := os.ReadFile("/tmp/usertraffic")
		var data interface{}
		err := json.Unmarshal(stats, &data)
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	type viewdata struct {
		Systats     any
		Usertraffic any
	}

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		stats, _ := os.ReadFile("/tmp/mystats")
		var statsdata interface{}
		err := json.Unmarshal(stats, &statsdata)
		if err != nil {
			log.Println(err)
		}
		traffic, _ := os.ReadFile("/tmp/usertraffic")
		var trafficdata interface{}
		err = json.Unmarshal(traffic, &trafficdata)
		if err != nil {
			log.Println(err)
		}
		dashdata := viewdata{}
		dashdata.Systats = statsdata
		dashdata.Usertraffic = trafficdata
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/dashboard/templates/dash.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", dashdata); err != nil {
			log.Println("template error:", err)
		}
	})
	return Router
}
