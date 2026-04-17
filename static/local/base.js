document.addEventListener('DOMContentLoaded', function getPfconfig(id){
  fetch(`/ui/dashboard/changes`)
    .then(response => response.json())
    .then(data => {
      if (data.length > 0) {
        const changebar = document.getElementById('changes');
        changebar.innerHTML = 'Saved configuration not yet applied. &nbsp<button id="apply" type="button" class="btn btn-outline-dark">Apply Changes</button>';
      }
    })
    .catch(error => console.error('Error fetching items:', error));
});