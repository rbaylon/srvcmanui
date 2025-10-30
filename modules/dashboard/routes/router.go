package dashrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/dashboard/templates/dash.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", nil); err != nil {
			log.Println("template error:", err)
		}
	})
	return Router
}
