package main

import (
	"log"
	"net/http"

	"github.com/rbaylon/srvcman/database"
	accesslogger "github.com/rbaylon/srvcmanui/middlewares/access/logger"
	accessvalidator "github.com/rbaylon/srvcmanui/middlewares/access/validator"
	acmerouter "github.com/rbaylon/srvcmanui/modules/acme/routes"
	billingrouter "github.com/rbaylon/srvcmanui/modules/billing/routes"
	billportalrouter "github.com/rbaylon/srvcmanui/modules/billportal/routes"
	captiveportalrouter "github.com/rbaylon/srvcmanui/modules/captiveportal/routes"
	changesrouter "github.com/rbaylon/srvcmanui/modules/changes/routes"
	collectionrouter "github.com/rbaylon/srvcmanui/modules/colls/routes"
	dashrouter "github.com/rbaylon/srvcmanui/modules/dashboard/routes"
	dhcpserverrouter "github.com/rbaylon/srvcmanui/modules/dhcpserver/routes"
	diagnosticsrouter "github.com/rbaylon/srvcmanui/modules/diagnostics/routes"
	fdmonitorrouter "github.com/rbaylon/srvcmanui/modules/fdmonitor/routes"
	fwrouter "github.com/rbaylon/srvcmanui/modules/firewall/routes"
	gatewaymonitorrouter "github.com/rbaylon/srvcmanui/modules/gatewaymonitor/routes"
	grouprouter "github.com/rbaylon/srvcmanui/modules/groups/routes"
	instancerouter "github.com/rbaylon/srvcmanui/modules/instances/routes"
	netblocksrouter "github.com/rbaylon/srvcmanui/modules/netblocks/routes"
	pfifacerouter "github.com/rbaylon/srvcmanui/modules/pfifaces/routes"
	powerrouter "github.com/rbaylon/srvcmanui/modules/power/routes"
	pflowrouter "github.com/rbaylon/srvcmanui/modules/pflows/routes"
	planrouter "github.com/rbaylon/srvcmanui/modules/plans/routes"
	policyrouter "github.com/rbaylon/srvcmanui/modules/policies/routes"
	pppoerouter "github.com/rbaylon/srvcmanui/modules/pppoes/routes"
	snmpserverrouter "github.com/rbaylon/srvcmanui/modules/snmpserver/routes"
	staticroutesrouter "github.com/rbaylon/srvcmanui/modules/staticroutes/routes"
	subrouter "github.com/rbaylon/srvcmanui/modules/subs/routes"
	tunnelrouter "github.com/rbaylon/srvcmanui/modules/tunnels/routes"
	unbounddnsrouter "github.com/rbaylon/srvcmanui/modules/unbounddns/routes"
	updatesrouter "github.com/rbaylon/srvcmanui/modules/updates/routes"
	userrouter "github.com/rbaylon/srvcmanui/modules/users/routes"
	vlanrouter "github.com/rbaylon/srvcmanui/modules/vlans/routes"
	voucherrouter "github.com/rbaylon/srvcmanui/modules/vouchers/routes"
	"github.com/rbaylon/srvcmanui/version"
)

