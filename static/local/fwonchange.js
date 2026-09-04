const pathname = window.location.pathname;
function getPfconfig(id){
  fetch(`/ui/firewall/pfconfig/${id}`)
    .then(response => response.json())
    .then(data => {
      const ifaceSelect = document.getElementById('basic-form-interface');
      if (pathname != "/ui/subs/create") {
        ifaceSelect.innerHTML = '<option value="any">any</option>'; // Clear previous options
      } else {
        ifaceSelect.innerHTML = '<option value="x">Choose One</option>'; // Clear previous options
      }

      const gwSelect = document.getElementById('basic-form-gateway');
      gwSelect.innerHTML = '<option value="">Default</option>'; // Clear previous options

      data.ifaces.forEach(iface => {
        if (pathname == "/ui/subs/create") {
          if (iface.type == "internal"){
            const option = document.createElement('option');
            option.value = iface.name;
            option.textContent = iface.name;
            ifaceSelect.appendChild(option);
          }
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

function disableFields(interface){
  const pfc = document.getElementById('basic-form-pfconfig');
  fetch(`/ui/firewall/pfconfig/${pfc.value}`)
    .then(response => response.json())
    .then(data => {
      data.ifaces.forEach(iface => {
        if (iface.device == interface) {
          const gwSelect = document.getElementById('basic-form-gateway');
          const planSelect = document.getElementById('basic-form-plan');
          if (iface.type == "internal"){
            gwSelect.disabled = false;
            planSelect.disabled = false;
          } else {
            gwSelect.disabled = true;
            planSelect.disabled = true;
          }
        }
      });
    })
    .catch(error => console.error('Error fetching items:', error));
}


// update interface drop down on router change
document.getElementById('basic-form-pfconfig').addEventListener('change', function() {
  const selectedCategory = this.value;
  getPfconfig(encodeURIComponent(selectedCategory));
});

// Disable non applicable features for internal source rules
document.getElementById('basic-form-interface').addEventListener('change', function() {
  const selectedCategory = this.value;
  disableFields(encodeURIComponent(selectedCategory));
});

// Toggle the Protocol select vs. its "Other" free-text fallback. Both
// controls share name="Proto", so whichever one isn't the active source is
// stripped of its name (and, for the text input, disabled) - otherwise both
// would submit a "Proto" value. The select itself stays enabled even while
// "Other" is active so the user can switch back to a known protocol.
(function() {
  const select = document.getElementById('basic-form-protocol');
  const other = document.getElementById('basic-form-protocol-other');
  if (!select || !other) return;

  function sync() {
    const isOther = select.value === 'other';
    other.disabled = !isOther;
    other.style.display = isOther ? 'block' : 'none';
    if (isOther) {
      select.removeAttribute('name');
    } else {
      select.setAttribute('name', 'Proto');
    }
  }

  const isKnown = Array.from(select.options).some(function(opt) {
    return opt.value !== 'other' && opt.value === other.value;
  });
  if (other.value && isKnown) {
    select.value = other.value;
    other.value = '';
  } else if (other.value) {
    select.value = 'other';
  }
  sync();
  select.addEventListener('change', sync);
})();
