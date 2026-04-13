document.addEventListener('DOMContentLoaded', function getPfconfig(id){
  fetch(`/ui/dashboard/changes`)
    .then(response => response.json())
    .then(data => {
      if (data.length > 0) {
        const changebar = document.getElementById('changes');
        changebar.innerText = data.length;
      }
    })
    .catch(error => console.error('Error fetching items:', error));
});