'use strict';
(() => {
  const family = document.body.dataset.walletFamily;
  const language = document.getElementById('language-select');
  const theme = document.getElementById('theme-toggle');
  language.value = window.FlowI18n.locale;
  language.addEventListener('change', () => window.FlowI18n.setLocale(language.value));
  function themeLabel() {
    const dark = document.documentElement.dataset.theme === 'dark';
    const label = window.FlowI18n.t(dark ? '淺色模式' : '暗黑模式');
    theme.setAttribute('aria-pressed', String(dark));
    theme.setAttribute('aria-label', label);
    document.getElementById('theme-label').textContent = label;
  }
  theme.addEventListener('click', () => {
    const next = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.dataset.theme = next;
    try { localStorage.setItem('multi-chain-wallet:theme', next); } catch { /* Session-only preference. */ }
    themeLabel();
  });
  window.addEventListener('localechange', themeLabel);
  themeLabel();
  if (family === 'showcase') return;
  const sidebar = document.getElementById('app-sidebar');
  const menu = document.getElementById('menu-toggle');
  const backdrop = document.getElementById('menu-backdrop');
  function closeMenu(restoreFocus = false) {
    document.body.classList.remove('menu-open');
    document.querySelector('main').inert = false;
    menu.setAttribute('aria-expanded', 'false');
    backdrop.hidden = true;
    if (restoreFocus) menu.focus();
  }
  menu.addEventListener('click', () => {
    if (document.body.classList.contains('menu-open')) return closeMenu(true);
    document.body.classList.add('menu-open');
    menu.setAttribute('aria-expanded', 'true');
    backdrop.hidden = false;
    document.querySelector('main').inert = true;
    requestAnimationFrame(() => document.getElementById('menu-close').focus());
  });
  backdrop.addEventListener('click', () => closeMenu(true));
  document.getElementById('menu-close').addEventListener('click', () => closeMenu(true));
  document.addEventListener('keydown', event => {
    if (!document.body.classList.contains('menu-open')) return;
    if (event.key === 'Escape') { event.preventDefault(); closeMenu(true); }
    if (event.key === 'Tab') {
      const items = [...sidebar.querySelectorAll('a,summary,button')].filter(el => el.getClientRects().length);
      if (event.shiftKey && document.activeElement === items[0]) { event.preventDefault(); items.at(-1).focus(); }
      else if (!event.shiftKey && document.activeElement === items.at(-1)) { event.preventDefault(); items[0].focus(); }
    }
  });
  matchMedia('(min-width: 901px)').addEventListener('change', event => {
    if (event.matches) closeMenu();
  });
  const views = [...document.querySelectorAll('[data-view]')];
  const titles = {overview:'總覽', 'send-panel':'發送資產', 'receive-panel':'收款', 'exchange-panel':'資產兌換', 'test-funding-panel':'領取測試幣', 'history-panel':'活動', 'activity-panel':'活動', 'settings-panel':'設定與備份', 'contacts-panel':'地址簿', 'watch-panel':'唯讀觀察', 'diagnostics-panel':'交易診斷', 'balance-panel':'地址餘額', 'transaction-panel':'交易查核', 'first-transaction':'操作指南', 'tokens-panel':'我的代幣', 'vault-panel':'智慧合約'};
  const descriptions = {
    overview: '查看餘額，或選擇下一步操作。',
    'send-panel': '填入收款地址與數量，下一步核對費用。',
    'receive-panel': '分享地址，請確認對方使用相同測試網路。',
    'exchange-panel': '選擇支付資產與數量，逐步完成鏈上兌換。',
    'test-funding-panel': '補充測試餘額，開始體驗收付款。',
    'history-panel': '查看交易狀態，或切換收支明細核對資產變動。',
    'activity-panel': '核對收付款與手續費，匯出需要的紀錄。',
    'settings-panel': '管理加密備份與錢包密碼。',
    'contacts-panel': '儲存常用收款地址，下次轉帳直接選用。',
    'vault-panel': '練習存入、取回測試 ETH，或用測試 USDC 付款。'
  };
  function navigate(focus = false) {
    let current = location.hash.slice(1) || 'overview';
    if (family === 'evm' && /\/net\//.test(location.pathname) && ['exchange-panel','first-transaction','vault-panel'].includes(current)) current = 'overview';
    if (!titles[current] || !views.some(el => el.dataset.view.split(' ').includes(current))) current = 'overview';
    document.body.dataset.view = current;
    views.forEach(el => { el.dataset.viewHidden = String(!el.dataset.view.split(' ').includes(current)); });
    document.getElementById('page-title').textContent = titles[current];
    const description = document.getElementById('page-description');
    description.textContent = family !== 'evm' && current === 'history-panel' ? '追蹤送出的交易，查看結果與處理進度。' : descriptions[current] || '';
    description.hidden = !descriptions[current];
    for (const link of sidebar.querySelectorAll('a[href^="#"]')) {
      const active = link.hash === '#' + (current === 'activity-panel' ? 'history-panel' : current);
      link.classList.toggle('active', active);
      if (active) { link.setAttribute('aria-current', 'page'); const details = link.closest('details'); if (details) details.open = true; }
      else link.removeAttribute('aria-current');
    }
    for (const link of document.querySelectorAll('.activity-toolbar nav a')) {
      if (link.hash === '#' + current) link.setAttribute('aria-current', 'page');
      else link.removeAttribute('aria-current');
    }
    closeMenu();
    if (focus) { const title = document.getElementById('page-title'); title.tabIndex = -1; title.focus({preventScroll:true}); window.scrollTo(0,0); }
    window.FlowI18n.refresh();
    window.dispatchEvent(new CustomEvent('wallet-view', {detail: current}));
  }
  sidebar.addEventListener('click', event => {
    const link = event.target.closest('a[href^="#"]');
    if (link && link.hash === (location.hash || '#overview')) navigate(true);
  });
  window.addEventListener('hashchange', () => navigate(true));
  navigate();
  if (family !== 'evm') {
    const network = document.getElementById('network-select');
    network.value = '/' + family + '/';
    network.addEventListener('change', () => { location.href = network.value + (location.hash || ''); });
  }
  // Programmatic tutorial actions must reveal their target before focusing it.
  for (const id of ['prepare-self-transfer','prepare-first-wrap','sol-self','tron-self']) {
    document.getElementById(id)?.addEventListener('click', () => { location.hash = id === 'prepare-first-wrap' ? 'exchange-panel' : 'send-panel'; });
  }
})();
