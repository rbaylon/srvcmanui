package dashrouter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	Changemodel "github.com/rbaylon/srvcman/modules/changes/model"
	gatewaymonitormodel "github.com/rbaylon/srvcman/modules/gatewaymonitor/model"
	pfconfigmodel "github.com/rbaylon/srvcman/modules/pfconfig/model"
	pfifacemodel "github.com/rbaylon/srvcman/modules/pfifaces/model"
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/rbaylon/srvcmanui/utils"
)

type subgw struct {
	Gws  []string
	Subs []int
}

type subplan struct {
	Plans []string
	Subs  []int
}

type chartData struct {
	GwToSub   subgw
	PlanToSub subplan
}

// buildChartData computes the gateway->subscriber and plan->subscriber
// distributions for one set of ifaces/subs/plans - shared by the grouped
// dashboard (one call per pfconfig) and the standalone /stats endpoint
// (one call across everything).
func buildChartData(ifs []pfifacemodel.Pfiface, subs []submodel.Sub, plans []planmodel.Plan) chartData {
	gws := []string{}
	sgws := []int{}
	for _, v := range ifs {
		if v.Type == "external" {
			gws = append(gws, v.Name)
			scount := 0
			for _, s := range subs {
				if s.Gateway == v.Gateway && s.Status == "active" {
					scount += 1
				}
			}
			sgws = append(sgws, scount)
		}
	}
	defaultCount := 0
	for _, s := range subs {
		if s.Gateway == "" && s.Status == "active" {
			defaultCount += 1
		}
	}
	gws = append(gws, "Default")
	sgws = append(sgws, defaultCount)

	sPlans := []string{}
	splans := []int{}
	for _, v := range plans {
		sPlans = append(sPlans, v.Plan)
		scount := 0
		for _, s := range subs {
			if s.Plan == v.Plan && s.Status == "active" {
				scount += 1
			}
		}
		splans = append(splans, scount)
	}
	return chartData{
		GwToSub:   subgw{Gws: gws, Subs: sgws},
		PlanToSub: subplan{Plans: sPlans, Subs: splans},
	}
}

// routerGroup is one pfconfig's slice of the dashboard - a pfconfig
// represents a single OpenBSD router instance, so subs/pfifaces/gateway
// monitors are scoped to it via their PfconfigID, and SysInfo is fetched
// from that router's own arkgated via the pfconfig/{id}/sysinfo endpoint
// (see pfconfigmodel.GetSystemInfo in srvcman) rather than gathered
// in-process, since srvcmanui may not run on the router it's displaying
// data for.
type routerGroup struct {
	Pfconfig        pfconfigmodel.Pfconfig
	Unassigned      bool
	SysInfo         map[string]string
	SysInfoError    string
	Pfifaces        []pfifacemodel.Pfiface
	GatewayMonitors []gatewaymonitormodel.GatewayMonitor
	ChartDataJSON   template.JS
}

// dashPageData is what dash.html renders: the full list of pfconfigs (to
// populate the router-select dropdown), whether an "Unassigned" bucket has
// anything in it (whether to show that option), which one is currently
// selected, and that one selection's data - only one router's worth of
// data is ever fetched/rendered per request, chosen via the dropdown
// instead of stacking every router's section on one page.
type dashPageData struct {
	Pfconfigs     []pfconfigmodel.Pfconfig
	HasUnassigned bool
	SelectedID    uint
	Group         routerGroup
}

