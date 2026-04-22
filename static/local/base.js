document.addEventListener('DOMContentLoaded', function getPfconfig(id){
  fetch(`/ui/dashboard/changes`)
    .then(response => response.json())
    .then(data => {
      if (data.length > 0) {
        const changebar = document.getElementById('changes');
        changebar.innerHTML = 'Saved configuration not yet applied. &nbsp<button id="apply" type="button" class="btn btn-sm btn-outline-danger">Apply Changes</button>';
      }
    })
    .catch(error => console.error('Error fetching items:', error));
});

function configerrors(data){

}

const button = document.querySelector('#apply');
button.addEventListener('click', function(event) {
  fetch(`/ui/dashboard/changes/apply`)
    .then(response => response.json())
    .then(data => {
      if (data.length > 0) {
        alert("Errors found while applying the configurations, see web console logs.")
        data.forEach(element => console.log(element));
      }
    })
    .catch(error => console.error('Error fetching items:', error));
});

