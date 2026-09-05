document.addEventListener('DOMContentLoaded', function () {
  if (typeof feather !== 'undefined') {
    feather.replace();
  }
});

document.addEventListener('DOMContentLoaded', function () {
  const collectionsLink = document.getElementById('nav-collections');
  if (collectionsLink) {
    collectionsLink.href = '/ui/collections/' + new Date().getFullYear();
  }
});

document.addEventListener('DOMContentLoaded', function () {
  const path = window.location.pathname;
  let bestLink = null;
  let bestLen = 0;
  document.querySelectorAll('#sidebarMenu a.nav-link[href]').forEach(function (link) {
    const href = link.getAttribute('href');
    if (!href || href.charAt(0) === '#') {
      return;
    }
    if ((path === href || path.startsWith(href + '/')) && href.length > bestLen) {
      bestLen = href.length;
      bestLink = link;
    }
  });
  if (!bestLink) {
    return;
  }
  bestLink.classList.add('active');
  bestLink.setAttribute('aria-current', 'page');
  const parentCollapse = bestLink.closest('.collapse');
  if (parentCollapse && typeof bootstrap !== 'undefined') {
    new bootstrap.Collapse(parentCollapse, { toggle: false }).show();
  }
});

document.addEventListener('DOMContentLoaded', function () {
  const nameEl = document.getElementById('userMenuName');
  const profileLink = document.getElementById('userMenuProfile');
  if (!nameEl && !profileLink) {
    return;
  }
  fetch('/ui/users/whoami')
    .then(response => response.ok ? response.json() : Promise.reject('HTTP ' + response.status))
    .then(data => {
      if (nameEl) {
        nameEl.textContent = data.username || '';
      }
      const userId = data.ID ?? data.id;
      if (profileLink) {
        if (userId) {
          profileLink.href = '/ui/users/' + userId;
        } else {
          console.warn('/ui/users/whoami response had no ID/id field:', data);
        }
      }
    })
    .catch(error => console.error('Error fetching current user from /ui/users/whoami:', error));
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
