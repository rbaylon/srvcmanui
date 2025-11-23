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

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		stats, _ := os.ReadFile("/tmp/mystats")
		var data interface{}
		err := json.Unmarshal(stats, &data)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/dashboard/templates/dash.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
			log.Println("template error:", err)
		}
	})
	return Router
}
