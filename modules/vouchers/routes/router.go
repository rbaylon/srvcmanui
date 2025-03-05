package voucherrouter

import (
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"

	vouchermodel "github.com/rbaylon/srvcman/modules/vouchers/model"
)

var letters = []byte("abcdefghijkmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ@*$%!+/123456789")

func New(db vouchermodel.Crud) *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		records, err := db.GetAll()
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/vouchers/templates/vouchers.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "darktheme", records)
		if err != nil {
			log.Println("template error:", err)
		}
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		voucherid, _ := strconv.Atoi(r.PathValue("id"))
		record, err := db.GetById(uint(voucherid))
		if err != nil {
			log.Println(err)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/darknav.html",
			pwd + "/modules/vouchers/templates/voucher.html",
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
			pwd + "/modules/vouchers/templates/voucher.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "darktheme", nil)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		count, err := strconv.Atoi(r.FormValue("count"))
		if err != nil {
			log.Println("return html error atoi issue")
		}

		hours, err1 := strconv.Atoi(r.FormValue("hours"))
		if err1 != nil {
			log.Println("return html error atoi issue", hours)
		}
		routerid, err2 := strconv.Atoi(r.FormValue("routerid"))
		if err2 != nil {
			log.Println("return html error atoi issue", routerid)
		}
		ifacetype := r.FormValue("ifacetype")
		code := []byte("00000000")
		maxlen := len(letters) - 1
		v := &vouchermodel.Voucher{}
		vcs := []vouchermodel.Voucher{}
		for i := 0; i < count; i++ {
			for j := 0; j < 8; j++ {
				c := letters[rand.IntN(maxlen)]
				code[j] = c
			}
			newcode := string(code)
			v.Value = newcode
			v.Type = ifacetype
			v.Hours = hours
			v.Status = "available"
			v.Downspeed = 3
			v.Upspeed = 2
			v.Burstspeed = 20
			v.Duration = 180000
			v.PfconfigID = uint(routerid)
			vcs = append(vcs, *v)
		}

		err = db.AddBulk(vcs)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, "/ui/vouchers", http.StatusSeeOther)
	})

	return Router
}
