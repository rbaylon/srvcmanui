// Package fdmonitorrouter - srvcmanui UI for the fdmonitor module.
//
// Read-only, like the diagnostics UI: one GET against srvcman renders the
// whole page. A failure reaching arkgated comes back in the payload's
// "error" rather than as an HTTP error, so the page shows what it did get
// alongside the problem instead of a full error page.
package fdmonitorrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	fdmonitorroutes "github.com/rbaylon/srvcman/modules/fdmonitor/routes"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

// viewdata adds the presentation-only bits the template needs on top of
// srvcman's reading.
type viewdata struct {
	fdmonitorroutes.Status
	// Level is the bootstrap contextual class for the usage bar, stepped
	// off Percent: green under 70, amber from 70, red from 90.
	Level string
}

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var status fdmonitorroutes.Status
		err, code := client.Rester("fdmonitor", "GET", &status, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, code, err.Error())
			return
		}
		data := viewdata{Status: status, Level: "success"}
		switch {
		case status.Percent >= 90:
			data.Level = "danger"
		case status.Percent >= 70:
			data.Level = "warning"
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/fdmonitor/templates/fdmonitor.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", data); err != nil {
			log.Println(err)
		}
	})

	return Router
}
