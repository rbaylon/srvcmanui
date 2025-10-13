function getPfconfig(id){
  fetch(`/ui/firewall/pfconfig/${id}`)
    .then(response => response.json())
    .then(data => {
      const pathname = window.location.pathname;
      console.log(pathname);
      const ifaceSelect = document.getElementById('basic-form-interface');
      if (pathname != "/ui/subs/create") {
        ifaceSelect.innerHTML = '<option value="any">any</option>'; // Clear previous options
      }

      const gwSelect = document.getElementById('basic-form-gateway');
      gwSelect.innerHTML = '<option value="">Default</option>'; // Clear previous options

      data.ifaces.forEach(iface => {
        if (pathname == "/ui/subs/create" && iface.type != "external") {
          const option = document.createElement('option');
          option.value = iface.device;
          option.textContent = iface.name;
          ifaceSelect.appendChild(option);
        } else {
          const option = document.createElement('option');
          option.value = iface.device;
          option.textContent = iface.name;
          ifaceSelect.appendChild(option);
        }
        if (iface.type == "external"){
            const optiongw = document.createElement('option');
            optiongw.value = iface.gateway;
            if (iface.default) {
                optiongw.textContent = `${iface.gateway} ${iface.name} Default`;
            } else {
                optiongw.textContent = `${iface.gateway} ${iface.name}`;
            }
            gwSelect.appendChild(optiongw);
        }
      });
    })
    .catch(error => console.error('Error fetching items:', error));
}

document.getElementById('basic-form-pfconfig').addEventListener('change', function() {
  const selectedCategory = this.value;
  getPfconfig(encodeURIComponent(selectedCategory));
});

document.addEventListener('DOMContentLoaded', function() {
    const myField = document.getElementById('basic-form-pfconfig');
    getPfconfig(encodeURIComponent(myField.value));
});
