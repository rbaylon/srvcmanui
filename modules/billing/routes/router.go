// Package billingrouter - srvcmanui page promoting the hosted billing
// platform at www.specz.net/billing. Static content: it has no srvcman
// backend and makes no API calls.
package billingrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/billing/templates/billing.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", nil); err != nil {
			log.Println(err)
		}
	})

	return Router
}
