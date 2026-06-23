// Theme toggle
(function() {
  var STORAGE_KEY = 'fur-theme';
  var LEGACY_STORAGE_KEY = 'lookit-theme';
  var html = document.documentElement;

  function getStoredTheme() {
    try {
      var v = localStorage.getItem(STORAGE_KEY);
      if (v) return v;
      // One-time migration from the pre-rename key so existing users keep their choice.
      var legacy = localStorage.getItem(LEGACY_STORAGE_KEY);
      if (legacy) {
        localStorage.setItem(STORAGE_KEY, legacy);
        localStorage.removeItem(LEGACY_STORAGE_KEY);
        return legacy;
      }
      return null;
    } catch(e) { return null; }
  }

  function setTheme(theme) {
    html.setAttribute('data-theme', theme);
    try { localStorage.setItem(STORAGE_KEY, theme); } catch(e) {}
  }

  var stored = getStoredTheme();
  if (stored) setTheme(stored);

  var btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.addEventListener('click', function() {
      var current = html.getAttribute('data-theme') || 'auto';
      var next = current === 'dark' ? 'light' : current === 'light' ? 'auto' : 'dark';
      setTheme(next);
    });
  }
})();

// Search overlay
(function() {
  var overlay = document.getElementById('search-overlay');
  var input = document.getElementById('search-input');
  var results = document.getElementById('search-results');
  if (!overlay || !input || !results) return;

  var activeIdx = -1;
  var items = [];
  var debounceTimer = null;

  function open() {
    overlay.hidden = false;
    input.value = '';
    results.innerHTML = '';
    activeIdx = -1;
    items = [];
    input.focus();
  }

  function close() {
    overlay.hidden = true;
  }

  function navigate(idx) {
    if (items.length === 0) return;
    if (activeIdx >= 0 && activeIdx < items.length) items[activeIdx].classList.remove('active');
    activeIdx = Math.max(0, Math.min(items.length - 1, idx));
    items[activeIdx].classList.add('active');
    items[activeIdx].scrollIntoView({ block: 'nearest' });
  }

  function selectCurrent() {
    if (activeIdx >= 0 && activeIdx < items.length) {
      var a = items[activeIdx].querySelector('a');
      if (a) window.location.href = a.href;
    }
  }

  function doSearch(query) {
    if (!query) { results.innerHTML = ''; items = []; activeIdx = -1; return; }
    fetch('/__api/files?q=' + encodeURIComponent(query))
      .then(function(r) { return r.json(); })
      .then(function(entries) {
        results.innerHTML = '';
        var list = (entries || []).slice(0, 20);
        list.forEach(function(entry) {
          var li = document.createElement('li');
          var a = document.createElement('a');
          a.href = '/' + (entry.RelPath || entry.Path || '');
          a.textContent = entry.RelPath || entry.Path || '';
          a.style.color = 'inherit';
          a.style.textDecoration = 'none';
          a.style.display = 'block';
          li.appendChild(a);
          results.appendChild(li);
        });
        items = Array.from(results.querySelectorAll('li'));
        activeIdx = -1;
        if (items.length > 0) navigate(0);
      });
  }

  input.addEventListener('input', function() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function() { doSearch(input.value.trim()); }, 150);
  });

  input.addEventListener('keydown', function(e) {
    if (e.key === 'ArrowDown') { e.preventDefault(); navigate(activeIdx + 1); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); navigate(activeIdx - 1); }
    else if (e.key === 'Enter') { e.preventDefault(); selectCurrent(); }
    else if (e.key === 'Escape') { close(); }
  });

  overlay.addEventListener('click', function(e) {
    if (e.target === overlay) close();
  });

  document.addEventListener('keydown', function(e) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      overlay.hidden ? open() : close();
    }
    if (e.key === 'Escape' && !overlay.hidden) close();
  });

  var searchBtn = document.getElementById('search-toggle');
  if (searchBtn) searchBtn.addEventListener('click', open);
})();
