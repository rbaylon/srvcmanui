package dashrouter

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	Changemodel "github.com/rbaylon/srvcman/modules/changes/model"
	pfifacemodel "github.com/rbaylon/srvcman/modules/pfifaces/model"
	planmodel "github.com/rbaylon/srvcman/modules/plans/model"
	submodel "github.com/rbaylon/srvcman/modules/subs/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func _sysinfo() map[string]string {
	info := map[string]string{}
	// 1. Host Information (Uptime, OS Version)
	hostInfo, err := host.Info()
	if err != nil {
		log.Fatalf("Failed to get host info: %v", err)
	}
	info["Host"] = hostInfo.Hostname
	info["OS"] = hostInfo.OS + " " + hostInfo.PlatformVersion
	info["Uptime"] = strconv.FormatUint(hostInfo.Uptime/60/60, 10) + " hours"

	// 2. CPU Information (Model Name & Speeds)
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info["CPU Model"] = cpuInfo[0].ModelName + " " + strconv.Itoa(int(cpuInfo[0].Cores)) + " " +
			strconv.FormatFloat(cpuInfo[0].Mhz, 'f', -1, 64) + " MHz"
	}

	// 3. Virtual Memory Statistics (RAM usage)
	vMem, err := mem.VirtualMemory()
	if err == nil {
		info["Memory"] = "Total: " + strconv.FormatUint(vMem.Total/1024/1024/1024, 10) + " GB " +
			"Available: " + strconv.FormatUint(vMem.Available/1024/1024/1024, 10) + " GB"
	}

	// 4. Disk Partition Usage
	diskUsage, err := disk.Usage("/")
	if err == nil {
		info["Disk /"] = strconv.FormatUint(diskUsage.Total/1024/1024, 10) + " MB"
	}
	return info
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3
	type subgw struct {
		Gw   string
		Subs int
	}
	type subplan struct {
		Plan string
		Subs int
	}
	type viewdata struct {
		SysInfo   map[string]string
		GwToSub   []subgw
		PlanTosub []subplan
	}
	Router.HandleFunc("GET /changes", func(w http.ResponseWriter, r *http.Request) {
		var records []Changemodel.Change
		err := client.Rester("changes/pending", "GET", &records, r)
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	})

	Router.HandleFunc("GET /changes/apply", func(w http.ResponseWriter, r *http.Request) {
		var errors []Changemodel.Change
		err := client.Rester("changes/pending", "PUT", &errors, r)
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errors)
	})
	Router.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		subdata := viewdata{}
		ifs := []pfifacemodel.Pfiface{}
		err := client.Rester("pfiface", "GET", &ifs, r)
		if err != nil {
			log.Println(err)
		}
		subs := []submodel.Sub{}
		err = client.Rester("subs", "GET", &subs, r)
		if err != nil {
			log.Println(err)
		}
		plans := []planmodel.Plan{}
		err = client.Rester("plans", "GET", &plans, r)
		if err != nil {
			log.Println(err)
		}
		sgws := []subgw{}
		for _, v := range ifs {
			sgw := subgw{}
			sgw.Gw = v.Gateway
			for _, s := range subs {
				if s.Gateway == v.Gateway {
					sgw.Subs += 1
				}
			}
			sgws = append(sgws, sgw)
		}
		splans := []subplan{}
		for _, v := range plans {
			splan := subplan{}
			splan.Plan = v.Plan
			for _, s := range subs {
				if s.Plan == v.Plan {
					splan.Subs += 1
				}
			}
			splans = append(splans, splan)
		}
		subdata.GwToSub = sgws
		subdata.PlanTosub = splans
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subdata)
	})
	Router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		info := _sysinfo()
		dashdata := viewdata{}
		dashdata.SysInfo = info
		pwd, _ := os.Getwd()
		files := []string{
			pwd + "/layouts/newbase.html",
			pwd + "/modules/dashboard/templates/dash.html",
		}
		tmpl := template.Must(template.ParseFiles(files...))
		if err := tmpl.ExecuteTemplate(w, "htmx", dashdata); err != nil {
			log.Println("template error:", err)
		}
	})
	return Router
}