func main() {
	log.Println("Specz ISP UI version", version.Version)
	port := database.GetEnvVariable("APP_PORT")
	acmesrouter := acmerouter.New()
	captiveportalsrouter := captiveportalrouter.New()
	billportalsrouter := billportalrouter.New()
	billingsrouter := billingrouter.New()
	changessrouter := changesrouter.New()
	tunnelsrouter := tunnelrouter.New()
	fdmonitorsrouter := fdmonitorrouter.New()
	pflowsrouter := pflowrouter.New()
	snmpserversrouter := snmpserverrouter.New()
	subsrouter := subrouter.New()
	usersrouter := userrouter.New()
	collsrouter := collectionrouter.New()
	dhcpserversrouter := dhcpserverrouter.New()
	unbounddnssrouter := unbounddnsrouter.New()
	vouchersrouter := voucherrouter.New()
	updatessrouter := updatesrouter.New()
	netblockssrouter := netblocksrouter.New()
	powersrouter := powerrouter.New()
	plansrouter := planrouter.New()
	fwsrouter := fwrouter.New()
	gatewaymonitorsrouter := gatewaymonitorrouter.New()
	staticroutessrouter := staticroutesrouter.New()
	diagnosticsrouter := diagnosticsrouter.New()
	dashrouter := dashrouter.New()
	pppoerouter := pppoerouter.New()
	instacerouter := instancerouter.New()
	ifacesrouter := pfifacerouter.New()
	vlanrouter := vlanrouter.New()
	grouprouter := grouprouter.New()
	policyrouter := policyrouter.New()

	mainrouter := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	mainrouter.HandleFunc("GET /ui/static/version.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(version.Version))
	})
	mainrouter.Handle("GET /ui/static/", http.StripPrefix("/ui/static/", fs))
	mainrouter.Handle("/ui/subs/", http.StripPrefix("/ui/subs", subsrouter))
	mainrouter.Handle("/ui/users/", http.StripPrefix("/ui/users", usersrouter))
	mainrouter.Handle("/ui/collections/", http.StripPrefix("/ui/collections", collsrouter))
	mainrouter.Handle("/ui/dhcpserver/", http.StripPrefix("/ui/dhcpserver", dhcpserversrouter))
	mainrouter.Handle("/ui/unbounddns/", http.StripPrefix("/ui/unbounddns", unbounddnssrouter))
	mainrouter.Handle("/ui/power/", http.StripPrefix("/ui/power", powersrouter))
	mainrouter.Handle("/ui/netblocks/", http.StripPrefix("/ui/netblocks", netblockssrouter))
	mainrouter.Handle("/ui/updates/", http.StripPrefix("/ui/updates", updatessrouter))
	mainrouter.Handle("/ui/vouchers/", http.StripPrefix("/ui/vouchers", vouchersrouter))
	mainrouter.Handle("/ui/plans/", http.StripPrefix("/ui/plans", plansrouter))
	mainrouter.Handle("/ui/firewall/", http.StripPrefix("/ui/firewall", fwsrouter))
	mainrouter.Handle("/ui/gatewaymonitor/", http.StripPrefix("/ui/gatewaymonitor", gatewaymonitorsrouter))
	mainrouter.Handle("/ui/staticroutes/", http.StripPrefix("/ui/staticroutes", staticroutessrouter))
	mainrouter.Handle("/ui/diagnostics/", http.StripPrefix("/ui/diagnostics", diagnosticsrouter))
	mainrouter.Handle("/ui/dashboard/", http.StripPrefix("/ui/dashboard", dashrouter))
	mainrouter.Handle("/ui/pppoes/", http.StripPrefix("/ui/pppoes", pppoerouter))
	mainrouter.Handle("/ui/instances/", http.StripPrefix("/ui/instances", instacerouter))
	mainrouter.Handle("/ui/pfifaces/", http.StripPrefix("/ui/pfifaces", ifacesrouter))
	mainrouter.Handle("/ui/vlan/", http.StripPrefix("/ui/vlan", vlanrouter))
	mainrouter.Handle("/ui/groups/", http.StripPrefix("/ui/groups", grouprouter))
	mainrouter.Handle("/ui/policies/", http.StripPrefix("/ui/policies", policyrouter))
	mainrouter.Handle("/ui/acme/", http.StripPrefix("/ui/acme", acmesrouter))
	mainrouter.Handle("/ui/captiveportal/", http.StripPrefix("/ui/captiveportal", captiveportalsrouter))
	mainrouter.Handle("/ui/billing/", http.StripPrefix("/ui/billing", billingsrouter))
	mainrouter.Handle("/ui/billportal/", http.StripPrefix("/ui/billportal", billportalsrouter))
	mainrouter.Handle("/ui/tunnels/", http.StripPrefix("/ui/tunnels", tunnelsrouter))
	mainrouter.Handle("/ui/fdmonitor/", http.StripPrefix("/ui/fdmonitor", fdmonitorsrouter))
	mainrouter.Handle("/ui/pflows/", http.StripPrefix("/ui/pflows", pflowsrouter))
	mainrouter.Handle("/ui/snmpserver/", http.StripPrefix("/ui/snmpserver", snmpserversrouter))
	mainrouter.Handle("/ui/changes/", http.StripPrefix("/ui/changes", changessrouter))
	finalrouter := accessvalidator.Check(accesslogger.Log(mainrouter))

	log.Println("Go app listening on port", port)
	server := http.Server{Addr: ":" + port, Handler: finalrouter}
	server.ListenAndServe()

}