type trafficStat struct {
	Device string `json:"device"`
	Ibytes uint64 `json:"ibytes"`
	Obytes uint64 `json:"obytes"`
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

	Router.HandleFunc("GET /changes", func(w http.ResponseWriter, r *http.Request) {
		var records []Changemodel.Change
		err, status := client.Rester("changes/pending", "GET", &records, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	})

	Router.HandleFunc("GET /changes/apply", func(w http.ResponseWriter, r *http.Request) {
		var errors []Changemodel.Change
		err, status := client.Rester("changes/pending", "PUT", &errors, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errors)
	})
	Router.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		ifs := []pfifacemodel.Pfiface{}
		err, status := client.Rester("pfiface", "GET", &ifs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		subs := []submodel.Sub{}
		err, status = client.Rester("subs", "GET", &subs, r)
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildChartData(ifs, subs, plans))
	})
	Router.HandleFunc("GET /traffic", func(w http.ResponseWriter, r *http.Request) {
		var stats []trafficStat
		err, status := client.Rester("pfiface/traffic", "GET", &stats, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})
	Router.HandleFunc("GET /gatewaystatus", func(w http.ResponseWriter, r *http.Request) {
		gateways := []gatewaymonitormodel.GatewayMonitor{}
		// gatewaymonitor is an optional module - a fetch failure (e.g. not
		// enabled on this deployment) shouldn't break the poll, just report
		// nothing.
		if err, _ := client.Rester("gatewaymonitor", "GET", &gateways, r); err != nil {
			log.Println("gatewaystatus:", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gateways)
	})
	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		pfconfigs := []pfconfigmodel.Pfconfig{}
		err, status := client.Rester("pfconfig", "GET", &pfconfigs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		ifs := []pfifacemodel.Pfiface{}
		err, status = client.Rester("pfiface", "GET", &ifs, r)
		if err != nil {
			fmt.Println(err)
			utils.RenderErrorPage(w, status, err.Error())
			return
		}
		subs := []submodel.Sub{}
		err, status = client.Rester("subs", "GET", &subs, r)
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
		gateways := []gatewaymonitormodel.GatewayMonitor{}
		// gatewaymonitor is an optional module - don't fail the whole
		// dashboard if it isn't enabled on this deployment.
		if gwerr, _ := client.Rester("gatewaymonitor", "GET", &gateways, r); gwerr != nil {
			log.Println("dashboard: gatewaymonitor fetch failed:", gwerr)
		}

		hasUnassigned := false
		for _, v := range ifs {
			if v.PfconfigID == 0 {
				hasUnassigned = true
				break
			}
		}
		if !hasUnassigned {
			for _, s := range subs {
				if s.PfconfigID == 0 {
					hasUnassigned = true
					break
				}
			}
		}
		if !hasUnassigned {
			for _, p := range plans {
				if p.PfconfigID == 0 {
					hasUnassigned = true
					break
				}
			}
		}
		if !hasUnassigned {
			for _, g := range gateways {
				if g.PfconfigID == 0 {
					hasUnassigned = true
					break
				}
			}
		}

		// Default to the first pfconfig (or Unassigned, if there are none)
		// unless the dropdown picked a specific one via ?pfconfig_id=.
		var selectedID uint
		if len(pfconfigs) > 0 {
			selectedID = pfconfigs[0].ID
		}
		if idStr := r.URL.Query().Get("pfconfig_id"); idStr != "" {
			if parsed, perr := strconv.ParseUint(idStr, 10, 64); perr == nil {
				selectedID = uint(parsed)
			}
		}

		group := buildRouterGroup(selectedID, pfconfigs, ifs, subs, plans, gateways, r)

		pagedata := dashPageData{
			Pfconfigs:     pfconfigs,
			HasUnassigned: hasUnassigned,
			SelectedID:    selectedID,
			Group:         group,
		}

		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/dashboard/templates/dash.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", pagedata); err != nil {
			log.Println("template error:", err)
		}
	})
	return Router
}

// buildRouterGroup filters ifs/subs/plans/gateways down to the ones
// belonging to pfconfig id (0 meaning the "Unassigned" bucket - records
// that predate pfconfig grouping or were never assigned one) and, for a
// real pfconfig, fetches that router's live system info.
func buildRouterGroup(id uint, pfconfigs []pfconfigmodel.Pfconfig, ifs []pfifacemodel.Pfiface, subs []submodel.Sub, plans []planmodel.Plan, gateways []gatewaymonitormodel.GatewayMonitor, r *http.Request) routerGroup {
	group := routerGroup{}
	if id == 0 {
		group.Unassigned = true
	} else {
		for _, pfc := range pfconfigs {
			if pfc.ID == id {
				group.Pfconfig = pfc
				break
			}
		}
		var sysinfo struct {
			SystemInfo map[string]string `json:"system_info"`
			Error      string            `json:"error,omitempty"`
		}
		if sierr, _ := client.Rester(fmt.Sprintf("pfconfig/%d/sysinfo", id), "GET", &sysinfo, r); sierr != nil {
			log.Println("dashboard: sysinfo fetch failed for pfconfig", id, ":", sierr)
		}
		group.SysInfo = sysinfo.SystemInfo
		group.SysInfoError = sysinfo.Error
	}

	var groupSubs []submodel.Sub
	var groupPlans []planmodel.Plan
	for _, v := range ifs {
		if v.PfconfigID == id {
			group.Pfifaces = append(group.Pfifaces, v)
		}
	}
	for _, s := range subs {
		if s.PfconfigID == id {
			groupSubs = append(groupSubs, s)
		}
	}
	for _, p := range plans {
		if p.PfconfigID == id {
			groupPlans = append(groupPlans, p)
		}
	}
	for _, g := range gateways {
		if g.PfconfigID == id {
			group.GatewayMonitors = append(group.GatewayMonitors, g)
		}
	}

	cdJSON, jerr := json.Marshal(buildChartData(group.Pfifaces, groupSubs, groupPlans))
	if jerr != nil {
		log.Println("dashboard: chart data marshal failed:", jerr)
	} else {
		group.ChartDataJSON = template.JS(cdJSON)
	}
	return group
}
