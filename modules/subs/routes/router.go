package subrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	submodel "github.com/rbaylon/srvcman/modules/subs/model"
)

func New(db submodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		users, _ := db.GetAll()
		pwd, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/subs/templates/subs.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "darktheme", users)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		subid, _ := strconv.Atoi(r.PathValue("id"))
		tmpl := template.Must(template.ParseFiles("sub.html"))
		users, _ := db.GetById(uint(subid))
		tmpl.Execute(w, users)
	})

	return Router
}
