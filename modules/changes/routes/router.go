// Package changesrouter - read-only view of every config change record
// srvcman has logged (see security.audittrail in srvcman), regardless of
// whether it has been applied yet. The dashboard's own /ui/dashboard/changes
// only ever shows pending ones for the "Apply Changes" banner; this page is
// the full history, with no edit or delete actions.
package changesrouter

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	changemodel "github.com/rbaylon/srvcman/modules/changes/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

func New() *http.ServeMux {
	Router := http.NewServeMux()

	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		var records []changemodel.Change
		err, status := client.Rester("changes", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/changes/templates/changes.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", records); err != nil {
			log.Println("template error:", err)
		}
	})

	return Router
}
