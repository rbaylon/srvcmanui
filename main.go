package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/rbaylon/srvcman/database"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
)

type Film struct {
	Title    string
	Director string
}

func main() {
	fmt.Println("Go app...")

	db, err := database.ConnectToDB()
	if err != nil {
		log.Fatal(err)
	}

	subStore := submodel.New(db)

	// handler function #1 - returns the index.html template, with film data
	h1 := func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		users, _ := subStore.GetAll()
		tmpl.Execute(w, users)
	}

	// handler function #2 - returns the template block with the newly added film, as an HTMX response
	h2 := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		title := r.PostFormValue("title")
		director := r.PostFormValue("director")
		// htmlStr := fmt.Sprintf("<li class='list-group-item bg-primary text-white'>%s - %s</li>", title, director)
		// tmpl, _ := template.New("t").Parse(htmlStr)
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.ExecuteTemplate(w, "film-list-element", Film{Title: title, Director: director})
	}

	// define handlers
	http.HandleFunc("/ui/", h1)
	http.HandleFunc("/ui/add-film/", h2)

	log.Fatal(http.ListenAndServe(":8000", nil))

}
