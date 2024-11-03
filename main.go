package main

import (
	"log"
	"net/http"

	"github.com/rbaylon/srvcman/database"
	collectionmodel "github.com/rbaylon/srvcman/modules/collections/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	usermodel "github.com/rbaylon/srvcman/modules/users/model"
	accesslogger "github.com/rbaylon/srvcmanui/middlewares/access/logger"
	accessvalidator "github.com/rbaylon/srvcmanui/middlewares/access/validator"
	collectionrouter "github.com/rbaylon/srvcmanui/modules/colls/routes"
	subrouter "github.com/rbaylon/srvcmanui/modules/subs/routes"
	userrouter "github.com/rbaylon/srvcmanui/modules/users/routes"
)

func main() {
	port := database.GetEnvVariable("APP_PORT")
	db, err := database.ConnectToDB()
	if err != nil {
		log.Fatal(err)
	}

	subsrouter := subrouter.New(submodel.New(db))
	usersrouter := userrouter.New(usermodel.New(db))
	collsrouter := collectionrouter.New(collectionmodel.New(db))

	mainrouter := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	mainrouter.Handle("GET /ui/static/", http.StripPrefix("/ui/static/", fs))
	mainrouter.Handle("/ui/subs/", http.StripPrefix("/ui/subs", subsrouter))
	mainrouter.Handle("/ui/users/", http.StripPrefix("/ui/users", usersrouter))
	mainrouter.Handle("/ui/collections/", http.StripPrefix("/ui/collections", collsrouter))
	finalrouter := accessvalidator.Check(accesslogger.Log(mainrouter))

	log.Println("Go app listening on port", port)
	server := http.Server{Addr: ":" + port, Handler: finalrouter}
	server.ListenAndServe()

}
