// ORZU POS - Universal Touchscreen Virtual Keyboard for POS Monoblocks
// 100% Offline, Pure Vanilla JS, Day/Light Mode ("Kun rejimi") High Contrast
(function() {
  let activeInput = null;
  let isVisible = false;
  let currentMode = 'numpad'; // 'numpad' or 'qwerty'
  let isCyrillic = false;
  let isShift = false;

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

  function createKeyboard() {
    if (document.getElementById('orzu-virtual-keyboard-root')) return;

    const root = document.createElement('div');
    root.id = 'orzu-virtual-keyboard-root';
    // Clean, crisp Light Mode ("Kun rejimi") styling with solid borders (NO GRADIENTS)
    root.style.cssText = `
      position: fixed;
      left: 0;
      right: 0;
      bottom: 0;
      z-index: 99999;
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
      padding-bottom: 5px;
      border-bottom: 1px solid #e2e8f0;
      background: #f8fafc;
      padding: 4px 8px;
      border-radius: 8px;
    `;

    header.innerHTML = `
      <div style="display: flex; align-items: center; gap: 8px;">
        <span style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: #059669;"></span>
        <span style="font-size: 11px; font-weight: 800; color: #0f172a; letter-spacing: 0.5px;">SENSOR KLAVIATURA</span>
        <span id="vk-target-hint" style="font-size: 11px; color: #475569; max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 600;"></span>
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

    // Header buttons
    document.getElementById('vk-close-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      hideKeyboard();
    });
    document.getElementById('vk-mode-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      currentMode = (currentMode === 'qwerty') ? 'numpad' : 'qwerty';
      renderKeys();
      if (activeInput) adjustScrollContainerForInput(activeInput);
    });
    document.getElementById('vk-lang-btn').addEventListener('click', (e) => {
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

      // Sleek, Ultra-Compact Light Numpad Layout (~160px height)
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
        row.forEach(ch => {
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

      // Bottom Row (Space, Tab, Clear)
      html += `
        <div style="display: flex; justify-content: center; gap: 6px; margin-top: 3px;">
          <button class="vk-action-key" data-action="clear" style="padding: 0 16px; background: #f8fafc; color: #475569; border: 1px solid #cbd5e1; font-size: 12px; font-weight: 700;">Tozalash</button>
          <button class="vk-key" data-key=" " style="flex: 1; max-width: 480px; background: #f8fafc; color: #0f172a; border: 1px solid #cbd5e1; font-size: 12px; font-weight: 700;">Bo'sh joy (Space)</button>
          <button class="vk-key" data-key="@" style="width: 46px; font-weight: 700;">@</button>
          <button class="vk-key" data-key=".uz" style="padding: 0 14px; font-size: 12px; font-weight: 700;">.uz</button>
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
      k.style.transition = 'background 0.1s, transform 0.05s';
      k.style.userSelect = 'none';

      if (k.classList.contains('vk-key') && !k.style.background) {
        k.style.background = '#ffffff';
        k.style.color = '#0f172a';
        k.style.border = '1px solid #cbd5e1';
        k.style.boxShadow = '0 1px 2px rgba(0,0,0,0.05)';
        k.style.minWidth = isNum ? '48px' : '38px';
        k.style.flex = '1';
      }

      k.addEventListener('mousedown', (e) => {
        e.preventDefault(); // Prevent blurring active input
        k.style.transform = 'scale(0.95)';
      });
      k.addEventListener('mouseup', () => {
        k.style.transform = 'scale(1)';
      });
      k.addEventListener('touchstart', (e) => {
        e.preventDefault();
        k.style.transform = 'scale(0.95)';
      }, { passive: false });
      k.addEventListener('touchend', () => {
        k.style.transform = 'scale(1)';
      });
    });
  }

  function attachKeyListeners() {
    const keys = document.querySelectorAll('.vk-key');
    keys.forEach(k => {
      k.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();
        const char = k.getAttribute('data-key');
        if (char !== null) {
          insertText(char);
          if (isShift) {
            isShift = false;
            renderKeys();
          }
        }
      };
    });

    const actions = document.querySelectorAll('.vk-action-key');
    actions.forEach(a => {
      a.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();
        const action = a.getAttribute('data-action');
        if (action === 'backspace') {
          deleteBackward();
        } else if (action === 'clear') {
          clearInput();
        } else if (action === 'enter') {
          pressEnter();
        } else if (action === 'shift') {
          isShift = !isShift;
          renderKeys();
        }
      };
    });
  }

  function insertText(str) {
    if (!activeInput) return;
    const el = activeInput;

    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? el.value.length;
    const val = el.value || '';

    // Decimal guard: if already has dot and str is '.', don't add second dot
    if (str === '.' && val.includes('.') && isNumericInput(el)) {
      return;
    }

    const newVal = val.slice(0, start) + str + val.slice(end);
    el.value = newVal;

    const newCursor = start + str.length;
    try {
      el.setSelectionRange(newCursor, newCursor);
    } catch (e) {}

    triggerEvents(el);
  }

  function deleteBackward() {
    if (!activeInput) return;
    const el = activeInput;
    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? el.value.length;
    const val = el.value || '';

    if (start === end) {
      if (start > 0) {
        el.value = val.slice(0, start - 1) + val.slice(start);
        try {
          el.setSelectionRange(start - 1, start - 1);
        } catch (e) {}
      }
    } else {
      el.value = val.slice(0, start) + val.slice(end);
      try {
        el.setSelectionRange(start, start);
      } catch (e) {}
    }
    triggerEvents(el);
  }

  function clearInput() {
    if (!activeInput) return;
    activeInput.value = '';
    triggerEvents(activeInput);
  }

  function pressEnter() {
    if (!activeInput) {
      hideKeyboard();
      return;
    }
    const el = activeInput;
    triggerEvents(el);

    // Dispatch keyboard event for Enter
    const enterEvent = new KeyboardEvent('keydown', {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
      which: 13,
      bubbles: true,
      cancelable: true
    });
    el.dispatchEvent(enterEvent);

    const keyUpEvent = new KeyboardEvent('keyup', {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
      which: 13,
      bubbles: true,
      cancelable: true
    });
    el.dispatchEvent(keyUpEvent);

    // Enter closes the virtual keyboard
    hideKeyboard();
    try { el.blur(); } catch (e) {}
  }

  function triggerEvents(el) {
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }

  // Adjust scrollable modal containers so input (e.g. kirim miqdori) is NEVER covered by keyboard
  function adjustScrollContainerForInput(input) {
    if (!input) return;

    // Check modal or parent scroll containers
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
      if (!activeInput) {
        const inputs = Array.from(document.querySelectorAll('input:not([type="hidden"]):not([disabled]):not([readonly]), textarea:not([disabled]):not([readonly])'));
        const usable = inputs.filter(inp => !isExcludedInput(inp) && inp.id !== 'barcode-input');
        if (usable.length > 0) {
          activeInput = usable[0];
        }
      }
      if (activeInput) {
        currentMode = isNumericInput(activeInput) ? 'numpad' : 'qwerty';
      }
      renderKeys();
      showKeyboard();
    }
  }

  function handleInputActivation(el) {
    if (isExcludedInput(el)) return;

    activeInput = el;
    const hint = document.getElementById('vk-target-hint');
    if (hint) {
      hint.innerText = el.placeholder || el.name || el.id || '';
    }

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

    // STRICT USER INTERACTION ONLY:
    // Only user click on an editable input shows the keyboard
    document.addEventListener('click', (e) => {
      const el = e.target;
      if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
        // Barcode input in kassa should NOT force virtual keyboard on hardware scan
        if (el.id === 'barcode-input') {
          // If user clicked manually, they can open keyboard from top bar if desired
          return;
        }
        handleInputActivation(el);
      }
    }, true);

    // Dismiss keyboard on outside touch/click:
    document.addEventListener('pointerdown', (e) => {
      if (!isVisible) return;
      const root = document.getElementById('orzu-virtual-keyboard-root');
      if (root && root.contains(e.target)) return;
      if (activeInput && (e.target === activeInput || activeInput.contains(e.target))) return;
      if (e.target && (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA')) return;
      if (e.target && e.target.closest && e.target.closest('button[onclick*="VirtualKeyboard"]')) return;

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
