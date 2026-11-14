// ORZU POS - Universal Touchscreen Virtual Keyboard for POS Monoblocks
// 100% Offline, Pure Vanilla JS, Day/Light Mode ("Kun rejimi") High Contrast
(function() {
  let activeInput = null;
  let isVisible = false;
  let currentMode = 'numpad'; // 'numpad' or 'qwerty'
  let isCyrillic = false;
  let isShift = false;
  let lastActionTime = 0;
  let badgeTimeout = null;
  let bubbleTimeout = null;

  const latinShiftRow0 = ['!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_', '+'];

  const latinRows = [
    ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='],
    ['Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P', "O'", "G'"],
    ['A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', 'Sh', 'Ch'],
    ['Z', 'X', 'C', 'V', 'B', 'N', 'M', ',', '.', '/']
  ];

  const cyrillicRows = [
    ['1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='],
    ['Й', 'Ц', 'У', 'К', 'Е', 'Н', 'Г', 'Ш', 'Щ', 'З', 'Х', 'Ъ'],
    ['Ф', 'Ы', 'В', 'А', 'П', 'Р', 'О', 'Л', 'Д', 'Ж', 'Э'],
    ['Я', 'Ч', 'С', 'М', 'И', 'Т', 'Ь', 'Б', 'Ю', '.', '/']
  ];

  function isExcludedInput(el) {
    if (!el) return true;
    if (el.tagName !== 'INPUT' && el.tagName !== 'TEXTAREA') return true;
    if (el.type === 'hidden' || el.type === 'file' || el.type === 'checkbox' || el.type === 'radio') return true;
    if (el.readOnly || el.disabled) return true;
    if (el.dataset && el.dataset.noVk === 'true') return true;
    if (el.closest && el.closest('[data-no-vk="true"]')) return true;
    // Modals with their own built-in large touch pads
    if (el.closest && (el.closest('#cashier-lock-modal') || el.closest('#modal-qty-edit'))) return true;
    return false;
  }

  function isNumericInput(el) {
    if (!el) return false;
    if (el.dataset && el.dataset.mode === 'numpad') return true;
    if (el.dataset && el.dataset.mode === 'qwerty') return false;

    const type = (el.type || '').toLowerCase();
    const mode = (el.inputMode || '').toLowerCase();
    const id = (el.id || '').toLowerCase();
    const name = (el.name || '').toLowerCase();
    const placeholder = (el.placeholder || '').toLowerCase();
    const step = (el.step || '').toLowerCase();

    if (type === 'number' || type === 'tel' || mode === 'numeric' || mode === 'decimal') return true;
    if (step && step !== 'any' && !isNaN(parseFloat(step))) return true;

    const numKeywords = [
      'qty', 'kilo', 'kelo', 'kg', 'vazn', 'ogirlik', 'miqdor', 'soni', 'dona',
      'price', 'narx', 'sum', 'summa', 'amount', 'qiymat',
      'pin', 'phone', 'tel', 'kod', 'code', 'barcode', 'shtrix',
      'foiz', 'chegirma', 'percent', 'discount',
      'stock', 'qoldiq', 'cost', 'tannarx', 'count', 'raqam'
    ];

    for (let i = 0; i < numKeywords.length; i++) {
      const kw = numKeywords[i];
      if (id.includes(kw) || name.includes(kw) || placeholder.includes(kw)) {
        return true;
      }
    }
    return false;
  }

  function findBestActiveInput() {
    const active = document.activeElement;
    if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA') && !isExcludedInput(active)) {
      return active;
    }

    // Check open visible modals
    const openModals = Array.from(document.querySelectorAll('.fixed, [role="dialog"], .modal')).filter(m => {
      return !m.classList.contains('hidden') && m.style.display !== 'none' && m.id !== 'orzu-virtual-keyboard-root';
    });

    for (const modal of openModals) {
      const inp = modal.querySelector('input:not([type="hidden"]):not([disabled]):not([readonly]), textarea:not([disabled]):not([readonly])');
      if (inp && !isExcludedInput(inp)) return inp;
    }

    // Check barcode input in kassa
    const bInput = document.getElementById('barcode-input');
    if (bInput && !bInput.disabled && !bInput.readOnly) {
      return bInput;
    }

    // Any available visible input
    const anyInp = document.querySelector('input:not([type="hidden"]):not([disabled]):not([readonly]), textarea:not([disabled]):not([readonly])');
    if (anyInp && !isExcludedInput(anyInp)) {
      return anyInp;
    }

    return null;
  }

  function updateTargetHint() {
    const hint = document.getElementById('vk-target-hint');
    if (!hint) return;
    if (!activeInput) {
      hint.innerText = '';
      return;
    }
    const name = activeInput.placeholder || activeInput.getAttribute('aria-label') || activeInput.name || activeInput.id || 'Maydon';
    hint.innerText = '👉 ' + name;
  }

  function createKeyboard() {
    if (document.getElementById('orzu-virtual-keyboard-root')) return;

    const root = document.createElement('div');
    root.id = 'orzu-virtual-keyboard-root';
    root.style.cssText = `
      position: fixed;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 999999;
      background: #ffffff;
      border-top: 2px solid #cbd5e1;
      box-shadow: 0 -8px 30px rgba(0, 0, 0, 0.15);
      padding: 6px 14px 10px 14px;
      font-family: inherit;
      user-select: none;
      -webkit-user-select: none;
      display: none;
    `;

    // Compact Keyboard Header Bar
    const header = document.createElement('div');
    header.style.cssText = `
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 6px;
      border-bottom: 1px solid #e2e8f0;
      background: #f8fafc;
      padding: 4px 8px;
      border-radius: 8px;
    `;

    header.innerHTML = `
      <div style="display: flex; align-items: center; gap: 8px;">
        <span style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: #059669;"></span>
        <span style="font-size: 11px; font-weight: 800; color: #0f172a; letter-spacing: 0.5px;">SENSOR KLAVIATURA</span>
        <span id="vk-target-hint" style="font-size: 11px; color: #881337; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 700;"></span>
        <div id="vk-last-pressed" style="display: none; align-items: center; gap: 4px; padding: 2px 8px; background: #881337; color: #ffffff; border-radius: 6px; font-size: 11px; font-weight: 800; box-shadow: 0 1px 3px rgba(136,19,55,0.3); transition: opacity 0.2s;">
          <span style="opacity: 0.85; font-size: 10px;">Bosildi:</span>
          <span id="vk-last-char" style="background: #ffffff; color: #881337; padding: 0 6px; border-radius: 4px; font-size: 12px; font-weight: 900; min-width: 16px; text-align: center;">-</span>
        </div>
      </div>
      <div style="display: flex; align-items: center; gap: 6px;">
        <button id="vk-mode-btn" type="button" style="padding: 4px 12px; border-radius: 8px; font-size: 11px; font-weight: 700; background: #ffffff; color: #0f172a; border: 1px solid #cbd5e1; cursor: pointer; box-shadow: 0 1px 2px rgba(0,0,0,0.05);">123 Raqamlar</button>
        <button id="vk-lang-btn" type="button" style="padding: 4px 12px; border-radius: 8px; font-size: 11px; font-weight: 700; background: #ffffff; color: #0f172a; border: 1px solid #cbd5e1; cursor: pointer; display: none; box-shadow: 0 1px 2px rgba(0,0,0,0.05);">UZ Lotin</button>
        <button id="vk-close-btn" type="button" style="padding: 4px 14px; border-radius: 8px; font-size: 11px; font-weight: 800; background: #fee2e2; color: #b91c1c; border: 1px solid #fca5a5; cursor: pointer;">✕ Yopish</button>
      </div>
    `;

    const body = document.createElement('div');
    body.id = 'vk-body';

    root.appendChild(header);
    root.appendChild(body);
    document.body.appendChild(root);

    if (!document.getElementById('vk-style-block')) {
      const st = document.createElement('style');
      st.id = 'vk-style-block';
      st.textContent = `
        .vk-key, .vk-action-key {
          touch-action: manipulation;
          -webkit-tap-highlight-color: transparent;
        }
        .vk-pressed {
          background-color: #881337 !important;
          color: #ffffff !important;
          border-color: #4c0519 !important;
          box-shadow: 0 0 10px rgba(136, 19, 55, 0.7) !important;
          transform: scale(0.92) !important;
        }
      `;
      document.head.appendChild(st);
    }

    // Floating Magnifier Bubble directly above pressed key
    let bubble = document.getElementById('vk-floating-bubble');
    if (!bubble) {
      bubble = document.createElement('div');
      bubble.id = 'vk-floating-bubble';
      bubble.style.cssText = `
        position: fixed;
        display: none;
        pointer-events: none;
        z-index: 1000002;
        min-width: 44px;
        height: 48px;
        padding: 0 8px;
        background: #881337;
        color: #ffffff;
        font-size: 22px;
        font-weight: 900;
        border-radius: 10px;
        box-shadow: 0 8px 24px rgba(0,0,0,0.35);
        border: 2px solid #ffffff;
        align-items: center;
        justify-content: center;
        text-align: center;
        transform: translate(-50%, -100%);
      `;
      document.body.appendChild(bubble);
    }

    // Header buttons
    document.getElementById('vk-close-btn').addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      hideKeyboard();
    });
    document.getElementById('vk-mode-btn').addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      currentMode = (currentMode === 'qwerty') ? 'numpad' : 'qwerty';
      renderKeys();
      if (activeInput) adjustScrollContainerForInput(activeInput);
    });
    document.getElementById('vk-lang-btn').addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      isCyrillic = !isCyrillic;
      renderKeys();
    });

    renderKeys();
  }

  function renderKeys() {
    const body = document.getElementById('vk-body');
    const modeBtn = document.getElementById('vk-mode-btn');
    const langBtn = document.getElementById('vk-lang-btn');
    if (!body) return;

    if (currentMode === 'numpad') {
      if (modeBtn) modeBtn.innerText = 'ABC Harflar';
      if (langBtn) langBtn.style.display = 'none';

      // Sleek, Ultra-Responsive Light Numpad Layout (~165px height)
      body.innerHTML = `
        <div style="display: flex; justify-content: center; gap: 10px; max-width: 520px; margin: 0 auto;">
          <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; flex: 3;">
            <button class="vk-key" data-key="7">7</button>
            <button class="vk-key" data-key="8">8</button>
            <button class="vk-key" data-key="9">9</button>
            <button class="vk-key" data-key="4">4</button>
            <button class="vk-key" data-key="5">5</button>
            <button class="vk-key" data-key="6">6</button>
            <button class="vk-key" data-key="1">1</button>
            <button class="vk-key" data-key="2">2</button>
            <button class="vk-key" data-key="3">3</button>
            <button class="vk-key" data-key="0">0</button>
            <button class="vk-key" data-key="00">00</button>
            <button class="vk-key" data-key="." style="font-weight: 900; font-size: 18px;">.</button>
          </div>
          <div style="display: flex; flex-direction: column; gap: 6px; flex: 1.2;">
            <button class="vk-action-key" data-action="backspace" style="background: #fef2f2; color: #dc2626; border: 1px solid #fca5a5; flex: 1; font-size: 13px; font-weight: 800;">⌫ O'chirish</button>
            <button class="vk-action-key" data-action="clear" style="background: #f8fafc; color: #475569; border: 1px solid #cbd5e1; flex: 1; font-size: 13px; font-weight: 700;">C Tozalash</button>
            <button class="vk-action-key" data-action="enter" style="background: #059669; color: #ffffff; border: 1px solid #047857; flex: 1.6; font-size: 14px; font-weight: 900; box-shadow: 0 2px 4px rgba(5,150,105,0.25);">✓ OK (Enter)</button>
          </div>
        </div>
      `;
    } else {
      if (modeBtn) modeBtn.innerText = '123 Raqamlar';
      if (langBtn) {
        langBtn.style.display = 'inline-block';
        langBtn.innerText = isCyrillic ? 'ЎЗ Кирилл' : 'UZ Lotin';
      }

      // Compact QWERTY Touch Layout in Light Mode
      const rows = isCyrillic ? cyrillicRows : latinRows;
      let html = '<div style="display: flex; flex-direction: column; gap: 4px; max-width: 960px; margin: 0 auto;">';

      rows.forEach((row, idx) => {
        html += '<div style="display: flex; justify-content: center; gap: 4px;">';
        if (idx === 3) {
          html += `<button class="vk-action-key" data-action="shift" style="width: 68px; background: ${isShift ? '#881337' : '#f8fafc'}; color: ${isShift ? '#ffffff' : '#0f172a'}; border: 1px solid ${isShift ? '#881337' : '#cbd5e1'}; font-size: 12px; font-weight: 800;">⇧ Shift</button>`;
        }
        const activeRow = (idx === 0 && isShift && !isCyrillic) ? latinShiftRow0 : row;
        activeRow.forEach(ch => {
          let displayChar = ch;
          if (ch.length === 1 && ch.match(/[a-zA-Zа-яА-ЯёЁ]/)) {
            displayChar = isShift ? ch.toUpperCase() : ch.toLowerCase();
          }
          html += `<button class="vk-key" data-key="${displayChar}">${displayChar}</button>`;
        });
        if (idx === 0) {
          html += '<button class="vk-action-key" data-action="backspace" style="width: 68px; background: #fef2f2; color: #dc2626; border: 1px solid #fca5a5; font-size: 13px; font-weight: 800;">⌫</button>';
        } else if (idx === 3) {
          html += '<button class="vk-action-key" data-action="enter" style="width: 80px; background: #059669; color: #ffffff; border: 1px solid #047857; font-weight: 800; font-size: 13px; box-shadow: 0 1px 3px rgba(5,150,105,0.25);">↵ Kirish</button>';
        }
        html += '</div>';
      });

      // Bottom Row (Space, Tab, Clear) - NO '@' AT THE BOTTOM
      html += `
        <div style="display: flex; justify-content: center; gap: 6px; margin-top: 3px;">
          <button class="vk-action-key" data-action="clear" style="padding: 0 18px; background: #f8fafc; color: #475569; border: 1px solid #cbd5e1; font-size: 12px; font-weight: 700;">Tozalash</button>
          <button class="vk-key" data-key=" " style="flex: 1; max-width: 560px; background: #f8fafc; color: #0f172a; border: 1px solid #cbd5e1; font-size: 13px; font-weight: 700;">Bo'sh joy (Space)</button>
          <button class="vk-key" data-key=".uz" style="padding: 0 16px; font-size: 12px; font-weight: 700;">.uz</button>
        </div>
      `;

      html += '</div>';
      body.innerHTML = html;
    }

    applyKeyStyles();
    attachKeyListeners();
  }

  function applyKeyStyles() {
    const isNum = (currentMode === 'numpad');
    const keys = document.querySelectorAll('.vk-key, .vk-action-key');
    keys.forEach(k => {
      k.style.height = isNum ? '38px' : '33px';
      k.style.borderRadius = '8px';
      k.style.fontSize = isNum ? '16px' : '13px';
      k.style.fontWeight = '700';
      k.style.cursor = 'pointer';
      k.style.display = 'flex';
      k.style.alignItems = 'center';
      k.style.justifyContent = 'center';
      k.style.userSelect = 'none';
      k.style.webkitUserSelect = 'none';

      if (k.classList.contains('vk-key') && !k.style.background) {
        k.style.background = '#ffffff';
        k.style.color = '#0f172a';
        k.style.border = '1px solid #cbd5e1';
        k.style.boxShadow = '0 1px 2px rgba(0,0,0,0.05)';
        k.style.minWidth = isNum ? '48px' : '38px';
        k.style.flex = '1';
      }
    });
  }

  let activeFeedbackBtn = null;
  let activeFeedbackTimeout = null;

  function showKeyFeedback(btn, charOrAction, isAction) {
    if (!btn) return;

    // Tezkor teginishda oldingi tugmani bir zumda tozalash
    if (activeFeedbackBtn && activeFeedbackBtn !== btn) {
      activeFeedbackBtn.classList.remove('vk-pressed');
    }
    if (activeFeedbackTimeout) {
      clearTimeout(activeFeedbackTimeout);
      activeFeedbackTimeout = null;
    }

    // 1. Tugmani chaqmoqdek tez yorqin bordo holatiga o'tkazish
    activeFeedbackBtn = btn;
    btn.classList.add('vk-pressed');

    // 2. Format label
    let label = charOrAction;
    if (isAction) {
      if (charOrAction === 'backspace') label = '⌫';
      else if (charOrAction === 'clear') label = 'Tozalash';
      else if (charOrAction === 'enter') label = '✓ OK';
      else if (charOrAction === 'shift') label = '⇧ Shift';
    } else if (charOrAction === ' ') {
      label = 'Bo\'sh joy';
    }

    // 3. Sarlavhadagi "Bosildi" nishonini kechikishsiz yangilash
    const badge = document.getElementById('vk-last-pressed');
    const charSpan = document.getElementById('vk-last-char');
    if (badge && charSpan) {
      charSpan.innerText = label;
      badge.style.display = 'inline-flex';
    }

    // 4. Barmoq ustida darhol chiqib, bir zumda yo'qoluvchi ko'rsatkich pufak
    const bubble = document.getElementById('vk-floating-bubble');
    if (bubble) {
      const rect = btn.getBoundingClientRect();
      bubble.innerText = (label.length > 4) ? label.slice(0, 4) : label;
      bubble.style.left = (rect.left + rect.width / 2) + 'px';
      bubble.style.top = (rect.top - 6) + 'px';
      bubble.style.display = 'flex';

      if (bubbleTimeout) clearTimeout(bubbleTimeout);
      bubbleTimeout = setTimeout(() => {
        bubble.style.display = 'none';
      }, 85);
    }

    activeFeedbackTimeout = setTimeout(() => {
      btn.classList.remove('vk-pressed');
      if (activeFeedbackBtn === btn) activeFeedbackBtn = null;
    }, 70);
  }

  function executeKey(charOrAction, isAction = false) {
    const now = Date.now();
    if (now - lastActionTime < 10) return;
    lastActionTime = now;

    if (isAction) {
      if (charOrAction === 'backspace') deleteBackward();
      else if (charOrAction === 'clear') clearInput();
      else if (charOrAction === 'enter') pressEnter();
      else if (charOrAction === 'shift') {
        isShift = !isShift;
        renderKeys();
      }
    } else {
      insertText(charOrAction);
      if (isShift) {
        isShift = false;
        renderKeys();
      }
    }
  }

  function attachKeyListeners() {
    const allButtons = document.querySelectorAll('.vk-key, .vk-action-key');

    allButtons.forEach(btn => {
      const isAction = btn.classList.contains('vk-action-key');
      const val = isAction ? btn.getAttribute('data-action') : btn.getAttribute('data-key');
      if (val === null) return;

      // Pointerdown gives instant zero-latency touch response and keeps active input focused
      btn.onpointerdown = (e) => {
        e.preventDefault();
        e.stopPropagation();
        showKeyFeedback(btn, val, isAction);
        executeKey(val, isAction);
      };

      btn.onpointerup = () => {
        btn.classList.remove('vk-pressed');
      };
      btn.onpointercancel = () => {
        btn.classList.remove('vk-pressed');
      };
      btn.onpointerleave = () => {
        btn.classList.remove('vk-pressed');
      };

      // Fallback click for systems without pointer events
      btn.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();
        if (Date.now() - lastActionTime > 150) {
          showKeyFeedback(btn, val, isAction);
          executeKey(val, isAction);
        }
      };
    });
  }
        }
      };
    });
  }

  function insertText(str) {
    if (!activeInput) {
      activeInput = findBestActiveInput();
      if (!activeInput) return;
    }
    const el = activeInput;

    try { el.focus(); } catch (e) {}

    // First attempt native execCommand which cleanly handles cursor, undo, and type="number"
    let success = false;
    try {
      success = document.execCommand('insertText', false, str);
    } catch (e) {
      success = false;
    }

    if (!success) {
      const val = el.value || '';
      let start = val.length;
      let end = val.length;
      let canSlice = true;

      try {
        start = el.selectionStart ?? val.length;
        end = el.selectionEnd ?? val.length;
      } catch (e) {
        canSlice = false; // type="number" in Chromium
      }

      if (canSlice) {
        if (str === '.' && val.includes('.') && isNumericInput(el)) return;
        el.value = val.slice(0, start) + str + val.slice(end);
        try {
          const newPos = start + str.length;
          el.setSelectionRange(newPos, newPos);
        } catch (e) {}
      } else {
        // Fallback for type="number"
        if (str === '.' && val.includes('.')) return;
        const origType = el.type;
        try {
          if (str === '.') {
            el.type = 'text';
            el.value = val + '.';
          } else {
            el.value = val + str;
          }
        } catch (e) {
          el.value = val + str;
        } finally {
          if (str !== '.' && el.type !== origType) {
            try { el.type = origType; } catch (e) {}
          }
        }
      }
    }

    triggerEvents(el);
  }

  function deleteBackward() {
    if (!activeInput) {
      activeInput = findBestActiveInput();
      if (!activeInput) return;
    }
    const el = activeInput;
    try { el.focus(); } catch (e) {}

    let success = false;
    try {
      success = document.execCommand('delete', false, null);
    } catch (e) {
      success = false;
    }

    if (!success) {
      const val = el.value || '';
      if (!val) return;

      let start = val.length;
      let end = val.length;
      let canSlice = true;

      try {
        start = el.selectionStart ?? val.length;
        end = el.selectionEnd ?? val.length;
      } catch (e) {
        canSlice = false;
      }

      if (canSlice) {
        if (start === end) {
          if (start > 0) {
            el.value = val.slice(0, start - 1) + val.slice(start);
            try { el.setSelectionRange(start - 1, start - 1); } catch (e) {}
          }
        } else {
          el.value = val.slice(0, start) + val.slice(end);
          try { el.setSelectionRange(start, start); } catch (e) {}
        }
      } else {
        // Fallback: remove last character
        el.value = val.slice(0, -1);
      }
    }

    triggerEvents(el);
  }

  function clearInput() {
    if (!activeInput) {
      activeInput = findBestActiveInput();
      if (!activeInput) return;
    }
    activeInput.value = '';
    triggerEvents(activeInput);
  }

  function pressEnter() {
    if (!activeInput) {
      activeInput = findBestActiveInput();
    }
    const el = activeInput;
    if (el) {
      triggerEvents(el);

      // Dispatch keyboard event for Enter
      const enterDown = new KeyboardEvent('keydown', {
        key: 'Enter',
        code: 'Enter',
        keyCode: 13,
        which: 13,
        bubbles: true,
        cancelable: true
      });
      el.dispatchEvent(enterDown);

      const enterUp = new KeyboardEvent('keyup', {
        key: 'Enter',
        code: 'Enter',
        keyCode: 13,
        which: 13,
        bubbles: true,
        cancelable: true
      });
      el.dispatchEvent(enterUp);

      if (el.form && typeof el.form.requestSubmit === 'function') {
        try { el.form.requestSubmit(); } catch (e) {}
      }
    }

    hideKeyboard();
    if (el) {
      try { el.blur(); } catch (e) {}
    }
  }

  function triggerEvents(el) {
    try {
      el.dispatchEvent(new InputEvent('input', { bubbles: true, cancelable: true, data: el.value }));
    } catch (e) {
      el.dispatchEvent(new Event('input', { bubbles: true }));
    }
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }

  // Adjust scrollable containers so input is never covered by keyboard
  function adjustScrollContainerForInput(input) {
    if (!input) return;

    const scrollParents = [
      input.closest('.overflow-y-auto'),
      input.closest('.overflow-auto'),
      input.closest('#product-modal > div'),
      input.closest('#modal-customer > div'),
      input.closest('#modal-cashier > div'),
      input.closest('#modal-print-label > div'),
      input.closest('main')
    ].filter(Boolean);

    scrollParents.forEach(p => {
      if (!p.dataset.origPaddingBottom) {
        p.dataset.origPaddingBottom = p.style.paddingBottom || '0px';
      }
      p.style.paddingBottom = '320px';
    });

    document.body.style.paddingBottom = '260px';

    setTimeout(() => {
      try {
        input.scrollIntoView({ behavior: 'smooth', block: 'center' });
      } catch (e) {}
    }, 70);
  }

  function restoreScrollContainers() {
    document.querySelectorAll('[data-orig-padding-bottom]').forEach(el => {
      el.style.paddingBottom = el.dataset.origPaddingBottom || '';
      delete el.dataset.origPaddingBottom;
    });
    document.body.style.paddingBottom = '';
  }

  function showKeyboard() {
    const root = document.getElementById('orzu-virtual-keyboard-root');
    if (!root) return;
    root.style.display = 'block';
    isVisible = true;

    if (activeInput) {
      adjustScrollContainerForInput(activeInput);
    }
  }

  function hideKeyboard() {
    const root = document.getElementById('orzu-virtual-keyboard-root');
    if (!root) return;
    root.style.display = 'none';
    isVisible = false;
    restoreScrollContainers();
  }

  function toggleKeyboard() {
    if (isVisible) {
      hideKeyboard();
    } else {
      if (!activeInput || isExcludedInput(activeInput)) {
        activeInput = findBestActiveInput();
      }
      if (activeInput) {
        currentMode = isNumericInput(activeInput) ? 'numpad' : 'qwerty';
        updateTargetHint();
        adjustScrollContainerForInput(activeInput);
      }
      renderKeys();
      showKeyboard();
    }
  }

  function handleInputActivation(el) {
    if (isExcludedInput(el)) return;

    activeInput = el;
    updateTargetHint();

    if (isNumericInput(el)) {
      currentMode = 'numpad';
    } else {
      currentMode = 'qwerty';
    }
    renderKeys();
    showKeyboard();
    adjustScrollContainerForInput(el);
  }

  // Init when DOM is ready
  function init() {
    createKeyboard();

    // Listen to user click / tap on inputs (Real User Interaction)
    function onUserInputOnElement(e) {
      const el = e.target;
      if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
        handleInputActivation(el);
      }
    }
    document.addEventListener('pointerup', onUserInputOnElement, true);
    document.addEventListener('click', onUserInputOnElement, true);

    // Dismiss keyboard on outside touch/click:
    document.addEventListener('pointerdown', (e) => {
      if (!isVisible) return;
      const root = document.getElementById('orzu-virtual-keyboard-root');
      if (root && root.contains(e.target)) return;
      if (activeInput && (e.target === activeInput || activeInput.contains(e.target))) return;
      if (e.target && (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA')) return;
      if (e.target && e.target.closest && (e.target.closest('button[onclick*="VirtualKeyboard"]') || e.target.closest('#vk-toggle-btn'))) return;

      // Clicked outside -> hide immediately!
      hideKeyboard();
    }, true);

    // Dismiss on physical keyboard Escape or Enter
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        hideKeyboard();
      } else if (e.key === 'Enter' && isVisible) {
        setTimeout(hideKeyboard, 50);
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  window.__orzuVirtualKeyboard = {
    show: showKeyboard,
    hide: hideKeyboard,
    toggle: toggleKeyboard,
    setTarget: (el) => {
      if (el && !isExcludedInput(el)) {
        handleInputActivation(el);
      }
    }
  };
})();
