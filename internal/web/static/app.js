'use strict';
const accountPrefix = location.pathname.match(/^\/accounts\/[a-f0-9]{32}/)?.[0] || '';
const networkSuffix = location.pathname.slice(accountPrefix.length).match(/^\/net\/(arbitrum|base|optimism|polygon)(?:\/|$)/)?.[0].replace(/\/$/, '') || '';
const networkConfig = { '': ['Ethereum Sepolia','https://sepolia.etherscan.io',11155111], '/net/arbitrum': ['Arbitrum Sepolia','https://sepolia.arbiscan.io',421614], '/net/base': ['Base Sepolia','https://sepolia.basescan.org',84532], '/net/optimism': ['OP Sepolia','https://testnet-explorer.optimism.io',11155420], '/net/polygon': ['Polygon Amoy','https://amoy.polygonscan.com',80002] }[networkSuffix];
const networkPrefix = accountPrefix + networkSuffix;
const [networkName, explorerURL, networkID] = networkConfig;
const nativeSymbol = networkID === 80002 ? 'POL' : 'ETH';
const $ = (id) => document.getElementById(id);
const time = (value) => new Date(value).toLocaleString(document.documentElement.lang, { hour12: false });
let copyTimer;

async function request(path) {
  const response = await fetch(networkPrefix + path, { signal: AbortSignal.timeout(12000) });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || '查詢失敗，請稍後重試。');
  return data;
}

function node(tag, text, className) {
  const element = document.createElement(tag);
  element.textContent = text;
  if (className) element.className = className;
  return element;
}

function details(entries) {
  const list = node('dl', '', 'detail-grid');
  for (const [label, value] of entries) {
    const row = document.createElement('div');
    if (String(value).length > 24) row.className = 'detail-wide';
    row.append(node('dt', label), node('dd', value));
    list.append(row);
  }
  return list;
}

function explorer(kind, value) {
  const link = node('a', '在 Etherscan 核對 ↗', 'explorer');
  link.href = `${explorerURL}/${kind}/${encodeURIComponent(value)}`;
  link.target = '_blank';
  link.rel = 'noopener noreferrer';
  return link;
}

function resultActions(kind, value) {
  const actions = node('div', '', 'result-actions');
  const copy = node('button', kind === 'tx' ? '複製交易雜湊' : '複製地址', 'secondary');
  copy.type = 'button';
  copy.addEventListener('click', async () => {
    copy.disabled = true;
    try {
      await navigator.clipboard.writeText(value);
      $('copy-status').textContent = '已複製到剪貼簿';
    } catch {
      $('copy-status').textContent = '無法存取剪貼簿，請選取完整文字後手動複製。';
    } finally {
      copy.disabled = false;
      $('copy-status').hidden = false;
      clearTimeout(copyTimer);
      copyTimer = setTimeout(() => { $('copy-status').hidden = true; }, 5000);
    }
  });
  actions.append(explorer(kind, value), copy);
  return actions;
}

function errorMessage(error) {
  if (error.name === 'TimeoutError') return '查詢逾時，結果未知。請稍後重試。';
  if (error instanceof TypeError) return '無法連線，請確認網路後重試。';
  return error.message;
}

function validate(input) {
  input.value = input.value.trim();
  const valid = input.checkValidity();
  const error = $(`${input.id}-error`);
  error.hidden = valid;
  input.setAttribute('aria-invalid', String(!valid));
  if (!valid) error.textContent = input.id === 'address'
    ? '請輸入完整地址：0x 加上 40 個十六進位字元（0–9、a–f）。'
    : '請輸入完整交易雜湊：0x 加上 64 個十六進位字元（0–9、a–f）。';
  return valid;
}

function showError(result, error, form) {
  result.className = 'result error';
  const retry = node('button', '重新查詢', 'secondary');
  retry.type = 'button';
  retry.addEventListener('click', () => form.requestSubmit());
  result.replaceChildren(node('p', errorMessage(error)), retry);
}

async function refreshNetwork() {
  const button = $('refresh-network');
  if (button.disabled) return;
  button.disabled = true;
  button.querySelector('span').textContent = '更新中…';
  $('network-metrics').setAttribute('aria-busy', 'true');
  $('network-error').hidden = true;
  $('network-state').textContent = '查詢中…';
  $('network-state').dataset.state = 'loading';
  try {
    const data = await request('/api/network');
    $('chain-id').textContent = String(data.chainId);
    $('block-number').textContent = BigInt(data.block).toLocaleString('en-US');
    $('block-time').textContent = `區塊時間 ${time(data.blockTime)}`;
    $('network-state').textContent = '已連線';
    $('network-state').dataset.state = 'success';
    $('checked-at').textContent = `最後查核 ${time(data.checkedAt)}`;
  } catch (error) {
    $('chain-id').textContent = '未驗證';
    $('block-number').textContent = '—';
    $('block-time').textContent = '未取得最新區塊';
    $('network-state').textContent = '無法連線';
    $('network-state').dataset.state = 'error';
    $('checked-at').textContent = '目前狀態未知';
    $('network-error').textContent = errorMessage(error);
    $('network-error').hidden = false;
  } finally {
    button.disabled = false;
    button.querySelector('span').textContent = '更新網路';
    $('network-metrics').setAttribute('aria-busy', 'false');
  }
}

