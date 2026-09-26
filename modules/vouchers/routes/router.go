package voucherrouter

import (
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"time"

	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	vouchermodel "github.com/rbaylon/srvcman/modules/vouchers/model"
	"github.com/rbaylon/srvcman/utils/validate"
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

	// Every voucher, not just the page DataTables happens to be showing.
	// A static path, so it wins over "GET /{id}" below.
	Router.HandleFunc("GET /export", func(w http.ResponseWriter, r *http.Request) {
		var records []vouchermodel.Voucher
		err, status := client.Rester("vouchers", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		// A zero time means "not set yet", which the table shows as
		// 0001-01-01; an empty cell says it better.
		when := func(t time.Time) any {
			if t.IsZero() {
				return nil
			}
			return t.Format("2006-01-02 15:04:05")
		}
		rows := make([][]any, 0, len(records))
		for _, v := range records {
			rows = append(rows, []any{
				v.ID, v.Value, v.Type, v.Hours, v.Status, v.Downspeed, v.Upspeed,
				v.Burstspeed, v.Duration, v.PfconfigID,
				when(v.DateStarted), when(v.DateEnd), when(v.DateExpires),
			})
		}
		data, err := utils.BuildXLSX("Vouchers", []string{
			"ID", "Code", "Iface", "Hours", "Status", "DownSpeed", "UpSpeed",
			"BurstSpeed", "BurstDuration", "RouterID", "DateStart", "DateEnd", "DateExpire",
		}, rows)
		if err != nil {
			log.Println("vouchers export:", err)
			utils.RenderErrorPage(w, http.StatusInternalServerError, err.Error())
			return
		}
		utils.ServeXLSX(w, "vouchers-"+time.Now().Format("2006-01-02")+".xlsx", data)
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
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vouchers/templates/vouchernew.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", struct {
			Pfconfigs []pfconfigmodel.Pfconfig
		}{pfconfigs}); err != nil {
			log.Println(err)
		}
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		count, err := strconv.Atoi(r.FormValue("count"))
		if err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, "count must be a number")
			return
		}
		if err := validate.IntRange("count", count, 1, 1000); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}

		hours, err := strconv.Atoi(r.FormValue("hours"))
		if err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, "hours must be a number")
			return
		}
		if err := validate.Int("hours", hours); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		storetype := r.FormValue("storetype")
		if err := validate.Alnum("type", storetype, 100); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
			return
		}
		// Without a router, every generated voucher carries pfconfig_id 0:
		// rejected outright by the vouchers -> pfconfigs foreign key on
		// Postgres, and silently useless on SQLite (which doesn't enforce
		// it), since pf.conf generation only ever reaches a voucher through
		// that association.
		pfconfigID, err := strconv.Atoi(r.FormValue("PfconfigID"))
		if err != nil || pfconfigID < 1 {
			utils.RenderErrorPage(w, http.StatusBadRequest, "a router must be selected")
			return
		}

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
			v.PfconfigID = uint(pfconfigID)
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
