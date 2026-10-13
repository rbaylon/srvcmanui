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
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
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

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		err, status := client.Rester("vouchers/"+r.PathValue("id"), "DELETE", nil, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/vouchers", http.StatusSeeOther)
	})

	// One voucher. New ones come from the bulk form below; this edits a single
	// existing voucher.
	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record vouchermodel.Voucher
		err, status := client.Rester("vouchers/"+r.PathValue("id"), "GET", &record, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		plans := []planmodel.Plan{}
		err, status = client.Rester("plans", "GET", &plans, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		// The states a voucher moves through, plus whatever it has now.
		statuses := []string{"available", "active", "used"}
		known := false
		for _, st := range statuses {
			known = known || st == record.Status
		}
		if !known && record.Status != "" {
			statuses = append(statuses, record.Status)
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/vouchers/templates/voucher.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		err = tmpl.ExecuteTemplate(w, "htmx", struct {
			Record   vouchermodel.Voucher
			Plans    []planmodel.Plan
			Statuses []string
		}{record, plans, statuses})
		if err != nil {
			log.Println(err)
		}
	})

	// Saves one voucher. Only what the form offers is taken from it; the rest of
	// the stored record (code, router, dates, address) is sent back unchanged.
	Router.HandleFunc("POST /update", func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("ID")
		var v vouchermodel.Voucher
		err, status := client.Rester("vouchers/"+id, "GET", &v, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
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
		st := r.FormValue("status")
		if err := validate.Alnum("status", st, 100); err != nil || st == "" {
			utils.RenderErrorPage(w, http.StatusBadRequest, "a status must be selected")
			return
		}
		if planName := r.FormValue("Plan"); planName != "" {
			if err := validate.Alnum("plan", planName, 100); err != nil {
				utils.RenderErrorPage(w, http.StatusBadRequest, err.Error())
				return
			}
			plans := []planmodel.Plan{}
			err, status := client.Rester("plans", "GET", &plans, r)
			if err != nil {
				fmt.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			found := false
			for _, p := range plans {
				if p.Plan == planName {
					v.Downspeed, v.Upspeed, v.Burstspeed = p.Downspeed, p.Upspeed, p.Downspeed*2
					found = true
					break
				}
			}
			if !found {
				utils.RenderErrorPage(w, http.StatusBadRequest, "unknown plan: "+planName)
				return
			}
		}
		v.Hours, v.Type, v.Status = hours, storetype, st
		err, status = client.Rester("vouchers/"+id, "PUT", &v, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/vouchers", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		plans := []planmodel.Plan{}
		err, status = client.Rester("plans", "GET", &plans, r)
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
			Plans     []planmodel.Plan
		}{pfconfigs, plans}); err != nil {
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

		// The voucher's Downspeed/Upspeed/Burstspeed used to be hardcoded
		// here; they now come from the selected Plan, the same source of
		// truth subs already uses, so a voucher's speed actually matches
		// the plan name printed on it. Burstspeed is fixed at 2x Downspeed.
		planName := r.FormValue("Plan")
		if err := validate.Alnum("plan", planName, 100); err != nil || planName == "" {
			utils.RenderErrorPage(w, http.StatusBadRequest, "a plan must be selected")
			return
		}
		plans := []planmodel.Plan{}
		err, status := client.Rester("plans", "GET", &plans, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		var plan *planmodel.Plan
		for i := range plans {
			if plans[i].Plan == planName {
				plan = &plans[i]
				break
			}
		}
		if plan == nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, "unknown plan: "+planName)
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
			v.Downspeed = plan.Downspeed
			v.Upspeed = plan.Upspeed
			v.Burstspeed = plan.Downspeed * 2
			v.Duration = 10000
			v.Type = storetype
			v.PfconfigID = uint(pfconfigID)
			vcs.Vouchers = append(vcs.Vouchers, *v)
		}

		err, status = client.Rester("vouchers/createBulk", "POST", &vcs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/vouchers", http.StatusSeeOther)
	})

	return Router
}
