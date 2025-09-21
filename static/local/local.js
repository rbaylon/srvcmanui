document.getElementById('basic-form-pfconfig').addEventListener('change', function() {
  const selectedCategory = this.value;

  fetch(`/ui/firewall/pfconfig/${encodeURIComponent(selectedCategory)}`)
    .then(response => response.json())
    .then(data => {
      const itemsSelect = document.getElementById('basic-form-interface');
      itemsSelect.innerHTML = '<option value="any">any</option>'; // Clear previous options

      data.Ifaces.forEach(item => {
        const option = document.createElement('option');
        option.value = item.device;
        option.textContent = item.name;
        itemsSelect.appendChild(option);
      });
    })
    .catch(error => console.error('Error fetching items:', error));
});