// SSE live-reload, externalized from markdown.html so the page CSP can drop
// 'unsafe-inline' from script-src (audit Chain D / hardening 4.4). The target
// document's relative path is read from a data attribute rather than
// interpolated into an inline script. No-ops on pages without the marker.
(function () {
  var el = document.getElementById('fur-livereload');
  if (!el) return;
  var relPath = el.getAttribute('data-relpath');
  var es = new EventSource('/__events');
  es.onmessage = function (e) {
    if (e.data === relPath || e.data === '*') {
      location.reload();
    }
  };
})();
