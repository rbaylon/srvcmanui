package subrouter

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	submodel "github.com/rbaylon/srvcman/modules/subs/model"
)

// temporary until I use reflection
func SetData(s *submodel.Sub, r *http.Request) {
	t, _ := time.Parse(time.RFC3339, r.FormValue("CreatedAt"))
	s.CreatedAt = t
	s.FirstName = r.FormValue("first_name")
	s.LastName = r.FormValue("last_name")
	s.FramedIp = r.FormValue("framed_ip")
	s.Type = r.FormValue("type")
	s.Status = r.FormValue("status")
	s.Mac = r.FormValue("mac")
	s.Loc = r.FormValue("loc")
	ds, _ := strconv.Atoi(r.FormValue("downspeed"))
	s.Downspeed = ds
	us, _ := strconv.Atoi(r.FormValue("upspeed"))
	s.Upspeed = us
	bs, _ := strconv.Atoi(r.FormValue("burstspeed"))
	s.Burstspeed = bs
	d, _ := strconv.Atoi(r.FormValue("duration"))
	s.Duration = d
	s.Gateway = r.FormValue("gateway")
	pr, _ := strconv.Atoi(r.FormValue("priority"))
	s.Priority = pr
	pfc, _ := strconv.Atoi(r.FormValue("pfconfig_id"))
	s.PfconfigID = uint(pfc)
	du, _ := strconv.Atoi(r.FormValue("due_day"))
	s.DueDay = du
	od, _ := strconv.ParseBool(r.FormValue("over_due"))
	s.OverDue = od
	lt, _ := strconv.Atoi(r.FormValue("later_count"))
	s.LaterCount = lt
}

func New(db submodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records, err := db.GetAll()
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/subs/templates/subs.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		tmpl.ExecuteTemplate(w, "darktheme", records)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		subid, _ := strconv.Atoi(r.PathValue("id"))
		record, err := db.GetById(uint(subid))
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/subs/templates/sub.html",
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
			pwd + "/modules/subs/templates/sub.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "darktheme", nil)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		s := &submodel.Sub{}
		SetData(s, r)
		id, _ := strconv.Atoi(r.FormValue("id"))
		if id > 0 {
			s.ID = uint(id)
			err := db.Update(s)
			if err != nil {
				log.Println(err)
			}
			http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
			return
		}
		err := db.Add(s)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/subs", http.StatusSeeOther)
	})

	return Router
}
