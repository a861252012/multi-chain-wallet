'use strict';
(() => {
  const $ = id => document.getElementById(id);
  const family = document.body.dataset.walletFamily;
  const chain = family === 'evm' ? networkID : family;
  const key = 'multi-chain-wallet:contacts:' + chain;
  const recipient = $(family === 'evm' ? 'send-to' : family === 'solana' ? 'sol-to' : 'tron-to');
  const valid = value => family === 'evm'
    ? /^0x[0-9a-fA-F]{40}$/.test(value) && !/^0x0{40}$/i.test(value)
    : family === 'solana' ? /^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(value) : /^T[1-9A-HJ-NP-Za-km-z]{33}$/.test(value);
  const normalize = value => family === 'evm' ? value.toLowerCase() : value;
  let contacts = [], accounts = [], removed = null;
  try {
    const saved = JSON.parse(localStorage.getItem(key) || '[]');
    if (Array.isArray(saved)) contacts = saved.filter(item => item && typeof item.label === 'string' && valid(item.address)).slice(0,100);
  } catch { /* A blocked or empty store starts with an empty address book. */ }
  function save(next) {
    localStorage.setItem(key, JSON.stringify(next));
    contacts = next;
    render();
  }
  function element(tag, text, className = '') {
    const el = document.createElement(tag); el.textContent = text; el.className = className; return el;
  }
  function renderRecipients() {
    const query = $('recipient-search').value.trim().toLowerCase();
    $('contact-select').replaceChildren(new Option(window.FlowI18n?.t('選擇收款人') || '選擇收款人',''));
    for (const [title, items] of [['我的錢包',accounts],['常用地址',contacts]]) {
      const group = document.createElement('optgroup'); group.label = window.FlowI18n?.t(title) || title;
      for (const item of items.filter(item => (item.label + item.address).toLowerCase().includes(query))) {
        const option = new Option(`${item.label} · ${item.address}`,item.address); option.translate = false; group.append(option);
      }
      if (group.children.length) $('contact-select').append(group);
    }
  }
  function render() {
    renderRecipients();
    $('contacts-list').replaceChildren();
    const query = $('contacts-search').value.trim().toLowerCase();
    for (const item of contacts.filter(item => (item.label + item.address).toLowerCase().includes(query))) {
      const row = element('article','','contact-row');
      const name = element('strong',item.label); name.translate = false;
      const address = element('p',item.address.slice(0,8) + '…' + item.address.slice(-6),'mono'); address.translate = false; address.title = item.address;
      const actions = element('div','','feature-actions');
      for (const [label, run] of [
        ['發送資產', () => { recipient.value = item.address; location.hash = 'send-panel'; recipient.focus(); }],
        ['複製地址', async () => { await navigator.clipboard.writeText(item.address); $('contacts-feedback').textContent = '已複製地址。'; }],
        ['編輯名稱', () => { $('contact-label').value = item.label; $('contact-address').value = item.address; $('contact-label').focus(); }],
        ['移除', () => { save(contacts.filter(c => normalize(c.address) !== normalize(item.address))); removed = item; $('undo-contact').hidden = false; $('contacts-feedback').textContent = '已移除，可復原。'; }]
      ]) {
        const button = element('button',label,'secondary'); button.type = 'button';
        button.addEventListener('click', async () => { try { await run(); } catch { $('contacts-feedback').textContent = '操作未完成，請確認瀏覽器權限後重試。'; } }); actions.append(button);
      }
      row.append(name,address,actions); $('contacts-list').append(row);
    }
    if (!$('contacts-list').children.length) $('contacts-list').append(element('p','尚無符合的常用地址。'));
    window.dispatchEvent(new Event('contacts-updated'));
  }
  window.walletAddressLabel = value => contacts.find(c => normalize(c.address) === normalize(value || ''))?.label || '';
  $('contact-form').addEventListener('submit', event => {
    event.preventDefault();
    try {
      const address = $('contact-address').value.trim(), label = $('contact-label').value.trim();
      if (!valid(address)) throw new Error('請輸入目前網路的完整有效地址；送出交易前仍會查核。');
      if (!label || Array.from(label).length > 40) throw new Error('名稱需為 1–40 字元。');
      const next = contacts.filter(item => normalize(item.address) !== normalize(address));
      if (next.length >= 100) throw new Error('地址簿最多 100 筆。');
      save([...next, {address,label}]); $('contacts-feedback').textContent = '已儲存。';
      $('contact-form').reset();
    } catch (error) { $('contacts-feedback').textContent = error.message; }
  });
  $('undo-contact').addEventListener('click', () => {
    if (!removed) return;
    try {
      if (contacts.some(item => normalize(item.address) === normalize(removed.address))) throw new Error('此地址已有新資料，未覆寫。');
      if (contacts.length >= 100) throw new Error('地址簿最多 100 筆。');
      save([...contacts,removed]); removed = null; $('undo-contact').hidden = true; $('contacts-feedback').textContent = '已復原。';
    } catch (error) { $('contacts-feedback').textContent = error.message; }
  });
  $('save-recipient').addEventListener('click', () => {
    $('contact-address').value = recipient.value.trim(); $('contact-label').value = '';
    location.hash = 'contacts-panel'; $('contact-label').focus();
  });
  $('contact-select').addEventListener('change', () => { if ($('contact-select').value) { recipient.value = $('contact-select').value; recipient.focus(); } });
  $('recipient-search').addEventListener('input',renderRecipients);
  $('contacts-search').addEventListener('input',render);
  window.addEventListener('wallet-accounts',event => {
    accounts = event.detail.filter(item => !item.archived && item.address).map(item => ({label:item.name,address:item.address})); renderRecipients();
  });
  window.addEventListener('localechange',renderRecipients);
  render();
})();
