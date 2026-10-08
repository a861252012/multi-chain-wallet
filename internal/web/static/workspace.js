'use strict';
(() => {
  const watchesKey = 'multi-chain-wallet:watches:' + networkID;
  function stored(name) {
    try { const value = JSON.parse(localStorage.getItem(name) || '[]'); return Array.isArray(value) ? value.filter(item => item && /^0x[0-9a-fA-F]{40}$/.test(item.address) && typeof item.label === 'string').slice(0,100) : []; }
    catch { return []; }
  }
  let watches = stored(watchesKey);
  let watchQuery = 0;
  let assetQuery = 0;
  function invalidateAssets() {
    assetQuery++;
    $('watch-result').replaceChildren();
  }
  for (const id of ['watch-address', 'watch-contract']) $(id).addEventListener('input', invalidateAssets);
  function invalidateWatch() {
    watchQuery++;
    $('watch-transactions').replaceChildren();
  }
  for (const id of ['watch-address', 'watch-hash', 'watch-block']) $(id).addEventListener('input', invalidateWatch);
  function address(input) {
    const value = $(input).value.trim();
    if (!/^0x[0-9a-fA-F]{40}$/.test(value) || /^0x0{40}$/i.test(value)) throw new Error('請輸入完整且非零的地址；轉帳時仍會驗證大小寫校驗。');
    return value;
  }
  function renderWatches() {
    $('watch-saved').replaceChildren();
    for (const item of watches) {
      const row = node('div','','contact-row'), select = node('button',item.address,'secondary'), remove = node('button','移除','secondary');
      select.type = remove.type = 'button';
      select.addEventListener('click',()=>{ invalidateWatch(); invalidateAssets(); $('watch-address').value=item.address; $('watch-form').requestSubmit(); });
      remove.addEventListener('click',()=>{
        try { const next=watches.filter(w=>w.address!==item.address); localStorage.setItem(watchesKey,JSON.stringify(next));watches=next;renderWatches(); }
        catch { $('watch-result').textContent='無法儲存變更。'; }
      });
      row.append(select,remove);$('watch-saved').append(row);
    }
  }
  renderWatches();
  $('watch-save').addEventListener('click',()=>{
    try { const value=address('watch-address');const next=watches.filter(w=>w.address.toLowerCase()!==value.toLowerCase()); if(next.length>=20)throw new Error('最多 20 個觀察地址。');next.push({address:value,label:''});localStorage.setItem(watchesKey,JSON.stringify(next));watches=next;renderWatches(); }
    catch(error){$('watch-result').textContent=errorMessage(error);}
  });
  $('watch-form').addEventListener('submit', async event => {
    event.preventDefault(); const button=event.currentTarget.querySelector('button');if(button.disabled)return;button.disabled=true;
    const queryID = ++assetQuery;
    $('watch-result').textContent='正在查詢…';
    try {
      const owner=address('watch-address'),contract=$('watch-contract').value.trim();
      const balance=await request('/api/balance?address='+encodeURIComponent(owner));
      if (queryID !== assetQuery) return;
      const results=[node('strong',`${balance.eth} ${nativeSymbol}`),node('p',`${networkName} · ${balance.address}`,'mono'),node('p',`區塊 ${balance.block} · ${time(balance.checkedAt)}`)];
      if(contract){
        try { const token=await request('/api/watch/token?'+new URLSearchParams({address:owner,contract}));results.push(node('p',`${token.balance} ${token.symbol}`),node('p',token.contract,'mono')); }
        catch(error){results.push(node('p','代幣查詢失敗：'+errorMessage(error),'error'));}
      }
      if (queryID === assetQuery) $('watch-result').replaceChildren(...results);
    }catch(error){if(queryID===assetQuery)$('watch-result').textContent=errorMessage(error);}finally{button.disabled=false;}
  });
  async function inspectWatch(hash) {
    const queryID = ++watchQuery;
    $('watch-hash').value = hash;
    const result=$('watch-transactions');result.textContent='正在核對收據…';
    try {
      const tx=await request('/api/watch/activity?'+new URLSearchParams({address:address('watch-address'),hash}));
      if (queryID !== watchQuery) return;
      result.replaceChildren(node('p',`${tx.state} · ${tx.blockTime ? time(tx.blockTime) : '尚無區塊時間'}`),explorer('tx',tx.hash));
      for(const movement of tx.movements)result.append(details([['方向',movement.kind],['資產',movement.asset],['原始整數數量',movement.raw],['證據',movement.evidence]]));
      if(!tx.movements.length)result.append(node('p','尚無可列入的資產移動。'));
    }catch(error){if(queryID===watchQuery)result.textContent=errorMessage(error);}
  }
  $('watch-tx-form').addEventListener('submit',event=>{event.preventDefault();inspectWatch($('watch-hash').value.trim());});
  $('watch-history-form').addEventListener('submit',async event=>{
    event.preventDefault();const button=event.currentTarget.querySelector('button');if(button.disabled)return;button.disabled=true;
    const queryID = ++watchQuery;
    $('watch-transactions').textContent='正在查核指定區塊…';
    try{
      const query=new URLSearchParams({address:address('watch-address')});const block=$('watch-block').value.trim();if(block)query.set('block',block);
      const response=await fetch(networkPrefix+'/api/watch/activity?'+query,{signal:AbortSignal.timeout(45000)});const data=await response.json();if(!response.ok)throw new Error(data.error);
      if (queryID !== watchQuery) return;
      $('watch-transactions').replaceChildren(node('p',`區塊 ${data.block} · ${data.coverage}`));
      for(const hash of data.hashes){const button=node('button',hash,'secondary mono');button.type='button';button.addEventListener('click',()=>inspectWatch(hash));$('watch-transactions').append(button);}
      if(!data.hashes.length)$('watch-transactions').append(node('p','此區塊沒有找到相關交易；這不代表此地址沒有其他歷史。'));
    }catch(error){if(queryID===watchQuery)$('watch-transactions').textContent=errorMessage(error);}finally{button.disabled=false;}
  });
  $('diagnostics-refresh').addEventListener('click',async()=>{
    const button=$('diagnostics-refresh');if(button.disabled)return;button.disabled=true;
    try{
      const result=await request('/api/diagnostics'),rpc=result.rpc||result||{};
      $('diagnostics-result').replaceChildren(details([['網路',networkName],['鏈 ID',result.chainId],['最新區塊',result.network?.block||'查詢失敗'],['使用端點',`${rpc.activeEndpoint||'—'} / ${rpc.endpointCount||'—'}`],['最近請求毫秒',rpc.lastRequestMs??'—'],['請求數',rpc.requests??'—'],['傳輸失敗數',rpc.transportFailures??'—'],['備援切換數',rpc.failovers??'—']]));
      if(result.error)$('diagnostics-result').append(node('p',result.error,'error'));
    }catch(error){$('diagnostics-result').textContent=errorMessage(error);}finally{button.disabled=false;}
  });
  $('diagnose-tx').addEventListener('click',async()=>{
    const button=$('diagnose-tx');if(button.disabled)return;button.disabled=true;
    try{
      const tx=await request('/api/transactions/'+encodeURIComponent($('diagnostic-hash').value.trim()));
      const steps=node('ol','','lifecycle');
      for(const label of ['已查到交易', tx.block?'已收錄區塊':'等待可驗證收據',tx.state==='succeeded'?'執行成功':tx.state==='reverted'?'執行失敗':'執行結果待確認',tx.finalized?'已 finalized':'等待 finalized'])steps.append(node('li',label));
      $('diagnostic-tx').replaceChildren(steps,details([['區塊雜湊',tx.blockHash||'尚無'],['確認數',tx.confirmations||'0'],['實際費用',tx.feeEth?tx.feeEth+' '+nativeSymbol:'尚無'],['查核時間',time(tx.checkedAt)]]),explorer('tx',tx.hash));
      if(tx.state==='reorg_detected')$('diagnostic-tx').prepend(node('p','偵測到鏈重組，原收據不能作為目前執行結果。','error'));
    }catch(error){$('diagnostic-tx').textContent=errorMessage(error);}finally{button.disabled=false;}
  });
})();
