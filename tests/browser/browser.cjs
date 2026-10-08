// UI contract tests. Every API response is a local fixture; no real wallet or RPC is accessed.
const { chromium } = require('playwright');
const http = require('node:http');
const fs = require('node:fs/promises');
const path = require('node:path');
const os = require('node:os');
const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const root = path.resolve(__dirname, '../..');
// Render once with Go's production template engine; API responses remain local fixtures.
const pages = JSON.parse(execFileSync('go', ['run', './tests/browser/render-templates'], {
 cwd: root, encoding: 'utf8', maxBuffer: 4 * 1024 * 1024,
 env: {...process.env, GOTOOLCHAIN: 'local', GOPROXY: 'off', GOSUMDB: 'off'},
}));
const address = '0x1111111111111111111111111111111111111111';
const weth = '0xfff9976782d46cc05630d1f6ebab18b2324d6b14';
const usdc = '0x1c7d4b196cb0c7b01d743fbc6116a902379c7238';
const customToken = '0x4444444444444444444444444444444444444444';
const router = '0x3bfa4769fb09eefc5a80d6e87c3b9c650f7ae48e';
const vaultContract = '0x5555555555555555555555555555555555555555';
let vaultEnabled = true, vaultFailure = false, vaultQuoteFailure = false, vaultDeposit = 20000000000000000n, walletWei = 1000000000000000000n;
let accountFixtures = [{id:'',name:'主要錢包',address,archived:false}];
let accountWrites = 0, accountFailure = false;
let walletCSRF='fixture', solCSRF='fixture', tronCSRF='fixture';
let enforceTokenLimit=false, activeTokenQueries=0, tokenLimitHits=0;
let rejectSendStatus = 0, scanTokens = [], syncRequests = [], syncFailureBatch = 0;
const transactionStatuses = new Map();
let exists = true, balanceFailure = false, loseSendResponse = false, history = [], sent = [], quotes = new Map(), allowances = new Map();
let tronExists=false,tronHistory=[],tronSends=0;const tronAddress='TUEZSdKsoDHQMeZwihtdoBiN46zxhGWYdH';
let tronTokenSymbol='TEST';
let solExists = false, solHistory = [], solSends = 0;
const solAddress = 'CghaUzuHcKaSKQG3UtoZzMafEPmnRk1V6LUN2Hdp8XWa';
const networks = {base:84532,optimism:11155420,arbitrum:421614,polygon:80002};
const server = http.createServer(async (req,res)=>{
 try {
  const url = new URL(req.url,'http://localhost');
  const chainId = networks[url.pathname.match(/^\/net\/([^/]+)/)?.[1]] || 11155111;
  const accountID = url.pathname.match(/^\/accounts\/([^/]+)/)?.[1] || '';
  const pathname = url.pathname.replace(/^\/accounts\/[^/]+/,'').replace(/^\/net\/[^/]+/,'');
  res.setHeader('Content-Type','application/json');
  let body='';for await(const chunk of req)body+=chunk;body=body?JSON.parse(body):{};
  const respond = data => res.end(JSON.stringify(data));
  if(req.method==='POST'&&(pathname.startsWith('/solana/api/')||pathname.startsWith('/tron/api/'))){
   const csrf=pathname.startsWith('/solana/')?solCSRF:tronCSRF;
   if(req.headers['x-wallet-csrf']!==csrf){res.statusCode=403;return respond({error:'fixture stale CSRF token'});}
  }

  if(pathname==='/showcase'){res.setHeader('Content-Type','text/html');return res.end(pages.showcase);}
  if(pathname==='/tron/'){res.setHeader('Content-Type','text/html');return res.end(pages['tron-'+(url.searchParams.has('shared')?'shared':'private')]);}
  if(pathname==='/tron/api/status')return respond({exists:tronExists,address:tronAddress,csrfToken:tronCSRF});
  if(pathname==='/tron/api/create'){tronExists=true;return respond({address:tronAddress});}
  if(pathname==='/tron/api/balance')return respond({trx:'100',active:true,bandwidth:600,energy:0});
  if(pathname==='/tron/api/history')return respond(tronHistory);
  if(pathname==='/tron/api/token')return respond({contract:tronAddress,symbol:tronTokenSymbol,balance:'1',decimals:6});
  if(pathname==='/tron/api/quote')return respond({...body,id:'tron-quote',amount:body.amount||body.amountRaw,symbol:body.contract?tronTokenSymbol:'TRX',feeTrx:'0.3',feeLimitTrx:'0',energy:0,bandwidth:300,expiresAt:new Date(Date.now()+60000).toISOString()});
  if(pathname==='/tron/api/send'){tronSends += 1;tronHistory=[{signature:'1'.repeat(64),to:tronAddress,amount:'0.000001',symbol:'TRX',feeTrx:'0.001',state:'finalized',finalized:true}];return respond(tronHistory[0]);}
  if(pathname==='/solana/'){res.setHeader('Content-Type','text/html');return res.end(pages['solana-'+(url.searchParams.has('shared')?'shared':'private')]);}
  if(pathname==='/solana/api/status')return respond({exists:solExists,address:solAddress,csrfToken:solCSRF});
  if(pathname==='/solana/api/create'){solExists=true;return respond({address:solAddress});}
  if(pathname==='/solana/api/balance'){if(balanceFailure){res.statusCode=502;return respond({error:'fixture Devnet unavailable'});}return respond({sol:'0.1',slot:100});}
  if(pathname==='/solana/api/history')return respond(solHistory);
  if(pathname==='/solana/api/quote')return respond({...body,id:'sol-quote',feeSol:'0.000005',lastValidBlockHeight:200,expiresAt:new Date(Date.now()+60000).toISOString()});
  if(pathname==='/solana/api/send'){solSends += 1;solHistory=[{signature:'1'.repeat(88),to:solAddress,amount:'0.000001',state:'finalized',finalized:true}];return respond(solHistory[0]);}
  if(url.pathname.startsWith('/static/')){res.setHeader('Content-Type',url.pathname.endsWith('.css')?'text/css':'text/javascript');return res.end(await fs.readFile(path.join(root,'internal/web',url.pathname)));}
  if(pathname==='/' || pathname==='' || pathname==='/public-demo' || pathname==='/shared-demo'){res.setHeader('Content-Type','text/html');return res.end(pages['index-'+(pathname==='/public-demo'?'public':pathname==='/shared-demo'?'shared':'private')+'-'+(chainId===80002?'POL':'ETH')]);}
  if(pathname==='/api/faucet' && req.method==='GET')return respond({enabled:true,csrfToken:'fixture'});
  if(pathname==='/api/faucet' && req.method==='POST'){
   assert.equal(req.headers['x-wallet-csrf'],'fixture'); assert.equal(body.address,address);
   assert.ok(['native','usdc'].includes(body.asset));
   return respond({hash:'0x'+'f'.repeat(64),state:'submitted'});
  }
  if(pathname==='/api/faucet/tron'){res.statusCode=400;return respond({error:'Shasta 測試幣庫存不足'});}
  if(pathname==='/api/faucet/solana'){res.statusCode=502;return respond({error:'Devnet 空投限流，請稍後再試'});}
  if(pathname==='/api/network')return respond({chainId,block:'100',blockTime:new Date().toISOString(),checkedAt:new Date().toISOString()});
  if(pathname==='/api/wallet')return respond({exists:accountID ? Boolean(accountFixtures.find(item=>item.id===accountID)?.address) : exists,address,chainId,csrfToken:walletCSRF,exchange:chainId===11155111?{weth,usdc,router}:{}});
  if(pathname==='/api/wallet/accounts' && req.method==='GET')return respond(accountFixtures);
  if(pathname==='/api/wallet/vault'){
   if(vaultFailure){res.statusCode=502;return respond({error:'fixture vault unavailable'});}
   const enabled=vaultEnabled&&chainId===11155111;
   return respond({enabled,contract:enabled?vaultContract:'',balance:enabled?String(Number(accountID?0n:vaultDeposit)/1e18):'',balanceRaw:enabled?String(accountID?0n:vaultDeposit):''});
  }
  if(pathname==='/api/wallet/accounts' && req.method==='POST'){
   accountWrites++;assert.equal(req.headers['x-wallet-csrf'],'fixture');
   const item={id:'a'.repeat(32),name:body.name,address:'',archived:false};accountFixtures.push(item);return respond(item);
  }
  if(pathname==='/api/wallet/accounts/update'){
   accountWrites++;assert.equal(req.headers['x-wallet-csrf'],'fixture');
   if(accountFailure){res.statusCode=400;return respond({error:'fixture 無法儲存'});}
   const item=accountFixtures.find(item=>item.id===body.id);Object.assign(item,body);return respond(item);
  }
  if(pathname==='/api/balance'){if(balanceFailure){res.statusCode=502;return respond({error:'fixture RPC unavailable'});}return respond({address,eth:String(Number(walletWei)/1e18),wei:String(walletWei),block:'100',checkedAt:new Date().toISOString()});}
  if(pathname==='/api/wallet/history')return respond({transactions:history});
  if(pathname==='/api/wallet/activity')return respond({page:1,pages:1,totalTransactions:0,transactions:[],totals:[]});
  if(pathname==='/api/wallet/scan')return respond({enabled:false,start:90,next:91,finalized:100,tokens:scanTokens});
  if(pathname==='/api/wallet/activity/sync'){
   syncRequests.push(body);
   if(body.contracts.length>20){res.statusCode=400;return respond({error:'每次同步最多 20 個代幣合約'});}
   if(syncFailureBatch===syncRequests.length){res.statusCode=502;return respond({error:'fixture batch unavailable'});}
   return respond({from:body.from||81,to:100,added:1});
  }
  if(pathname==='/api/wallet/token'||pathname==='/api/watch/token'){
   if(pathname==='/api/wallet/token'&&req.headers['x-wallet-csrf']!==walletCSRF){res.statusCode=403;return respond({error:'fixture stale CSRF token'});}
   if(enforceTokenLimit){
    if(activeTokenQueries){tokenLimitHits++;res.statusCode=429;return respond({error:'fixture concurrent POST limit'});}
    activeTokenQueries++;await new Promise(resolve=>setTimeout(resolve,30));activeTokenQueries--;
   }
   const contract=(body.contract||url.searchParams.get('contract')).toLowerCase();
   const custom=contract===customToken,decimals=contract===usdc?6:18;
   return respond({contract,symbol:custom?'CUSTOM':contract===usdc?'USDC':'WETH',decimals,balance:'1',balanceRaw:10n**BigInt(decimals)+'',allowanceRaw:allowances.get(contract)||'0',allowance:'0',trustedMetadata:!custom});
  }
  if(pathname==='/api/wallet/exchange/pools')return respond({bestFee:500,symbol:'USDC',pools:[{fee:100,error:'fixture unavailable'},{fee:500,output:'0.2',outputRaw:'200000'},{fee:3000,output:'0.1',outputRaw:'100000'},{fee:10000,error:'no pool'}]});
  if(pathname==='/api/wallet/quote'){
   if(body.action?.startsWith('vault_')){
    assert.equal(body.to,address);assert.equal(body.contract,undefined);
    if(vaultQuoteFailure){res.statusCode=400;return respond({error:'fixture vault simulation reverted'});}
    const id=String(quotes.size+1);quotes.set(id,body);
    return respond({...body,id,from:address,contract:vaultContract,symbol:'ETH',method:body.action==='vault_deposit'?'deposit()':'withdraw(uint256)',amountRaw:'1000000000000000',maxFeeEth:'0.00001',totalEth:body.action==='vault_withdraw'?'0.00001':'0.00101',expiresAt:new Date(Date.now()+120000).toISOString(),nonce:'0',data:'0x',gasLimit:'60000',maxFeePerGas:'1',maxPriorityFeePerGas:'1'});
   }
   const id=String(quotes.size+1);quotes.set(id,body);
   return respond({...body,id,from:address,symbol:body.action==='wrap'?'ETH':'USDC',amountRaw:'1',maxFeeEth:'0.00001',totalEth:body.amount,expiresAt:new Date(Date.now()+120000).toISOString(),nonce:'0',data:'0x',gasLimit:'21000',maxFeePerGas:'1',maxPriorityFeePerGas:'1'});
  }
  if(pathname==='/api/wallet/send'){
   if(rejectSendStatus){res.statusCode=rejectSendStatus;return respond({error:'fixture rejected before signing',code:'send_rejected'});}
   const q=quotes.get(body.quoteId);assert.ok(q);sent.push(q.action);
   if(q.action==='vault_deposit')vaultDeposit+=1000000000000000n;
   if(q.action==='vault_withdraw')vaultDeposit-=1000000000000000n;
   if(q.action==='approve')allowances.set(q.contract,BigInt(Math.round(Number(q.amount)*10**(q.contract===usdc?6:18))).toString());
   const tx={...q,quoteId:body.quoteId,hash:'0x'+String(sent.length).padStart(64,'0'),state:'succeeded',createdAt:new Date().toISOString(),symbol:'ETH',confirmations:'1'};history.push(tx);if(loseSendResponse){loseSendResponse=false;res.statusCode=502;return respond({error:'fixture proxy lost upstream response after broadcast'});}return respond(tx);
  }
  if(pathname.startsWith('/api/transactions/') && transactionStatuses.has(pathname.split('/').pop()))return respond(transactionStatuses.get(pathname.split('/').pop()));
  if(pathname.startsWith('/api/transactions/'))return respond({hash:pathname.split('/').pop(),state:'succeeded',block:'100',blockHash:'0x'+'1'.repeat(64),finalized:true,confirmations:'2',feeEth:'0.000005',checkedAt:new Date().toISOString()});
  if(pathname==='/api/watch/activity')return respond({state:'succeeded',hash:url.searchParams.get('hash'),movements:[{kind:'receive',asset:weth,raw:'1000000000000',evidence:'fixture Transfer log'}]});
  if(pathname==='/api/diagnostics')return respond({chainId,network:{block:'100'},rpc:{requests:20,transportFailures:1,failovers:1,activeEndpoint:2,endpointCount:2,lastRequestMs:5}});
  res.statusCode=404;respond({error:'unhandled fixture path: '+pathname});
 }catch(error){res.statusCode=500;res.end(JSON.stringify({error:String(error)}));}
});
(async()=>{
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 const base='http://127.0.0.1:'+server.address().port;
 const browser=await chromium.launch({headless:true});
 const context=await browser.newContext({viewport:{width:1280,height:900}});
 await context.route('**/*',route=>new URL(route.request().url()).origin===base?route.continue():route.abort());
 const page=await context.newPage(), errors=[];page.on('pageerror',error=>errors.push(String(error)));
 async function view(id){await page.evaluate(id=>{location.hash=id;},id);await page.waitForFunction(id=>document.body.dataset.view===id,id);}
 try {
  const screenshotRoot = process.env.MULTI_CHAIN_WALLET_UI_SCREENSHOTS || os.tmpdir();
  await fs.mkdir(screenshotRoot,{recursive:true});
  const screenshots = await fs.mkdtemp(path.join(screenshotRoot,'multi-chain-wallet-browser-'));
  console.log('Browser screenshots: '+screenshots);
  const publicRequests = [];
  const trackPublic = req => publicRequests.push(new URL(req.url()).pathname);
  page.on('request', trackPublic);
  await page.goto(base+'/public-demo');
  await page.evaluate(() => {
    const prefix = String.fromCharCode(102,108,111,119,108,101,100,103,101,114)+':';
    localStorage.setItem(prefix+'contacts:evm', JSON.stringify([{address:'0x1111111111111111111111111111111111111111',label:'Saved recipient'}]));
    localStorage.setItem(prefix+'theme', 'dark');
    sessionStorage.setItem(prefix+'exchange-flow:/migration-fixture', JSON.stringify({id:'saved-flow',pending:{quoteID:'saved-quote',hash:'0x123'}}));
  });
  await page.reload();
  const migrated = await page.evaluate(() => ({
    contacts: JSON.parse(localStorage.getItem('multi-chain-wallet:contacts:evm')),
    flow: JSON.parse(sessionStorage.getItem('multi-chain-wallet:exchange-flow:/migration-fixture')),
    theme: document.documentElement.dataset.theme,
  }));
  assert.equal(migrated.contacts[0].label, 'Saved recipient');
  assert.equal(migrated.flow.pending.quoteID, 'saved-quote');
  assert.equal(migrated.theme, 'dark');
  assert.match(await page.title(), /Multi-Chain Wallet/);
  await page.evaluate(() => {
    localStorage.removeItem('multi-chain-wallet:contacts:evm');
    sessionStorage.removeItem('multi-chain-wallet:exchange-flow:/migration-fixture');
    localStorage.removeItem('multi-chain-wallet:theme');
  });
  console.log('PASS: browser namespace migration preserves contacts, preferences and pending quote IDs.');
  assert.equal(await page.locator('a[href="/login"]').count(),1);
  await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  assert.equal(await page.locator('#manage-accounts').isDisabled(),true);
  assert.equal(await page.locator('script[src="/static/wallet.js"]').count(),0);
  await view('send-panel');
  assert.equal(await page.locator('#send-form button[type=submit]').isDisabled(),true);
  await view('vault-panel');
  assert.equal(await page.locator('#vault-status').textContent(),'目前為唯讀展示');
  for (const selector of ['#vault-form','#vault-summary','#vault-history-section']) assert.equal(await page.locator(selector).isVisible(),false,'read-only visitor hides '+selector);
  await page.evaluate(()=>window.scrollTo(0,0));
  await page.screenshot({path:path.join(screenshots,'vault-readonly.png'),fullPage:true,animations:'disabled'});
  await view('balance-panel');
  await page.locator('#address').fill(address);
  await page.locator('#balance-form button[type=submit]').click();
  await page.waitForFunction(()=>document.getElementById('balance-result').textContent.includes('1000000000000000000'));
  await view('overview');
  await page.screenshot({path:path.join(screenshots,'wallet-public-shared-layout.png'),fullPage:true});
  await page.setViewportSize({width:375,height:812});
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'public demo fits mobile');
  await page.setViewportSize({width:1280,height:900});
  page.off('request', trackPublic);
  assert.ok(!publicRequests.some(path=>path.includes('/api/wallet') || path.includes('/api/faucet')), 'visitor never requests private wallet data');
  assert.match(await page.locator('#wallet-balance').textContent(),/1/);
  console.log('PASS: shared public wallet layout, read-only controls, public balance query, no private requests, mobile layout (mock APIs).');
  await page.goto(base+'/shared-demo#vault-panel');
  // Initial wallet loading and vault navigation can both refresh the balances.
  await page.waitForFunction(()=>document.querySelector('#wallet-loading').hidden && !document.querySelector('#refresh-wallet').disabled && !document.querySelector('#vault-preview').disabled);
  assert.equal(await page.locator('#vault-deposited-balance').textContent(),'0.02 ETH');
  assert.equal(await page.locator('#vault-wallet-balance').textContent(),'1 ETH');
  for (const selector of ['#vault-form','#vault-summary','#vault-history-section']) assert.equal(await page.locator(selector).isVisible(),true,'enabled contract shows '+selector);
  assert.equal(await page.locator('#vault-deposit').getAttribute('type'),'radio');
  assert.equal(await page.locator('#vault-withdraw').getAttribute('type'),'radio');
  assert.equal(await page.locator('input[name="vault-action"]:checked').inputValue(),'vault_deposit');
  if (process.env.MULTI_CHAIN_WALLET_UI_SCREENSHOTS) {
    await page.locator('#language-select').selectOption('en');
    if (await page.evaluate(() => document.documentElement.dataset.theme === 'dark')) await page.locator('#theme-toggle').click();
    await view('overview');
    await page.screenshot({path:path.join(screenshots,'wallet-overview.png'),fullPage:true,animations:'disabled'});
    await page.locator('#language-select').selectOption('zh-TW');
    await view('vault-panel');
  }
  const quotesBeforeSelection=quotes.size;
  await page.locator('#vault-deposit').focus();
  await page.keyboard.press('ArrowRight');
  assert.equal(await page.locator('#vault-withdraw').isChecked(),true,'arrow key selects withdrawal');
  assert.equal(await page.locator('#vault-direction').textContent(),'智慧合約 → 目前錢包');
  await page.keyboard.press('ArrowLeft');
  assert.equal(await page.locator('#vault-deposit').isChecked(),true,'arrow key selects deposit');
  assert.equal(await page.locator('#vault-direction').textContent(),'目前錢包 → 智慧合約');
  assert.equal(quotes.size,quotesBeforeSelection,'mode selection does not request a quote');
  assert.equal(await page.locator('#send-confirmation').isVisible(),false);
  await page.locator('#vault-amount').fill('0.001');
  await page.locator('#vault-preview').click();
  await page.locator('#send-confirmation').waitFor({state:'visible'});
  assert.equal(await page.locator('#confirm-title').textContent(),'確認存入合約');
  assert.match(await page.locator('#quote-details').textContent(),/存入合約/);
  assert.match(await page.locator('#quote-details').textContent(),new RegExp('扣款錢包'+address+'智慧合約'+vaultContract));
  assert.doesNotMatch(await page.locator('#quote-details').textContent(),/收款地址/);
  await page.locator('#send-password').fill('fixture-password');
  await page.locator('#confirm-send-button').click();
  await page.waitForFunction(()=>document.querySelector('#vault-deposited-balance').textContent==='0.021 ETH');
  await page.waitForFunction(()=>document.activeElement?.id==='vault-history-section');
  await page.locator('#vault-withdraw').check();
  const quotesBeforeWithdraw=quotes.size;
  await page.locator('#vault-amount').press('Enter');
  await page.locator('#send-confirmation').waitFor({state:'visible'});
  assert.equal(quotes.size,quotesBeforeWithdraw+1,'Enter submits one quote');
  assert.equal([...quotes.values()].at(-1).action,'vault_withdraw','Enter respects selected withdrawal');
  assert.equal(await page.locator('#confirm-title').textContent(),'確認取回錢包');
  assert.match(await page.locator('#quote-details').textContent(),/僅為費用上限/);
  assert.match(await page.locator('#quote-details').textContent(),new RegExp('智慧合約'+vaultContract+'收款錢包'+address));
  await page.locator('#send-password').fill('fixture-password');await page.locator('#confirm-send-button').click();
  await page.waitForFunction(()=>document.querySelector('#vault-deposited-balance').textContent==='0.02 ETH');
  await page.waitForFunction(()=>document.activeElement?.id==='vault-history-section');
  assert.match(await page.locator('#vault-history').textContent(),/鏈上執行成功/);
  for(const locale of ['en','zh-CN','zh-TW']) {
   await page.locator('#language-select').selectOption(locale);
   await page.waitForFunction(()=>document.querySelector('#vault-eth-content h2').textContent===window.FlowI18n.t('ETH 存入與取回'));
   assert.equal(await page.locator('#vault-amount').inputValue(),'0.001','locale switch preserves amount');
   assert.equal(await page.locator('#vault-withdraw').isChecked(),true,'locale switch preserves action');
   assert.equal(await page.locator('#vault-deposited-balance').textContent(),'0.02 ETH');
   await page.locator('#vault-eth-content details summary').click();
   if(locale==='en') assert.ok(!/[\u3400-\u9fff]/.test(await page.locator('#vault-panel').innerText()),'contract English messages are translated');
   await page.locator('#vault-eth-content details summary').click();
   if(locale==='zh-CN') assert.equal(await page.locator('#vault-eth-content h2').textContent(),'ETH 存入与取回');
   await page.setViewportSize({width:375,height:812});
   for(const mode of ['light','dark']) {
    if(await page.getAttribute('html','data-theme')!==mode)await page.locator('#theme-toggle').click();
    assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`contract fits 375 px: ${locale} ${mode}`);
    await page.evaluate(()=>window.scrollTo(0,0));
    await page.screenshot({path:path.join(screenshots,`vault-${locale}-375-${mode}.png`),fullPage:true,animations:'disabled'});
   }
  }
  await page.setViewportSize({width:1280,height:900});
  await page.locator('#theme-toggle').click();
  await page.evaluate(()=>window.scrollTo(0,0));
  await page.screenshot({path:path.join(screenshots,'vault-desktop.png'),fullPage:true,animations:'disabled'});
  await page.locator('#vault-deposit').check();
  vaultQuoteFailure=true;await page.locator('#vault-preview').click();
  await page.waitForFunction(()=>document.querySelector('#vault-error').textContent.includes('simulation reverted'));
  assert.equal(await page.locator('#send-confirmation').isVisible(),false);vaultQuoteFailure=false;
  vaultEnabled=false;balanceFailure=true;await page.locator('#vault-refresh').click();
  await page.waitForFunction(()=>document.querySelector('#vault-status').textContent==='此環境尚未開放合約操作');
  for(const selector of ['#vault-form','#vault-summary','#vault-history-section']) assert.equal(await page.locator(selector).isVisible(),false,'unconfigured contract hides '+selector);
  assert.equal(await page.locator('#vault-error').isVisible(),false,'disabled contract does not depend on wallet balance RPC');
  for(const locale of ['en','zh-CN','zh-TW']) {
   await page.locator('#language-select').selectOption(locale);
   await page.waitForFunction(()=>document.querySelector('#vault-status').textContent===window.FlowI18n.t('此環境尚未開放合約操作'));
   if(locale==='en') assert.ok(!/[\u3400-\u9fff]/.test(await page.locator('#vault-panel').innerText()),'unconfigured contract English messages are translated');
  }
  await page.evaluate(()=>window.scrollTo(0,0));
  await page.screenshot({path:path.join(screenshots,'vault-unconfigured.png'),fullPage:true,animations:'disabled'});
  balanceFailure=false;
  vaultEnabled=true;vaultFailure=true;await page.locator('#vault-refresh').click();
  await page.waitForFunction(()=>document.querySelector('#vault-error').textContent.includes('vault unavailable'));
  assert.equal(await page.locator('#vault-error').isVisible(),true,'RPC error remains visible outside hidden form');
  for(const selector of ['#vault-form','#vault-summary','#vault-history-section']) assert.equal(await page.locator(selector).isVisible(),false,'RPC failure hides stale '+selector);
  vaultFailure=false;vaultDeposit=0n;await page.locator('#vault-refresh').click();
  await page.waitForFunction(()=>!document.querySelector('#vault-preview').disabled);
  assert.equal(await page.locator('#vault-deposited-balance').textContent(),'0 ETH','zero balance is displayed after RPC recovery');
  assert.equal(await page.locator('#vault-form').isVisible(),true,'zero balance still allows a deposit');
  assert.equal(await page.locator('#vault-error').isVisible(),false);
  assert.match(await page.locator('#vault-status').textContent(),/此錢包尚未存入 ETH/);
  walletWei=0n;await page.locator('#vault-refresh').click();
  await page.waitForFunction(()=>document.querySelector('#vault-wallet-balance').textContent==='0 ETH');
  assert.match(await page.locator('#vault-status').textContent(),/請先領取/,'empty wallet explains how to obtain gas');
  assert.equal(await page.locator('.vault-funding').isVisible(),true);
  walletWei=1000000000000000000n;
  vaultDeposit=20000000000000000n;await page.locator('#vault-refresh').click();
  await page.waitForFunction(()=>document.querySelector('#vault-deposited-balance').textContent==='0.02 ETH');
  await page.goto(base+'/net/base/#vault-panel');await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  assert.equal(await page.locator('#nav-vault').isVisible(),false);
  assert.equal(await page.getAttribute('body','data-view'),'overview');
  accountFixtures.push({id:'b'.repeat(32),name:'Vault second account',address,archived:false});
  await page.goto(base+'/#vault-panel');await page.waitForFunction(()=>!document.querySelector('#vault-preview').disabled);
  await page.locator('#vault-amount').fill('0.001');await page.locator('#vault-preview').click();
  await page.locator('#send-confirmation').waitFor({state:'visible'});await page.locator('#cancel-send').click();
  await page.locator('#account-select').selectOption('b'.repeat(32));
  await page.waitForURL('**/accounts/'+ 'b'.repeat(32) +'/**');await view('vault-panel');
  await page.waitForFunction(()=>document.querySelector('#vault-deposited-balance').textContent==='0 ETH');
  assert.equal(await page.locator('#vault-amount').inputValue(),'','account switch clears amount');
  assert.equal(await page.locator('#send-confirmation').isVisible(),false,'account switch cannot reuse quote');
  accountFixtures.pop();
  history=[];sent=[];quotes.clear();
  console.log('PASS: contract deposit/withdraw confirmation, keyboard action selection and Enter, three locales, 375 px light/dark, zero balance, hidden unavailable states and RPC recovery (mock APIs).');
  await page.goto(base+'/shared-demo');
  await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  assert.equal(await page.locator('a[href="/login"]').count(),0);
  await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
  await page.waitForFunction(()=>document.querySelector('#token-list').getAttribute('aria-busy')==='false');
  enforceTokenLimit=true;
  await page.locator('#refresh-wallet').click();await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
  await page.waitForFunction(()=>document.querySelector('#token-list').getAttribute('aria-busy')==='false');
  assert.equal(tokenLimitHits,0,'token refresh respects single concurrent POST limit');
  assert.equal(await page.locator('#token-error').isVisible(),false);enforceTokenLimit=false;
  for(const csrf of ['fixture-after-restart','fixture']){
   walletCSRF=csrf;await page.locator('#refresh-wallet').click();
   await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
   await page.waitForFunction(()=>document.querySelector('#token-list').getAttribute('aria-busy')==='false');
   assert.equal(await page.locator('#token-error').isVisible(),false,'refresh renews CSRF before token queries after a restart');
  }

  assert.equal(await page.locator('#manage-accounts').isEnabled(),true);
  await page.locator('#manage-accounts').click();
  await page.locator('#add-account').click();
  await page.locator('#account-name').fill('訪客測試錢包');
  await page.locator('#account-password').fill('visitor-test-password');
  await page.locator('#account-password-confirm').fill('does-not-match');
  await page.locator('#save-account').click();
  assert.match(await page.locator('#account-feedback').textContent(), /兩次密碼不同/);
  await page.locator('#close-account-manager').click();
  await view('send-panel');
  assert.equal(await page.locator('#send-form button[type=submit]').isEnabled(),true);
  for (const value of ['/solana','/tron']) assert.equal(await page.locator(`#network-select option[value="${value}"]`).isDisabled(),false);
  for (const [family,prefix] of [['solana','sol'],['tron','tron']]) {
    await page.goto(base+'/'+family+'/?shared');
    assert.equal(await page.locator('#'+prefix+'-create button').isDisabled(),true);
    assert.equal(await page.locator('#'+prefix+'-password-form button').isDisabled(),true);
    assert.equal(await page.locator('#'+prefix+'-transfer button[type=submit]').isEnabled(),true);
  }
  console.log('PASS: shared demo enables named wallet creation and all networks; existing wallet administration restricted.');
  await page.goto(base+'/shared-demo');
  await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  assert.equal(await page.locator('.nav-tools').count(),0);
  assert.equal(await page.locator('#nav-activity').count(),0);
  await view('activity-panel');
  assert.equal(await page.locator('#activity-sync-settings').isVisible(),false);
  await page.locator('#activity-refresh').click();
  await page.waitForFunction(()=>document.querySelector('#activity-updated').textContent.includes('最後更新'));
  await page.screenshot({path:path.join(screenshots,'wallet-ux-activity.png'),fullPage:true});
  await view('settings-panel');assert.equal(await page.locator('#password-form').isVisible(),false);
  await view('test-funding-panel');assert.equal(await page.locator('#claim-native').isVisible(),true);
  await page.locator('#claim-native').click();
  await page.waitForFunction(()=>document.querySelector('#test-funding-status').textContent.includes('鏈上執行成功'));
  assert.equal(await page.locator('#claim-usdc').isEnabled(),true);
  await view('send-panel');
  await page.locator('#recipient-search').fill('主要');
  assert.ok(await page.locator('#contact-select optgroup option').count()>0,'own wallets available as recipients');
  await page.locator('#send-to').fill(address);
  await page.locator('#save-recipient').click();
  await page.locator('#contact-label').fill('UX 測試聯絡人');
  await page.locator('#contact-form button').click();
  await page.locator('#contacts-search').fill('不符合');assert.equal(await page.locator('#contacts-list strong').count(),0);
  await page.locator('#contacts-search').fill('UX');
  await page.locator('#contacts-list button').filter({hasText:'編輯名稱'}).click();
  await page.locator('#contact-label').fill('UX 已改名');await page.locator('#contact-form button').click();
  assert.equal(await page.locator('#contacts-list strong').textContent(),'UX 已改名');
  await page.screenshot({path:path.join(screenshots,'wallet-ux-contacts.png'),fullPage:true});
  await page.locator('#contacts-list button').filter({hasText:'移除'}).click();assert.equal(await page.locator('#contacts-list strong').count(),0);
  await page.locator('#undo-contact').click();assert.equal(await page.locator('#contacts-list strong').textContent(),'UX 已改名');
  await page.locator('#contacts-list button').filter({hasText:'發送資產'}).click();
  assert.equal(await page.locator('#send-to').inputValue(),address);
  await page.reload();await view('contacts-panel');assert.equal(await page.locator('#contacts-list strong').textContent(),'UX 已改名');
  await page.goto(base+'/net/base/');await view('contacts-panel');assert.equal(await page.locator('#contacts-list strong').count(),0,'contacts isolated by network');
  for (const [family,recipientID,value] of [['solana','sol-to',solAddress],['tron','tron-to',tronAddress]]) {
    await page.goto(base+'/'+family+'/?shared');await view('contacts-panel');
    await page.locator('#contact-label').fill('Chain contact');await page.locator('#contact-address').fill(address);await page.locator('#contact-form button').click();
    assert.equal(await page.locator('#contacts-list strong').count(),0,'EVM address rejected on another family');
    await page.locator('#contact-address').fill(value);await page.locator('#contact-form button').click();
    assert.equal(await page.locator('#contacts-list strong').textContent(),'Chain contact');
    await page.locator('#contacts-list button').filter({hasText:'發送資產'}).click();
    assert.equal(await page.locator('#'+recipientID).inputValue(),value);
    await view('test-funding-panel');assert.equal(await page.locator(family==='solana'?'#sol-airdrop':'#tron-claim').isEnabled(),true);
  }
  console.log('PASS: activity navigation/refresh, hidden unavailable controls, faucets, own-wallet recipients, contact rename/search/undo/persistence and EVM/SOL/TRX isolation.');

  for(const [slug,id] of Object.entries(networks)){
   await page.goto(base+'/net/'+slug+'/');await page.locator('#wallet-dashboard').waitFor({state:'visible'});
   assert.equal(await page.locator('#network-select').inputValue(),'/net/'+slug);
   assert.equal(await page.locator('#exchange-panel').isVisible(),false);
   await page.waitForFunction(id=>document.querySelector('#chain-id').textContent===String(id),id);
   if(id===80002){await page.waitForFunction(()=>document.querySelector('#wallet-balance').textContent.includes('POL'));assert.ok((await page.locator('#send-asset').textContent()).includes('POL'));assert.ok((await page.locator('#receive-network').textContent()).includes('POL'));await view('watch-panel');await page.locator('#watch-address').fill(address);await page.locator('#watch-form button[type=submit]').click();await page.waitForFunction(()=>document.querySelector('#watch-result').textContent.includes('1 POL'));}

  }
  await page.goto(base);await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  await page.locator('#manage-accounts').click();
  await page.locator('#add-account').click();
  await page.locator('#account-name').fill('可取消');
  await page.locator('#cancel-account-edit').click();assert.equal(accountWrites,0);
  await page.locator('#add-account').click();await page.locator('#account-name').fill('收款測試');await page.locator('#save-account').click();
  await page.waitForURL('**/accounts/'+ 'a'.repeat(32)+'/');
  await page.locator('#wallet-setup').waitFor({state:'visible'});assert.equal(accountWrites,1);
  await page.goto(base);await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  await page.locator('#manage-accounts').click();
  const accountRow=()=>page.locator('.account-row').filter({hasText:'收款測試'});
  await accountRow().getByRole('button',{name:'更名',exact:true}).click();
  accountFailure=true;await page.locator('#account-name').fill('改名測試');await page.locator('#save-account').click();
  await page.waitForFunction(()=>document.querySelector('#account-feedback').textContent.includes('無法儲存'));
  assert.equal(await page.locator('#account-editor').isVisible(),true);accountFailure=false;
  await page.locator('#account-name').fill('<img src=x onerror=alert(1)>');await page.locator('#save-account').click();
  await page.locator('#account-list-view').waitFor({state:'visible'});assert.equal(await page.locator('#account-list img').count(),0);
  const renamedRow=()=>page.locator('.account-row').filter({hasText:'<img src=x onerror=alert(1)>'});
  await renamedRow().getByRole('button',{name:'封存',exact:true}).click();await page.locator('#cancel-account-edit').click();
  assert.equal(accountFixtures[1].archived,false);
  await renamedRow().getByRole('button',{name:'封存',exact:true}).click();await page.locator('#save-account').click();
  await page.waitForFunction(()=>document.querySelectorAll('#account-select option').length===1);
  await page.locator('#show-archived-accounts').check();
  await renamedRow().getByRole('button',{name:'還原',exact:true}).click();await page.locator('#save-account').click();
  await page.waitForFunction(()=>document.querySelectorAll('#account-select option').length===2);
  await page.setViewportSize({width:390,height:844});
  assert.ok(await page.evaluate(()=>document.querySelector('#account-manager').getBoundingClientRect().width<=innerWidth));
  await page.screenshot({path:path.join(screenshots,'multi-chain-wallet-wallet-manager-mobile.png')});
  await page.keyboard.press('Escape');assert.equal(await page.locator('#account-manager').isVisible(),false);
  await page.setViewportSize({width:1280,height:900});
  assert.equal(await page.locator('#token-form, #token-contract').count(),0);
  assert.equal(await page.locator('#token-list details').count(),0);
  assert.ok(await page.locator('#token-list .token-balance').count()>0);
  assert.ok(await page.locator('#token-list a[href*="/token/"]').count()>0);
  // Previously saved tokens remain available after removing the custom-token entry point.
  await page.evaluate(token=>localStorage.setItem('multi-chain-wallet:tokens:',JSON.stringify([token])),customToken);
  await page.locator('#refresh-wallet').click();
  await page.waitForFunction(token=>[...document.querySelector('#send-asset').options].some(option=>option.value===token),customToken);
  await view('send-panel');await page.locator('#send-asset').selectOption(customToken);assert.equal(await page.locator('#send-amount-label').textContent(),'最小單位數量（整數）');await page.locator('#send-to').fill(address);await page.locator('#send-amount').fill('123');await page.locator('#send-form button[type=submit]').click();await page.locator('#send-confirmation').waitFor({state:'visible'});assert.equal([...quotes.values()].at(-1).amountRaw,'123');assert.equal([...quotes.values()].at(-1).amount,'');await page.locator('#cancel-send').click();
  await view('test-funding-panel');await page.locator('#claim-native').click();
  await page.waitForFunction(()=>document.querySelector('#test-funding-status').textContent.includes('鏈上執行成功'));
  assert.ok((await page.locator('#test-funding-status a').getAttribute('href')).endsWith('f'.repeat(64)));
  await view('contacts-panel');await page.locator('#contact-label').fill('<img src=x onerror=alert(1)>');await page.locator('#contact-address').fill(address);await page.locator('#contact-form button').click();
  assert.equal(await page.locator('#contacts-list img').count(),0);
  await view('send-panel');await page.locator('#contact-select').selectOption(address);assert.equal(await page.locator('#send-to').inputValue(),address);
  await require('./history-freshness.cjs')(context, base);
  await require('./nonce-recovery.cjs')(context, base);
  await require('./workspace-regression.cjs')(page);
  await require('./escrow-polling.cjs')(page);
  await require('./send-recovery.cjs')(page);
  await require('./wallet-refresh.cjs')(page);
  await require('./escrow-speedup.cjs')(page);
  console.log('PASS: transaction and block queries reject stale successes/errors and clear invalidated results.');
  await view('watch-panel');await page.locator('#watch-address').fill(address);await page.locator('#watch-form button[type=submit]').click();await page.waitForFunction(()=>document.querySelector('#watch-result').textContent.includes('1 ETH'));
  balanceFailure=true;await page.locator('#watch-form button[type=submit]').click();await page.waitForFunction(()=>document.querySelector('#watch-result').textContent.includes('unavailable'));balanceFailure=false;
  await view('exchange-panel');await page.locator('#exchange-action').selectOption('eth-usdc');await page.locator('#exchange-amount').fill('0.000001');
  async function step(){await page.locator('#exchange-submit').click();await page.locator('#send-confirmation').waitFor({state:'visible'});await page.locator('#send-password').fill('fixture-password-only');await page.locator('#confirm-send-button').click();await page.locator('#send-confirmation').waitFor({state:'hidden'});await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled&&!document.querySelector('#exchange-workflow').textContent.includes('等待 0x'));}
  await step();await step();await step();assert.deepEqual(sent,['wrap','approve','swap']);
  assert.equal([...quotes.values()].find(q=>q.action==='swap').poolFee,500);
  await view('history-panel');await page.locator('#history-search').fill('absent-query');assert.equal(await page.locator('#wallet-history .history-row').count(),0);
  await page.locator('#history-search').fill('');assert.equal(await page.locator('#wallet-history .history-row').count(),3);
  await view('exchange-panel');await page.locator('#flow-stop').click();await view('exchange-panel');await page.locator('#exchange-action').selectOption('usdc-eth');await page.locator('#exchange-amount').fill('0.001');
  await step();await step();await step();assert.deepEqual(sent.slice(3),['approve','swap','unwrap']);
  assert.equal([...quotes.values()].at(-1).amount,'0.000001000000000000');
  await view('diagnostics-panel');await page.locator('#diagnostics-refresh').click();await page.waitForFunction(()=>document.querySelector('#diagnostics-result').textContent.includes('備援切換數'));
  await page.locator('#diagnostic-hash').fill('0x'+'1'.repeat(64));await page.locator('#diagnose-tx').click();await page.waitForFunction(()=>document.querySelector('#diagnostic-tx').textContent.includes('已 finalized'));
  await view('exchange-panel');await page.locator('#flow-stop').click();await view('exchange-panel');await page.locator('#exchange-action').selectOption('eth-usdc');await page.locator('#exchange-amount').fill('0.000001');loseSendResponse=true;
  await page.locator('#exchange-submit').click();await page.locator('#send-confirmation').waitFor({state:'visible'});await page.locator('#send-password').fill('fixture-password-only');await page.locator('#confirm-send-button').click();await page.locator('#confirm-error').waitFor({state:'visible'});
  await page.reload();await page.waitForFunction(()=>document.querySelector('#exchange-workflow').textContent.includes('查核授權並兌換'));
  assert.equal(sent.filter(action=>action==='wrap').length,2,'lost response repeated wrap');
  // Rejected sends may restart the step; ambiguous sends above retain the original quote.
  for (const status of [401, 400]) {
    await view('exchange-panel'); await page.locator('#flow-stop').click();
    await page.locator('#exchange-action').selectOption('eth-usdc');
    await page.locator('#exchange-amount').fill('0.000001');
    const sentBefore = sent.length;
    rejectSendStatus = status;
    await page.locator('#exchange-submit').click();
    await page.locator('#send-password').fill('incorrect-password');
    await page.locator('#confirm-send-button').click();
    await page.locator('#confirm-error').waitFor({state:'visible'});
    await page.locator('#cancel-send').click();
    assert.equal(sent.length, sentBefore, 'rejected request never broadcasts');
    await page.reload(); await view('exchange-panel');
    await page.waitForFunction(()=>!document.querySelector('#refresh-wallet').disabled);
    await page.locator('#exchange-submit').click();
    await page.locator('#send-confirmation').waitFor({state:'visible'});
    await page.locator('#cancel-send').click();
    rejectSendStatus = 0;
  }
  const originalHash = '0x'+'a'.repeat(64), replacementHash = '0x'+'b'.repeat(64);
  for (const action of ['speedup', 'cancel', 'speedup_cancel', 'unrelated']) {
    const original = {hash:originalHash,quoteId:'replacement-original',state:'replaced',replacedBy:replacementHash,action:'wrap',to:address,amount:'0.000001',symbol:'ETH'};
    const cancellation = action === 'cancel' || action === 'speedup_cancel';
    const replacement = {...original,hash:replacementHash,quoteId:'replacement',state:'succeeded',replacedBy:undefined,
      action:action==='speedup_cancel'?'speedup':action,amount:cancellation?'0':original.amount};
    history = [{...original,state:'pending',replacedBy:undefined}];
    transactionStatuses.set(originalHash,{hash:originalHash,state:'pending'});
    transactionStatuses.set(replacementHash,{hash:replacementHash,state:'succeeded'});
    await page.evaluate(({originalHash})=>sessionStorage.setItem('multi-chain-wallet:exchange-flow:',JSON.stringify({id:'replacement-flow',direction:'eth-usdc',amount:'0.000001',phase:'wrap',pending:{quoteID:'replacement-original',hash:originalHash,kind:'wrap'}})),{originalHash});
    await page.reload();
    await page.waitForFunction(()=>!document.querySelector('#wallet-dashboard').hidden&&!document.querySelector('#refresh-wallet').disabled);
    history = [original,replacement];
    await view('exchange-panel');
    await page.locator('#exchange-submit').click();
    await page.waitForFunction(()=>!document.querySelector('#exchange-submit').disabled);
    const current = await page.evaluate(()=>JSON.parse(sessionStorage.getItem('multi-chain-wallet:exchange-flow:')));
    assert.equal(current.phase,action==='speedup'?'swap':'wrap',action+' follows only the original intent');
    assert.equal(Boolean(current.pending),action==='unrelated',action+' preserves uncertainty');
  }
  transactionStatuses.clear(); history=[];
  await page.evaluate(()=>{sessionStorage.removeItem('multi-chain-wallet:exchange-flow:');localStorage.removeItem('multi-chain-wallet:tokens:');});
  scanTokens = Array.from({length:20},(_,i)=>'0x6'+String(i+1).padStart(39,'0'));
  await page.reload();
  await page.waitForFunction(()=>document.querySelectorAll('#send-asset option').length>=23);
  await view('activity-panel');
  // The developer controls are hidden in the normal UI, but their submit handler remains callable.
  await page.evaluate(()=>document.querySelector('#activity-from').value='');
  syncRequests=[]; syncFailureBatch=2;
  await page.evaluate(()=>document.querySelector('#activity-sync-form').requestSubmit());
  await page.waitForFunction(()=>!document.querySelector('#activity-sync-form button').disabled);
  assert.equal(syncRequests.length,2);
  assert.deepEqual(syncRequests.map(body=>body.from),[0,81]);
  assert.equal(await page.locator('#activity-from').inputValue(),'81','failed batch must retain the resolved start instead of moving the default window');
  syncRequests=[]; syncFailureBatch=0;
  await page.evaluate(()=>document.querySelector('#activity-sync-form').requestSubmit());
  await page.waitForFunction(()=>document.querySelector('#activity-from').value==='101');
  assert.deepEqual(syncRequests.map(body=>body.contracts.length),[20,2]);
  assert.deepEqual(syncRequests.map(body=>body.from),[81,81]);
  assert.equal(new Set(syncRequests.flatMap(body=>body.contracts)).size,22);
  scanTokens=[];
  console.log('PASS: explicit send rejection recovery, replacement intent/cancellation checks, and complete 22-token batched sync with failure cursor retention.');
  const discoveryCandidates = Array.from({length:21},(_,i)=>'0x7'+String(i+1).padStart(39,'0'));
  const discoveryFailures = new Set(discoveryCandidates.slice(0,20));
  const discoveryReads = [];
  let releaseScan;
  const scanGate = new Promise(resolve => { releaseScan = resolve; });
  await page.route('**/api/wallet/scan', async route => {
    await scanGate;
    return route.fulfill({json:{enabled:false,start:90,next:91,finalized:100,tokens:discoveryCandidates}});
  });
  await page.route('**/api/wallet/token', route => {
    const contract = route.request().postDataJSON().contract.toLowerCase();
    if (!discoveryCandidates.includes(contract)) return route.continue();
    discoveryReads.push(contract);
    return discoveryFailures.has(contract)
      ? route.fulfill({status:502,json:{error:'fixture nonstandard metadata'}})
      : route.fulfill({json:{contract,symbol:'DISCOVERED',decimals:18,balance:'1',balanceRaw:'1000000000000000000',allowanceRaw:'0'}});
  });
  await page.reload();
  await page.waitForFunction(()=>document.querySelector('#wallet-loading').hidden && document.querySelector('#token-list').getAttribute('aria-busy')==='false');
  async function watchDiscoveryRender() {
    await page.evaluate(()=>{
      window.discoveryRendered = false;
      const observer = new MutationObserver(()=>{window.discoveryRendered = true;observer.disconnect();});
      observer.observe(document.querySelector('#send-asset'),{childList:true});
    });
  }
  await watchDiscoveryRender();
  releaseScan();
  await page.waitForFunction(()=>window.discoveryRendered);
  assert.deepEqual(discoveryReads,discoveryCandidates.slice(0,20));
  await watchDiscoveryRender();
  await page.locator('#stop-scan').evaluate(button=>button.click());
  await page.waitForFunction(()=>window.discoveryRendered);
  assert.equal(discoveryReads[20],discoveryCandidates[20],'later candidates get a turn after a full failed batch');
  assert.equal(discoveryReads.length,40,'each discovery pass remains capped at 20 requests');
  assert.ok(await page.locator(`#send-asset option[value="${discoveryCandidates[20]}"]`).count());
  discoveryFailures.delete(discoveryCandidates[0]);
  await watchDiscoveryRender();
  await page.locator('#stop-scan').evaluate(button=>button.click());
  await page.waitForFunction(()=>window.discoveryRendered);
  assert.ok(await page.locator(`#send-asset option[value="${discoveryCandidates[0]}"]`).count(),'previously failed metadata is retried and can recover');
  await page.unroute('**/api/wallet/scan');
  await page.unroute('**/api/wallet/token');
  console.log('PASS: failed discovery candidates rotate, preserve the batch limit, and recover on retry.');
  exists=false;await page.reload();await page.locator('#wallet-setup').waitFor({state:'visible'});await view('watch-panel');assert.equal(await page.locator('#watch-panel').isVisible(),true);
  await page.setViewportSize({width:390,height:844});assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'mobile overflow');
  await page.goto(base+'/solana/');await page.locator('#sol-setup').waitFor({state:'visible'});
  await page.locator('#sol-mnemonic').fill('fixture mnemonic');await page.locator('#sol-password').fill('fixture-password-only');await page.locator('#sol-password-confirm').fill('fixture-password-only');await page.locator('#sol-create button').click();
  await page.locator('#sol-dashboard').waitFor({state:'visible'});await page.waitForFunction(()=>document.querySelector('#sol-balance').textContent==='0.1 SOL');
  await view('test-funding-panel');await page.locator('#sol-airdrop').click(); await page.waitForFunction(()=>document.querySelector('#sol-funding-status').textContent.includes('限流'));
  assert.equal(await page.locator('#sol-airdrop').isEnabled(),true);
  assert.equal(await page.locator('#sol-password').inputValue(),'');
  solCSRF='sol-after-restart';
  await view('send-panel');await page.locator('#sol-self').click();await page.locator('#sol-transfer button[type=submit]').click();
  await page.waitForFunction(()=>document.querySelector('#sol-error').textContent.includes('stale CSRF'));
  assert.equal(await page.locator('#sol-confirm').isVisible(),false);
  await view('overview');await page.locator('#sol-refresh').click();await page.waitForFunction(()=>!document.querySelector('#sol-refresh').disabled);
  assert.equal(solSends,0,'refresh never signs or sends a Solana transaction');
  await view('send-panel');await page.locator('#sol-transfer button[type=submit]').click();await page.locator('#sol-confirm').waitFor({state:'visible'});
  assert.ok((await page.locator('#sol-quote').textContent()).includes('Solana Devnet'));assert.equal(solSends,0);
  await page.locator('#sol-sign-password').fill('fixture-password-only');await page.locator('#sol-sign button[type=submit]').click();await page.locator('#sol-confirm').waitFor({state:'hidden'});
  await page.waitForFunction(()=>document.querySelector('#sol-history').textContent.includes('終局確認'));assert.equal(solSends,1);assert.equal(await page.locator('#sol-sign-password').inputValue(),'');
  balanceFailure=true;await view('overview');await page.locator('#sol-refresh').click();await page.waitForFunction(()=>document.querySelector('#sol-error').textContent.includes('Devnet unavailable'));assert.equal(await page.locator('#sol-balance').textContent(),'—');
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'Solana mobile overflow');
  balanceFailure=false;await page.goto(base+'/tron/');await page.locator('#tron-setup').waitFor({state:'visible'});
  await page.locator('#tron-mnemonic').fill('fixture mnemonic');await page.locator('#tron-password').fill('fixture-password-only');await page.locator('#tron-password-confirm').fill('fixture-password-only');await page.locator('#tron-create button').click();
  await page.locator('#tron-dashboard').waitFor({state:'visible'});await view('test-funding-panel');await page.locator('#tron-claim').click();await page.waitForFunction(()=>document.querySelector('#tron-funding-status').textContent.includes('庫存不足'));assert.equal(await page.locator('#tron-claim').isEnabled(),true);await page.waitForFunction(()=>document.querySelector('#tron-balance').textContent==='100 TRX');
  await view('send-panel');await page.locator('#tron-contract').fill(tronAddress);await page.locator('#tron-token-query').click();await page.waitForFunction(()=>document.querySelector('#tron-token-info').textContent.includes('TEST'));
  await page.locator('#tron-self').click();assert.equal(await page.locator('#tron-amount-label').textContent(),'數量');assert.equal(await page.locator('#tron-to').inputValue(),'');await page.locator('#tron-to').fill(tronAddress);
  tronCSRF='tron-after-restart';await page.locator('#tron-transfer button[type=submit]').click();
  await page.waitForFunction(()=>document.querySelector('#tron-error').textContent.includes('stale CSRF'));
  assert.equal(await page.locator('#tron-confirm').isVisible(),false);
  await view('overview');await page.locator('#tron-refresh').click();await page.waitForFunction(()=>!document.querySelector('#tron-refresh').disabled);
  assert.equal(tronSends,0,'refresh never signs or sends a TRON transaction');
  await view('send-panel');await page.locator('#tron-transfer button[type=submit]').click();await page.locator('#tron-confirm').waitFor({state:'visible'});assert.equal(tronSends,0);assert.ok((await page.locator('#tron-quote').textContent()).includes('TRON Shasta'));
  await page.locator('#tron-sign-password').fill('fixture-password-only');await page.locator('#tron-sign button[type=submit]').click();await page.locator('#tron-confirm').waitFor({state:'hidden'});await page.waitForFunction(()=>document.querySelector('#tron-history').textContent.includes('終局確認'));assert.equal(tronSends,1);assert.equal(await page.locator('#tron-sign-password').inputValue(),'');
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'TRON mobile overflow');
  console.log('PASS: Solana/TRON refresh renews rotated CSRF before quoting without automatically signing or sending (mock APIs).');
  await require('./recovery-focused.cjs')(page);
  exists=true; await page.setViewportSize({width:1280,height:900});await page.goto(base);await page.locator('#wallet-dashboard').waitFor({state:'visible'});
  await view('send-panel');await page.locator('#send-to').fill(address);await page.locator('#send-amount').fill('0.012345');
  await page.locator('#language-select').selectOption('en');
  await page.waitForFunction(()=>document.querySelector('#page-title').textContent==='Send');
  assert.equal(await page.locator('#send-to').inputValue(),address);assert.equal(await page.locator('#send-amount').inputValue(),'0.012345');
  assert.equal(await page.locator('#send-form button[type=submit]').textContent(),'Review transfer');
  await page.locator('#theme-toggle').click();const theme=await page.locator('html').getAttribute('data-theme');
  await page.reload();assert.equal(await page.locator('html').getAttribute('data-theme'),theme);assert.equal(await page.locator('#language-select').inputValue(),'en');
  await page.locator('#send-to').fill(address);await page.locator('#send-amount').fill('0.012345');await page.locator('#send-form button[type=submit]').click();await page.locator('#send-confirmation').waitFor({state:'visible'});
  await page.locator('#send-password').fill('fixture-secret-not-sent');
  await page.evaluate(()=>window.FlowI18n.setLocale('zh-CN'));
  assert.equal(await page.locator('#send-password').inputValue(),'fixture-secret-not-sent');assert.equal(await page.locator('#confirm-title').textContent(),'确认这笔交易');
  await page.locator('#cancel-send').click();
  await page.locator('#language-select').selectOption('zh-CN');await page.waitForFunction(()=>document.querySelector('#page-title').textContent==='发送资产');
  await view('overview');assert.equal(await page.locator('#send-panel').isVisible(),false);assert.equal(await page.locator('#diagnostics-panel').isVisible(),false);assert.equal(await page.locator('#first-transaction').isVisible(),false);
  await page.locator('#language-select').selectOption('en');
  await page.setViewportSize({width:390,height:844});await page.locator('#menu-toggle').click();assert.equal(await page.locator('#menu-toggle').getAttribute('aria-expanded'),'true');
  await page.waitForFunction(()=>document.querySelector('#app-sidebar').contains(document.activeElement));await page.keyboard.press('Escape');assert.equal(await page.locator('#menu-toggle').getAttribute('aria-expanded'),'false');
  await page.locator('#menu-toggle').click();await page.locator('#app-sidebar a[href="#receive-panel"]').click();await page.locator('#receive-panel').waitFor({state:'visible'});assert.equal(await page.locator('#receive-panel').isVisible(),true);assert.equal(await page.locator('#menu-toggle').getAttribute('aria-expanded'),'false');
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'English mobile overflow');
  await page.goto(base+'/solana/');assert.equal(await page.locator('#language-select').inputValue(),'en');await view('send-panel');assert.equal(await page.locator('#sol-transfer button[type=submit]').textContent(),'Review transfer');
  await page.goto(base+'/tron/');await view('test-funding-panel');assert.equal(await page.locator('#tron-claim').textContent(),'Get 5 test TRX');
  const previousTronHistory=tronHistory;
  tronTokenSymbol='TOKEN：發送資產';
  tronHistory=[{signature:'2'.repeat(64),to:tronAddress,amount:'1',symbol:tronTokenSymbol,state:'finalized',finalized:true}];
  await page.reload();await view('send-panel');
  await page.waitForFunction(()=>document.querySelector('#tron-history').textContent.includes('TOKEN'));
  await page.locator('#tron-contract').fill(tronAddress);await page.locator('#tron-token-query').click();
  await page.waitForFunction(()=>document.querySelector('#tron-token-info').textContent.includes('TOKEN'));
  await page.locator('#tron-to').fill(tronAddress);await page.locator('#tron-amount').fill('1');
  await page.locator('#tron-transfer button[type=submit]').click();await page.locator('#tron-confirm').waitFor({state:'visible'});
  for(const locale of ['en','zh-CN']){
    await page.locator('#language-select').selectOption(locale);
    for(const selector of ['#tron-history strong','#tron-token-info','#tron-quote'])
      assert.ok((await page.locator(selector).textContent()).includes(tronTokenSymbol),selector+' preserves the token symbol in '+locale);
  }
  await page.locator('#tron-cancel').click();
  tronTokenSymbol='TEST';tronHistory=previousTronHistory;
  await page.locator('#language-select').selectOption('zh-TW');assert.equal(await page.locator('#tron-claim').textContent(),'領取 5 測試 TRX');
  await page.setViewportSize({width:1280,height:900});await page.goto(base);await page.locator('#language-select').selectOption('en');
  const untranslated=[];
  for(const route of ['overview','send-panel','receive-panel','exchange-panel','test-funding-panel','activity-panel','history-panel','settings-panel','watch-panel','contacts-panel','diagnostics-panel','balance-panel','transaction-panel','first-transaction']){
    await view(route);
    await page.evaluate(()=>document.querySelectorAll('main details').forEach(el=>el.open=true));
    const remaining=await page.evaluate(()=>{
      const walker=document.createTreeWalker(document.querySelector('main'),NodeFilter.SHOW_TEXT);const result=[];
      while(walker.nextNode()) {const n=walker.currentNode,p=n.parentElement;if(!p.closest('[translate="no"],script,style,textarea,input,code,pre') && p.getClientRects().length && /[\u3400-\u9fff]/.test(n.textContent))result.push(n.textContent.trim());}
      return result;
    });
    untranslated.push(...remaining.map(text=>route+': '+text));
  }
  for(const family of ['solana','tron']) {
    await page.goto(base+'/'+family+'/');
    for(const route of ['overview','send-panel','receive-panel','settings-panel','test-funding-panel','history-panel']) {
      await view(route);await page.evaluate(()=>document.querySelectorAll('main details').forEach(el=>el.open=true));
      const remaining=await page.evaluate(()=>{const walker=document.createTreeWalker(document.querySelector('main'),NodeFilter.SHOW_TEXT),texts=[];while(walker.nextNode()){const n=walker.currentNode,p=n.parentElement;if(!p.closest('[translate="no"],script,style,textarea,input,code,pre')&&p.getClientRects().length&&/[\u3400-\u9fff]/.test(n.textContent))texts.push(n.textContent.trim());}return texts;});
      untranslated.push(...remaining.map(text=>family+'/'+route+': '+text));
    }
  }
  await page.goto(base+'/showcase');await page.waitForFunction(()=>document.querySelector('#verified-evidence').textContent.includes('Pending acceptance'));
  const evidenceText=await page.locator('#verified-evidence').textContent();assert.ok(!/[\u3400-\u9fff]/.test(evidenceText),'English evidence and limitations');
  assert.deepEqual(untranslated,[],'untranslated English UI messages');
  assert.equal(await page.evaluate(()=>window.FlowI18n.t('__proto__')),'__proto__');
  await page.goto(base);await view('contacts-panel');await page.locator('#contact-label').fill('總覽');await page.locator('#contact-address').fill(address);await page.locator('#contact-form button').click();assert.equal(await page.locator('#contacts-list strong').textContent(),'總覽');
  await page.locator('#language-select').selectOption('zh-CN');assert.equal(await page.locator('#contacts-list strong').textContent(),'總覽');
  // Reflow and theme checks use the same API fixtures as the interaction tests.
  for (const family of ['', 'solana/', 'tron/']) {
    await page.goto(base+'/'+family);
    for (const locale of ['zh-TW','en','zh-CN']) {
      await page.locator('#language-select').selectOption(locale);
      for (const width of [375,768,1024,1440]) {
        await page.setViewportSize({width,height:900});
        for (const mode of ['light','dark']) {
          if(await page.getAttribute('html','data-theme')!==mode)await page.locator('#theme-toggle').click();
          for (const target of ['overview','send-panel','receive-panel','test-funding-panel']) {
            await view(target);
            assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth), `horizontal overflow: ${family} ${locale} ${width} ${mode} ${target}`);
          }
          await view('overview');
          if (!family && width===1440) {
            const assets=await page.locator('#tokens-panel').boundingBox(),activity=await page.locator('#history-panel').boundingBox();
            assert.ok(Math.abs(assets.y-activity.y)<2 && assets.x<activity.x,'desktop overview groups assets beside history');
          }
          if (process.env.MULTI_CHAIN_WALLET_UI_SCREENSHOTS && locale==='en' && [375,1440].includes(width)) {
            await page.screenshot({path:path.join(screenshots,`${family.replace('/','')||'evm'}-${width}-${mode}.png`),fullPage:true,animations:'disabled'});
          }
        }
      }
    }
  }
  await page.emulateMedia({reducedMotion:'reduce'});
  assert.equal(await page.locator('#theme-toggle').evaluate(el=>getComputedStyle(el).transitionDuration),'0s');
  await page.goto(base);await view('exchange-panel');
  assert.equal(await page.locator('#swap-advanced').getAttribute('open'),null);
  await page.locator('#swap-advanced summary').focus();await page.keyboard.press('Enter');
  assert.equal(await page.locator('#swap-advanced').evaluate(el=>el.open),true,'keyboard opens advanced swap settings');
  console.log('PASS: all wallet families reflow at 375/768/1024/1440 px in three locales and both themes; keyboard disclosure and reduced motion.');
  console.log('PASS: task-based navigation, en/zh-CN/zh-TW live switching, draft/password preservation, theme persistence, mobile menu/escape, and cross-chain preferences.');
  assert.deepEqual(errors,[]);console.log('PASS: five EVM networks including POL, contacts, observation, RPC failures, pool selection, both exchange workflows, lost-response reload recovery, history search, diagnostics, Solana and TRON create/quote/send/finalized UI, TRC20 query, mobile layout (mock APIs only).');
 }finally{await browser.close();server.close();}
})().catch(error=>{console.error(error);server.close();process.exitCode=1;});
