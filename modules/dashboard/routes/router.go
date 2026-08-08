package dashrouter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	Changemodel "github.com/rbaylon/srvcman/modules/changes/model"
	"github.com/rbaylon/srvcmanui/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func _sysinfo() []string {
	info := []string{}
	// 1. Host Information (Uptime, OS Version)
	hostInfo, err := host.Info()
	if err != nil {
		log.Fatalf("Failed to get host info: %v", err)
	}
	info = append(info, fmt.Sprintf("Host: %s, OS: %s %s, Uptime: %d sec\n",
		hostInfo.Hostname, hostInfo.OS, hostInfo.PlatformVersion, hostInfo.Uptime))

	// 2. CPU Information (Model Name & Speeds)
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info = append(info, fmt.Sprintf("CPU Model: %s, Cores: %d, Speed: %.2f MHz\n",
			cpuInfo[0].ModelName, cpuInfo[0].Cores, cpuInfo[0].Mhz))
	}

	// 3. Virtual Memory Statistics (RAM usage)
	vMem, err := mem.VirtualMemory()
	if err == nil {
		info = append(info, fmt.Sprintf("Total Memory: %d GB, Available: %d GB, Used Percent: %.2f%%\n",
			vMem.Total/1024/1024/1024, vMem.Available/1024/1024/1024, vMem.UsedPercent))
	}

	// 4. Disk Partition Usage
	diskUsage, err := disk.Usage("/")
	if err == nil {
		info = append(info, fmt.Sprintf("Disk Space: Total: %d GB, Free: %d GB, Used Percent: %.2f%%\n",
			diskUsage.Total/1024/1024/1024, diskUsage.Free/1024/1024/1024, diskUsage.UsedPercent))
	}
	return info
}

func New() *http.ServeMux {
	Router := http.NewServeMux() // 3

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

	type viewdata struct {
		SysInfo []string
	}

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
