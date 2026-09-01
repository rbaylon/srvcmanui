document.addEventListener('DOMContentLoaded', function () {
  if (typeof feather !== 'undefined') {
    feather.replace();
  }
});

document.addEventListener('DOMContentLoaded', function getPfconfig(id){
  fetch(`/ui/dashboard/changes`)
    .then(response => response.json())
    .then(data => {
      if (data.length > 0) {
        const changebar = document.getElementById('changes');
        changebar.innerHTML = 'Saved configuration not yet applied. &nbsp<button type="button" class="btn btn-sm btn-outline-danger" onclick="applyCahnges()">Apply Changes</button>';
      }
    })
    .catch(error => console.error('Error fetching items:', error));
});

async function applyCahnges() {
  await fetch(`/ui/dashboard/changes/apply`)
    .then(response => response.json())
    .then(data => {
      if (data.length > 0) {
        alert("Errors found while applying the configurations, see web console logs.")
        data.forEach(element => console.log(element));
      }
  })
  .catch(error => console.error('Error fetching items:', error));
  window.location.reload()
}