$('balance-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  if (button.disabled) return;
  const input = $('address');
  if (!validate(input)) { input.focus(); return; }
  const result = $('balance-result');
  button.disabled = true;
  button.querySelector('span').textContent = '查詢中…';
  input.readOnly = true;
  result.className = 'result loading';
  result.replaceChildren(node('p', `正在讀取 ${networkName} 餘額…`, 'loading-label'));
  try {
    const data = await request(`/api/balance?address=${encodeURIComponent(input.value)}`);
    const amount = node('div', data.eth, 'balance-value mono');
    amount.append(node('span', nativeSymbol));
    if (document.body.dataset.public === 'true') {
      $('wallet-balance').replaceChildren(node('span', data.eth), node('small', nativeSymbol));
      $('wallet-balance-time').textContent = `公開地址查核 · ${time(data.checkedAt)}`;
      $('account-select').options[0].textContent = data.address;
      $('wallet-address').textContent = data.address;
      $('wallet-explorer').href = `${explorerURL}/address/${encodeURIComponent(data.address)}`;
    }
    result.className = 'result loaded';
    result.replaceChildren(amount, node('p', data.address, 'mono'), details([
      ['完整數值', `${data.wei} wei`], ['查詢區塊', data.block], ['查核時間', time(data.checkedAt)],
    ]), resultActions('address', data.address), node('p', '餘額以本次查詢的區塊為準。重新查詢可取得最新數值。', 'note'));
  } catch (error) {
    showError(result, error, form);
  } finally {
    button.disabled = false;
    button.querySelector('span').textContent = '查詢餘額';
    input.readOnly = false;
  }
});

$('transaction-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('button[type="submit"]');
  if (button.disabled) return;
  const input = $('hash');
  if (!validate(input)) { input.focus(); return; }
  const result = $('transaction-result');
  button.disabled = true;
  button.querySelector('span').textContent = '查詢中…';
  input.readOnly = true;
  result.className = 'result loading';
  result.replaceChildren(node('p', '正在取得交易收據…', 'loading-label'));
  try {
    const data = await request(`/api/transactions/${encodeURIComponent(input.value)}`);
    const labels = { succeeded: '鏈上執行成功', reverted: '鏈上執行失敗', pending: '等待區塊收錄', receipt_unavailable: '收據暫時不可用', reorg_detected: '區塊已變更，結果待確認' };
    const tones = { succeeded: '', reverted: ' danger' };
    result.className = 'result loaded';
    result.replaceChildren(node('span', labels[data.state] || '結果未知', `state-badge${tones[data.state] ?? ' warning'}`), node('p', data.hash, 'mono'));
    if (data.state === 'succeeded' || data.state === 'reverted') result.append(details([
      ['區塊高度', data.block], ['確認數', data.confirmations], ['消耗 gas', data.gasUsed],
      ['實際手續費', `${data.feeEth} ${nativeSymbol}`], ['查核時間', time(data.checkedAt)],
    ]));
    else result.append(node('p', `查核時間 ${time(data.checkedAt)}。尚無可確認的執行結果，請稍後重新查詢。`));
    result.append(resultActions('tx', data.hash), node('p', '確認數不等於最終確定（finality）。這是本次查核的快照；重新查詢可更新結果。', 'note'));
  } catch (error) {
    showError(result, error, form);
  } finally {
    button.disabled = false;
    button.querySelector('span').textContent = '查詢交易';
    input.readOnly = false;
  }
});

for (const id of ['address', 'hash']) {
  $(id).addEventListener('blur', () => { if ($(id).value || $(id).getAttribute('aria-invalid') === 'true') validate($(id)); });
  $(id).addEventListener('input', () => {
    $(id).removeAttribute('aria-invalid');
    $(`${id}-error`).hidden = true;
  });
}

$('refresh-network').addEventListener('click', refreshNetwork);
refreshNetwork();

$('network-select').value = networkSuffix;
$('network-select').addEventListener('change', () => { location.href = (['/solana','/tron'].includes($('network-select').value) ? '' : accountPrefix) + $('network-select').value + '/' + (location.hash || ''); });
for (const element of document.querySelectorAll('[data-network-name]')) element.textContent = networkName;

document.querySelector('.brand').href = networkPrefix + '/';
document.title = `Multi-Chain Wallet · ${networkName} 錢包`;
