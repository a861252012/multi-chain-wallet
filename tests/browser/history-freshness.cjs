// Local fixture coverage for cached history, HTTP failures and flow reconciliation.
const assert = require('node:assert/strict');

module.exports = async function historyFreshness(context, base) {
  const page = await context.newPage(), errors = [];
  page.on('pageerror', error => errors.push(String(error)));
  const tx = {hash: '0x'+'c'.repeat(64), quoteId: 'history-fixture', action: 'wrap',
    to: '0x'+'1'.repeat(40), amount: '0.001', symbol: 'ETH', state: 'succeeded'};
  const stale = 'fixture history RPC unavailable';
  const failed = '無法更新紀錄，請稍後再試。';
  let mode = 'stale', transactions = [tx];
  await page.route(base+'/api/wallet/history', route => route.fulfill({
    status: mode === 'failed' ? 502 : 200,
    json: mode === 'failed' ? {error: 'fixture HTTP failure'}
      : {transactions, ...(mode === 'stale' ? {refreshError: stale} : {})},
  }));
  await page.route(base+'/api/transactions/'+tx.hash, route => route.fulfill({json: {hash: tx.hash, state: 'pending'}}));
  const warning = page.locator('#wallet-history > .error');
  const rows = page.locator('#wallet-history .history-row');
  async function view(id) {
    await page.evaluate(id => { location.hash = id; }, id);
    await page.waitForFunction(id => document.body.dataset.view === id, id);
    await page.waitForFunction(() => !document.querySelector('#refresh-wallet').disabled && !document.querySelector('#activity-refresh').disabled);
  }
  async function refresh() {
    await view('overview');
    await page.locator('#refresh-wallet').click();
    await page.waitForFunction(() => !document.querySelector('#refresh-wallet').disabled);
    await view('history-panel');
  }
  async function preservesWarning(message) {
    assert.equal(await warning.textContent(), message);
    await page.locator('#history-search').fill('no-matching-history');
    assert.equal(await rows.count(), 0);
    assert.equal(await warning.textContent(), message, 'filter preserves stale status even with no matches');
    await page.evaluate(() => window.dispatchEvent(new Event('contacts-updated')));
    assert.equal(await warning.textContent(), message, 'contact changes preserve stale status');
    await page.locator('#history-search').fill('');
    assert.equal(await rows.count(), transactions.length);
    assert.equal(await warning.textContent(), message, 'clearing the filter does not imply a successful refresh');
  }
  try {
    await page.goto(base);
    await page.locator('#wallet-dashboard').waitFor({state: 'visible'});
    await page.waitForFunction(() => !document.querySelector('#refresh-wallet').disabled);
    await page.evaluate(() => window.FlowI18n.setLocale('zh-TW'));
    await view('history-panel');
    await preservesWarning(stale);

    mode = 'failed';
    await refresh();
    await preservesWarning(failed);

    mode = 'fresh';
    transactions = [{...tx, amount: '0.002'}];
    await refresh();
    assert.equal(await warning.count(), 0, 'successful RPC refresh clears stale status');
    assert.match(await rows.textContent(), /0\.002 ETH/, 'fresh response replaces cached rows');

    mode = 'stale'; transactions = [];
    await refresh();
    await preservesWarning(stale);
    mode = 'fresh'; await refresh();
    assert.equal(await warning.count(), 0, 'successful empty history clears stale status');
    assert.equal(await rows.count(), 0);

    // The exchange progress button fetches history independently of refreshWallet.
    transactions = [tx];
    await page.evaluate(tx => sessionStorage.setItem('multi-chain-wallet:exchange-flow:', JSON.stringify({
      id: 'history-flow', direction: 'eth-usdc', amount: '0.001', phase: 'wrap',
      pending: {hash: tx.hash, quoteID: tx.quoteId, kind: 'wrap'},
    })), tx);
    await page.reload();
    await page.waitForFunction(() => !document.querySelector('#wallet-dashboard').hidden && !document.querySelector('#refresh-wallet').disabled);
    for (const [nextMode, message] of [['stale', stale], ['failed', failed], ['fresh', '']]) {
      mode = nextMode;
      await view('exchange-panel');
      await page.locator('#exchange-submit').click();
      await page.waitForFunction(() => !document.querySelector('#exchange-submit').disabled);
      // Assert before opening history: navigation itself also refreshes history.
      if (message) assert.equal(await warning.textContent(), message, 'flow reconciliation records its own refresh error');
      else assert.equal(await warning.count(), 0, 'successful flow refresh clears stale status');
      await view('history-panel');
      if (message) await preservesWarning(message);
      else assert.equal(await warning.count(), 0, 'successful flow refresh clears stale status');
    }
    assert.deepEqual(errors, []);
    console.log('PASS: history refresh warnings survive search/contact changes, HTTP failures, empty rows and exchange reconciliation until a fresh RPC response.');
  } finally {
    await page.close();
  }
};
