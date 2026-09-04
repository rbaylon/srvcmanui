package main

import (
	"log"
	"net/http"

	"github.com/rbaylon/srvcman/database"
	accesslogger "github.com/rbaylon/srvcmanui/middlewares/access/logger"
	accessvalidator "github.com/rbaylon/srvcmanui/middlewares/access/validator"
	collectionrouter "github.com/rbaylon/srvcmanui/modules/colls/routes"
	dashrouter "github.com/rbaylon/srvcmanui/modules/dashboard/routes"
	dhcpserverrouter "github.com/rbaylon/srvcmanui/modules/dhcpserver/routes"
	fwrouter "github.com/rbaylon/srvcmanui/modules/firewall/routes"
	gatewaymonitorrouter "github.com/rbaylon/srvcmanui/modules/gatewaymonitor/routes"
	grouprouter "github.com/rbaylon/srvcmanui/modules/groups/routes"
	instancerouter "github.com/rbaylon/srvcmanui/modules/instances/routes"
	pfifacerouter "github.com/rbaylon/srvcmanui/modules/pfifaces/routes"
	planrouter "github.com/rbaylon/srvcmanui/modules/plans/routes"
	policyrouter "github.com/rbaylon/srvcmanui/modules/policies/routes"
	pppoerouter "github.com/rbaylon/srvcmanui/modules/pppoes/routes"
	subrouter "github.com/rbaylon/srvcmanui/modules/subs/routes"
	userrouter "github.com/rbaylon/srvcmanui/modules/users/routes"
	vlanrouter "github.com/rbaylon/srvcmanui/modules/vlans/routes"
	voucherrouter "github.com/rbaylon/srvcmanui/modules/vouchers/routes"
)

func main() {
	port := database.GetEnvVariable("APP_PORT")
	subsrouter := subrouter.New()
	usersrouter := userrouter.New()
	collsrouter := collectionrouter.New()
	dhcpserversrouter := dhcpserverrouter.New()
	vouchersrouter := voucherrouter.New()
	plansrouter := planrouter.New()
	fwsrouter := fwrouter.New()
	gatewaymonitorsrouter := gatewaymonitorrouter.New()
	dashrouter := dashrouter.New()
	pppoerouter := pppoerouter.New()
	instacerouter := instancerouter.New()
	ifacesrouter := pfifacerouter.New()
	vlanrouter := vlanrouter.New()
	grouprouter := grouprouter.New()
	policyrouter := policyrouter.New()

	mainrouter := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))
	mainrouter.Handle("GET /ui/static/", http.StripPrefix("/ui/static/", fs))
	mainrouter.Handle("/ui/subs/", http.StripPrefix("/ui/subs", subsrouter))
	mainrouter.Handle("/ui/users/", http.StripPrefix("/ui/users", usersrouter))
	mainrouter.Handle("/ui/collections/", http.StripPrefix("/ui/collections", collsrouter))
	mainrouter.Handle("/ui/dhcpserver/", http.StripPrefix("/ui/dhcpserver", dhcpserversrouter))
	mainrouter.Handle("/ui/vouchers/", http.StripPrefix("/ui/vouchers", vouchersrouter))
	mainrouter.Handle("/ui/plans/", http.StripPrefix("/ui/plans", plansrouter))
	mainrouter.Handle("/ui/firewall/", http.StripPrefix("/ui/firewall", fwsrouter))
	mainrouter.Handle("/ui/gatewaymonitor/", http.StripPrefix("/ui/gatewaymonitor", gatewaymonitorsrouter))
	mainrouter.Handle("/ui/dashboard/", http.StripPrefix("/ui/dashboard", dashrouter))
	mainrouter.Handle("/ui/pppoes/", http.StripPrefix("/ui/pppoes", pppoerouter))
	mainrouter.Handle("/ui/instances/", http.StripPrefix("/ui/instances", instacerouter))
	mainrouter.Handle("/ui/pfifaces/", http.StripPrefix("/ui/pfifaces", ifacesrouter))
	mainrouter.Handle("/ui/vlan/", http.StripPrefix("/ui/vlan", vlanrouter))
	mainrouter.Handle("/ui/groups/", http.StripPrefix("/ui/groups", grouprouter))
	mainrouter.Handle("/ui/policies/", http.StripPrefix("/ui/policies", policyrouter))
	finalrouter := accessvalidator.Check(accesslogger.Log(mainrouter))

	log.Println("Go app listening on port", port)
	server := http.Server{Addr: ":" + port, Handler: finalrouter}
	server.ListenAndServe()

}
