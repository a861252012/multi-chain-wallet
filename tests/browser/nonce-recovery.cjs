// Local history fixtures only; never broadcasts or reads a real wallet.
const assert = require('node:assert/strict');
module.exports = async function nonceRecovery(context, base) {
  const page = await context.newPage(), errors = [];
  page.on('pageerror', error => errors.push(String(error)));
  const tx = {hash:'0x'+'a'.repeat(64), quoteId:'consumed-nonce', action:'wrap',
    to:'0x'+'1'.repeat(40), amount:'0.001', symbol:'ETH', state:'submitted', nonceConsumed:true};
  let allowed = true, failed = false, writes = 0;
  await page.route(base+'/api/wallet/history', route => route.fulfill({
    status:failed ? 502 : 200,
    json:failed ? {error:'fixture unavailable'} : {transactions:[tx],...(allowed ? {canCreateTransaction:true} : {})},
  }));
  await page.route(/\/api\/wallet\/(send|retry|quote)$/, route => { writes++; return route.fulfill({status:500,json:{error:'unexpected write'}}); });
  async function view(id) {
    await page.evaluate(id => { location.hash=id; },id);
    await page.waitForFunction(id => document.body.dataset.view===id,id);
    await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled && !document.querySelector('#activity-refresh').disabled);
  }
  const full = '原交易結果無法確認；該 nonce 已在 finalized 狀態被消耗，目前可建立新交易。請先核對原付款，避免重複支付。';
  try {
    await page.goto(base);
    await page.locator('#wallet-dashboard').waitFor({state:'visible'});
    await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
    await page.evaluate(()=>window.FlowI18n.setLocale('zh-TW'));
    await view('history-panel');
    const row = page.locator('#wallet-history .history-row');
    assert.equal(await row.locator('.state-badge').textContent(),'原交易結果無法確認');
    assert.ok((await row.textContent()).includes(full));
    for (const name of ['重新廣播原交易','加速交易','取消交易']) assert.equal(await row.getByRole('button',{name,exact:true}).count(),0);
    await row.locator('details > summary').click();
    assert.equal(await row.getByRole('button',{name:'交易詳情',exact:true}).count(),1);
    await page.evaluate(tx=>sessionStorage.setItem('multi-chain-wallet:exchange-flow:',JSON.stringify({
      id:'nonce-flow',direction:'eth-usdc',amount:'0.001',phase:'wrap',pending:{hash:tx.hash,quoteID:tx.quoteId,kind:'wrap'},
    })),tx);
    await page.reload();
    await page.waitForFunction(()=>!document.querySelector('#wallet-dashboard').hidden && !document.querySelector('#refresh-wallet').disabled);
    await view('exchange-panel');
    await page.locator('#exchange-submit').click();
    await page.waitForFunction(()=>!document.querySelector('#exchange-submit').disabled);
    assert.equal(await page.locator('#exchange-workflow').textContent(),full);
    const flow = await page.evaluate(()=>JSON.parse(sessionStorage.getItem('multi-chain-wallet:exchange-flow:')));
    assert.equal(flow.pending.hash,tx.hash); assert.equal(flow.phase,'wrap');
    assert.equal(writes,0,'nonce recovery must not retry or advance the payment');
    allowed=false;
    await view('overview'); await page.locator('#refresh-wallet').click();
    await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
    await view('history-panel');
    assert.ok(!(await row.textContent()).includes('目前可建立新交易'),'another pending tx still blocks');
    assert.equal(await row.locator('.state-badge').textContent(),'原交易結果無法確認');
    failed=true;
    await view('overview'); await page.locator('#refresh-wallet').click();
    await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
    await view('history-panel');
    assert.ok(!(await row.textContent()).includes('目前可建立新交易'),'failed refresh cannot advertise permission');
    assert.deepEqual(errors,[]);
    console.log('PASS: consumed nonce remains unknown, preserves lookup/flow, warns about duplicate payment and never auto-retries.');
  } finally { await page.close(); }
};
