const pathname = window.location.pathname;
function getPfconfig(id){
  fetch(`/ui/firewall/pfconfig/${id}`)
    .then(response => response.json())
    .then(data => {
      const ifaceSelect = document.getElementById('basic-form-interface');
      data.ifaces.forEach(iface => {
        // pppacN are the PPPoE servers' own interfaces; a server cannot listen on one.
        if (iface.type == "internal" && !iface.device.startsWith("pppac")){
            const option = document.createElement('option');
            option.value = iface.device;
            option.textContent = iface.name;
            ifaceSelect.appendChild(option);
        }
      });
    })
    .catch(error => console.error('Error fetching items:', error));
}