// Interface list of the subscriber forms (new and edit). It follows the
// "Enable PPPoE" checkbox: a PPPoE subscriber is served on a pppacN interface
// (one exists per PPPoE server), any other subscriber on a regular internal
// interface. pf matches a subscriber on the interface *name*, so that is the
// option value.
(function () {
  const sel = document.getElementById('basic-form-interface');
  const box = document.getElementById('basic-form-pppoe-enable');
  const pfc = document.getElementById('basic-form-pfconfig');
  if (!sel || !box || !pfc) return;
  let ifaces = null;
  const isPppac = (i) => i.device.startsWith('pppac');
  const hasServer = () => ifaces.some((i) => i.type == 'internal' && isPppac(i));
  const NO_SERVER = 'There is no PPPoE server on record. Add one under PPPoE first; PPPoE cannot be enabled for a subscriber until then.';

  function render(want) {
    sel.innerHTML = '';
    const pppoe = box.checked;
    if (!pppoe) sel.add(new Option('Choose One', 'x'));
    ifaces.filter((i) => i.type == 'internal' && isPppac(i) == pppoe).forEach((i) => {
      const o = new Option(pppoe ? i.name + ' (' + i.device + ')' : i.name, i.name);
      o.selected = i.name == want;
      sel.add(o);
    });
  }

  // Called by the checkbox. Refuses (and unticks) when no PPPoE server exists.
  window.pppoeInterfaces = function () {
    if (!ifaces) return;
    if (box.checked && !hasServer()) {
      box.checked = false;
      window.togglePppoeFields(box);
      alert(NO_SERVER);
    }
    render('');
  };

  document.querySelector('form').addEventListener('submit', function (e) {
    if (box.checked && ifaces && !hasServer()) {
      e.preventDefault();
      alert(NO_SERVER);
    }
  });

  fetch('/ui/firewall/pfconfig/' + encodeURIComponent(pfc.value))
    .then((r) => r.json())
    .then((data) => {
      ifaces = data.ifaces || [];
      // An existing PPPoE subscriber whose server was deleted is shown as is
      // but cannot be saved until a server exists again.
      if (box.checked && !hasServer()) alert(NO_SERVER);
      render(sel.dataset.current || '');
    })
    .catch((e) => console.error('Error fetching interfaces:', e));
})();
