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

      html += '</div>';

      container.innerHTML = html;
      if (window.lucide) {
        lucide.createIcons({ root: container });
      }
    } catch (e) {}
  }

  window.__orzuSwitchApp = async function(target) {
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

  // Start heartbeat (every 6 seconds) and tabs updater (every 5 seconds)
  sendHeartbeat();
  setInterval(sendHeartbeat, 6000);
  updateTabs();
  setInterval(updateTabs, 5000);
})();
