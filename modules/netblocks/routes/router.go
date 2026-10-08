// Package netblocksrouter - srvcmanui UI for the speedtest netblocks module.
//
// The standard list/create/edit/delete shape. Each record is one network
// block in CIDR notation; srvcman also keeps it in the gateway's
// fast.com.blocks file, which is why a failed add or delete can come back as
// an error even though the form itself was valid.
//
// The record struct is local rather than imported from srvcman's model: it
// is just the two JSON fields the API exchanges, and keeping it here means
// this UI builds without waiting on a srvcman release that carries the
// module.
package netblocksrouter

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type netblock struct {
	ID   uint   `json:"ID,omitempty"`
	Cidr string `json:"cidr"`
}

func page(w http.ResponseWriter, name string, data any) {
	pwd, _ := os.Getwd()
	files := []string{
		pwd + "/layouts/newbase.html",
		pwd + "/modules/netblocks/templates/" + name,
	}
	tmpl := template.Must(template.ParseFiles(files...))
	if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
		log.Println(err)
	}
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []netblock
		if err, status := client.Rester("netblocks", "GET", &records, r); err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		page(w, "netblocks.html", records)
	})

	Router.HandleFunc("GET /create", func(w http.ResponseWriter, r *http.Request) {
		page(w, "netblocknew.html", nil)
	})

	Router.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var record netblock
		if err, status := client.Rester("netblocks/"+r.PathValue("id"), "GET", &record, r); err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		page(w, "netblock.html", record)
	})

	Router.HandleFunc("POST /addorupdate", func(w http.ResponseWriter, r *http.Request) {
		cidr := strings.TrimSpace(r.FormValue("Cidr"))
		// srvcman checks this too; checking here as well is what lets the
		// user be told what is wrong, since the shared client only reports
		// the status code of a refused request.
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			utils.RenderErrorPage(w, http.StatusBadRequest, "Enter a network in CIDR notation, e.g. 192.168.1.0/24")
			return
		}
		s := netblock{Cidr: cidr}
		if id, _ := strconv.ParseUint(r.FormValue("ID"), 10, 32); id > 0 {
			if err, status := client.Rester("netblocks/"+strconv.FormatUint(id, 10), "PUT", &s, r); err != nil {
				log.Println(err)
				utils.RenderErrorPage(w, status, err.Error())
				return
			}
			http.Redirect(w, r, "/ui/netblocks", http.StatusSeeOther)
			return
		}
		if err, status := client.Rester("netblocks/create", "POST", &s, r); err != nil {
			log.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/netblocks", http.StatusSeeOther)
	})

	Router.HandleFunc("GET /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err, status := client.Rester("netblocks/"+r.PathValue("id"), "DELETE", nil, r); err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/netblocks", http.StatusSeeOther)
	})

	return Router
}
