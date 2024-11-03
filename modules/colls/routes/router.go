package collectionrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	collectionmodel "github.com/rbaylon/srvcman/modules/collections/model"
)

// temporary until I use reflection
func SetData(c *collectionmodel.Collection, r *http.Request) {
	c.Year = r.FormValue("year")
	jan, _ := strconv.Atoi(r.FormValue("jan"))
	c.Jan = jan
	feb, _ := strconv.Atoi(r.FormValue("feb"))
	c.Feb = feb
	mar, _ := strconv.Atoi(r.FormValue("mar"))
	c.Mar = mar
	apr, _ := strconv.Atoi(r.FormValue("apr"))
	c.Apr = apr
	may, _ := strconv.Atoi(r.FormValue("may"))
	c.May = may
	jun, _ := strconv.Atoi(r.FormValue("jun"))
	c.Jun = jun
	jul, _ := strconv.Atoi(r.FormValue("jul"))
	c.Jul = jul
	aug, _ := strconv.Atoi(r.FormValue("aug"))
	c.Aug = aug
	sep, _ := strconv.Atoi(r.FormValue("sep"))
	c.Sep = sep
	oct, _ := strconv.Atoi(r.FormValue("oct"))
	c.Oct = oct
	nov, _ := strconv.Atoi(r.FormValue("nov"))
	c.Nov = nov
	dec, _ := strconv.Atoi(r.FormValue("dec"))
	c.Dec = dec
	sid, _ := strconv.Atoi(r.FormValue("sub_id"))
	c.SubID = uint(sid)
}

func New(db collectionmodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /{year}", func(w http.ResponseWriter, r *http.Request) {
		y := r.PathValue("year")
		records, err := db.GetByYear(y)
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/colls/templates/colls.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "darktheme", records)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /{year}/{id}", func(w http.ResponseWriter, r *http.Request) {
		collectionid, _ := strconv.Atoi(r.PathValue("id"))
		record, err := db.GetById(uint(collectionid))
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/colls/templates/coll.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "darktheme", record)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/colls/templates/coll.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "darktheme", nil)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &collectionmodel.Collection{}
		SetData(s, r)
		id, _ := strconv.Atoi(r.FormValue("id"))
		if id > 0 {
			s.ID = uint(id)
			err := db.Update(s)
			if err != nil {
				log.Println(err)
			}
			http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
			return
		}
		err := db.Add(s)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/collections/"+s.Year, http.StatusSeeOther)
	})

	return Router
}
