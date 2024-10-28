package subrouter

import (
	"html/template"
	"net/http"
	"strconv"

	submodel "github.com/rbaylon/srvcman/modules/subs/model"
)

func New(db submodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		users, _ := db.GetAll()
		tmpl.Execute(w, users)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		subid, _ := strconv.Atoi(r.PathValue("id"))
		tmpl := template.Must(template.ParseFiles("sub.html"))
		users, _ := db.GetById(uint(subid))
		tmpl.Execute(w, users)
	})

	return Router
}
