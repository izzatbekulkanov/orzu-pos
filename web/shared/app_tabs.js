// ORZU POS - Chrome-like Unified Active App Tabs Bar
// 100% Offline, Pure Vanilla JS, Auto-detection of Running Apps
(function() {
  function detectCurrentApp() {
    if (window.__ORZU_APP_NAME__) return window.__ORZU_APP_NAME__;
    const path = (window.location.pathname || '').toLowerCase();
    const port = (window.location.port || '');
    if (port === '8899') return 'installer';
    if (path.startsWith('/admin')) return 'admin';
    if (path.startsWith('/db')) return 'db';
    if (path.startsWith('/backup')) return 'backup';
    return 'kassa';
  }

  const currentApp = detectCurrentApp();
  window.__ORZU_APP_NAME__ = currentApp;

  const appMeta = {
    kassa: { name: "ORZU Kassa", icon: "shopping-cart", target: "kassa", url: "/" },
    admin: { name: "ORZU Boshqaruv", icon: "layout-dashboard", target: "admin", url: "/admin" },
    db: { name: "ORZU DB", icon: "database", target: "db", url: "/db" },
    backup: { name: "ORZU Arxiv", icon: "archive", target: "backup", url: "/backup" },
    installer: { name: "O'rnatuvchi", icon: "wrench", target: "installer", url: "http://localhost:8899" }
  };

  async function sendHeartbeat() {
    try {
      await fetch(`/api/heartbeat/${currentApp}`, { method: 'POST' });
    } catch (e) {}
  }

  async function sendLeaveHeartbeat() {
    try {
      if (navigator.sendBeacon) {
        navigator.sendBeacon(`/api/heartbeat-leave/${currentApp}`);
      } else {
        fetch(`/api/heartbeat-leave/${currentApp}`, { method: 'POST', keepalive: true });
      }
    } catch (e) {}
  }

  window.addEventListener('beforeunload', sendLeaveHeartbeat);
  window.addEventListener('pagehide', sendLeaveHeartbeat);

  let lastTabsFingerprint = "";

  async function updateTabs() {
    const container = document.getElementById('orzu-app-tabs');
    if (!container) return;

    try {
      const res = await fetch('/api/apps-status');
      if (!res.ok) return;
      const status = await res.json();

      const fingerprint = JSON.stringify(status);
      if (fingerprint === lastTabsFingerprint && container.children.length > 0) {
        return; // Hech narsa o'zgarmagan bo'lsa, DOM ga ortiqcha tegmaymiz (Lag bo'lmasligi uchun)
      }
      lastTabsFingerprint = fingerprint;

      // Determine which apps to show in the tabs bar:
      // Show an app tab IF: it is active OR it is the current app
      const visibleApps = [];
      const order = ['kassa', 'admin', 'db', 'backup', 'installer'];

      order.forEach(appKey => {
        const isActive = (status[`${appKey}_active`] === true);
        const isCurrent = (currentApp === appKey);
        if (isActive || isCurrent) {
          visibleApps.push({ key: appKey, active: isActive, isCurrent: isCurrent });
        }
      });

      let html = '<div class="flex items-center gap-1.5 overflow-x-auto scrollbar-none py-1">';
      visibleApps.forEach(app => {
        const meta = appMeta[app.key] || { name: app.key, icon: 'app-window' };
        if (app.isCurrent) {
          // Current Active Tab (Solid Luxury Burgundy, high contrast in light mode)
          html += `
            <div class="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-[#881337] text-white text-xs font-bold border border-[#881337] shadow-sm">
              <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
              <i data-lucide="${meta.icon}" class="w-3.5 h-3.5 text-white"></i>
              <span class="whitespace-nowrap">${meta.name}</span>
            </div>
          `;
        } else {
          // Other Active Running Tab (Click to switch)
          html += `
            <button onclick="window.__orzuSwitchApp('${meta.target}')" type="button" 
                    class="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white hover:bg-slate-100 text-slate-800 text-xs font-bold border border-slate-300 transition active:scale-95 cursor-pointer shadow-sm">
              <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
              <i data-lucide="${meta.icon}" class="w-3.5 h-3.5 text-slate-600"></i>
              <span class="whitespace-nowrap">${meta.name}</span>
            </button>
          `;
        }
      });

      // Quick launcher '+' button to open other family apps
      html += `
        <div class="relative inline-block" id="app-launcher-menu">
          <button onclick="window.__toggleAppLauncher()" type="button" class="p-1.5 rounded-xl bg-white hover:bg-slate-100 text-slate-700 border border-slate-300 transition cursor-pointer shadow-sm" title="Boshqa ilovani ochish">
            <i data-lucide="plus" class="w-3.5 h-3.5"></i>
          </button>
          <div id="app-launcher-dropdown" class="hidden absolute right-0 top-full mt-2 w-52 bg-white border border-slate-200 rounded-2xl p-2 shadow-2xl z-[999] space-y-1">
            <div class="text-[10px] uppercase font-bold text-slate-400 px-2 py-1">ORZU Ilovalari</div>
            <button onclick="window.__orzuSwitchApp('kassa')" class="w-full text-left px-2.5 py-1.5 rounded-xl hover:bg-slate-100 text-xs text-slate-800 font-bold flex items-center gap-2">
              <i data-lucide="shopping-cart" class="w-3.5 h-3.5 text-[#881337]"></i> Kassa
            </button>
            <button onclick="window.__orzuSwitchApp('admin')" class="w-full text-left px-2.5 py-1.5 rounded-xl hover:bg-slate-100 text-xs text-slate-800 font-bold flex items-center gap-2">
              <i data-lucide="layout-dashboard" class="w-3.5 h-3.5 text-[#881337]"></i> Boshqaruv
            </button>
            <button onclick="window.__orzuSwitchApp('db')" class="w-full text-left px-2.5 py-1.5 rounded-xl hover:bg-slate-100 text-xs text-slate-800 font-bold flex items-center gap-2">
              <i data-lucide="database" class="w-3.5 h-3.5 text-[#881337]"></i> Ma'lumotlar Bazasi
            </button>
            <button onclick="window.__orzuSwitchApp('backup')" class="w-full text-left px-2.5 py-1.5 rounded-xl hover:bg-slate-100 text-xs text-slate-800 font-bold flex items-center gap-2">
              <i data-lucide="archive" class="w-3.5 h-3.5 text-[#881337]"></i> Arxiv va Zaxira
            </button>
            <button onclick="window.__orzuSwitchApp('installer')" class="w-full text-left px-2.5 py-1.5 rounded-xl hover:bg-slate-100 text-xs text-slate-800 font-bold flex items-center gap-2">
              <i data-lucide="wrench" class="w-3.5 h-3.5 text-[#881337]"></i> O'rnatuvchi markazi
            </button>
          </div>
        </div>
      `;
      html += '</div>';

      container.innerHTML = html;
      if (window.lucide) {
        lucide.createIcons({ root: container });
      }
    } catch (e) {}
  }

  window.__orzuSwitchApp = async function(target) {
    const dropdown = document.getElementById('app-launcher-dropdown');
    if (dropdown) dropdown.classList.add('hidden');

    try {
      await fetch('/api/open-app', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target: target })
      });
    } catch (e) {
      if (appMeta[target]) {
        window.location.href = appMeta[target].url;
      }
    }
  };

  window.__toggleAppLauncher = function() {
    const dropdown = document.getElementById('app-launcher-dropdown');
    if (dropdown) dropdown.classList.toggle('hidden');
  };

  document.addEventListener('click', function(e) {
    const menu = document.getElementById('app-launcher-menu');
    const dropdown = document.getElementById('app-launcher-dropdown');
    if (menu && dropdown && !menu.contains(e.target)) {
      dropdown.classList.add('hidden');
    }
  });

  // Start heartbeat (every 6 seconds) and tabs updater (every 5 seconds)
  sendHeartbeat();
  setInterval(sendHeartbeat, 6000);
  updateTabs();
  setInterval(updateTabs, 5000);
})();
