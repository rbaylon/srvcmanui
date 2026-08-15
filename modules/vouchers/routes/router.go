package voucherrouter

import (
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"

	vouchermodel "github.com/rbaylon/srvcman/modules/vouchers/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

var letters = []byte("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []vouchermodel.Voucher
		err, status := client.Rester("vouchers", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vouchers/templates/vouchers.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", records)
		if err != nil {
			log.Println("template error:", err)
		}
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record vouchermodel.Voucher
		err, status := client.Rester("vouchers/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vouchers/templates/voucher.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", record)
		if err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vouchers/templates/voucher.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err := tmpl.ExecuteTemplate(w, "htmx", nil)
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
		storetype := r.FormValue("storetype")

		code := []byte("00000000")
		maxlen := len(letters) - 1
		v := &vouchermodel.Voucher{}
		vcs := vouchermodel.Vouchers{}
		for i := 0; i < count; i++ {
			for j := 0; j < 8; j++ {
				c := letters[rand.IntN(maxlen)]
				code[j] = c
			}
			newcode := string(code)
			v.Value = newcode
			v.Hours = hours
			v.Status = "available"
			v.Downspeed = 3
			v.Upspeed = 2
			v.Burstspeed = 5
			v.Duration = 60000
			v.Type = storetype
			vcs.Vouchers = append(vcs.Vouchers, *v)
		}

		err, status := client.Rester("vouchers/createBulk", "POST", &vcs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/vouchers", http.StatusSeeOther)
	})

	return Router
}
