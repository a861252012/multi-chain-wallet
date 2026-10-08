'use strict';
(() => {
  $('confirm-send-button').textContent = `簽署並送出至 ${networkName}`;
  const sharedDemo = !!document.querySelector('script[src="/static/shared.js"]');
  let walletState;
  let walletRefreshQueued = false;
  let accounts = [];
  let accountEdit = null;
  let setupMode = 'create';
  let quote;
  let sending = false;
  let vaultEnabled = false;
  let vaultBusy = false;
  const flowKey = 'multi-chain-wallet:exchange-flow:' + networkPrefix;
  let flow;
  try { const saved = JSON.parse(sessionStorage.getItem(flowKey) || 'null'); if (saved && ['eth-usdc','usdc-eth'].includes(saved.direction) && ['wrap','swap','unwrap','done'].includes(saved.phase) && typeof saved.amount === 'string' && typeof saved.id === 'string') flow = saved; } catch {}
  const tokens = new Map();
  let tokensRefreshing = false;
  let tokenRefreshQueued = false;
  const tokenStorageKey = 'multi-chain-wallet:tokens:' + networkPrefix;
  let historySnapshot = [];
  let historyRefreshError = '';
  let canCreateTransaction = false;
  let activityPage = 1;
  let activityLoading = false;
  const actionLabels = {speedup:"加速原交易",cancel:"取消原交易（零額自轉）",eth:"資產轉帳",transfer:"代幣轉帳",approve:"代幣授權",wrap:"ETH → WETH 包裝",unwrap:"WETH → ETH 解包",swap:"代幣兌換",vault_deposit:'存入合約',vault_withdraw:'取回錢包',escrow_fund:'付款至合約',escrow_release:'放款給收款人',escrow_refund:'退款給付款人'};
  const stateLabels = { replaced:"同 Nonce 的另一筆交易已收錄", submitted: '已廣播，等待收錄', pending: '等待區塊收錄', broadcast_unknown: '廣播結果待確認', succeeded: '鏈上執行成功', reverted: '鏈上執行失敗', reorg_detected: '區塊變更，待確認', receipt_unavailable: '收據尚不可用' };

  const nonceConsumedNotice = '原交易結果無法確認；該 nonce 已在 finalized 狀態被消耗，目前可建立新交易。請先核對原付款，避免重複支付。';
  const nonceConsumedBlockedNotice = '原交易結果無法確認；該 nonce 已在 finalized 狀態被消耗。請先核對原付款，避免重複支付。';
  function transactionLabel(tx) { return tx.nonceConsumed ? '原交易結果無法確認' : stateLabels[tx.state] || '狀態待確認'; }
  function transactionActionLabel(tx) {
    const label = actionLabels[tx.action] || '資產操作';
    return tx.action === 'speedup' && actionLabels[tx.escrowAction] ? `${label} · ${actionLabels[tx.escrowAction]}` : label;
  }
  function nonceNotice() { return canCreateTransaction ? nonceConsumedNotice : nonceConsumedBlockedNotice; }

  async function walletRequest(path, body) {
    const options = { signal: AbortSignal.timeout(60000) };
    if (body !== undefined) {
      options.method = 'POST';
      options.headers = { 'Content-Type': 'application/json', 'X-Wallet-CSRF': walletState.csrfToken };
      options.body = JSON.stringify(body);
    }
    const response = await fetch(networkPrefix + path, options);
    let data;
    try { data = await response.json(); }
    catch {
      const error = new Error('無法讀取伺服器回應，請稍後再試。');
      error.status = response.status;
      throw error;
    }
    if (!data || typeof data !== 'object') throw new Error('無法讀取伺服器回應，請稍後再試。');
    if (!response.ok) {
      const error = new Error(data.error || '操作失敗，請稍後重試。');
      error.status = response.status;
      error.code = data.code;
      throw error;
    }
    return data;
  }

  function showWalletError(id, error) {
    $(id).textContent = error.name === 'TimeoutError' ? '操作逾時，結果未知。請先更新錢包與交易紀錄，再決定是否重試。' : errorMessage(error);
    $(id).hidden = false;
  }

  async function refreshWallet() {
    if (!walletState?.exists) return;
    const button = $('refresh-wallet');
    if (button.disabled) { walletRefreshQueued = true; return; }
    button.disabled = true;
    $('check-funding').disabled = true;
    $('funding-status').textContent = `正在查詢 ${networkName} 餘額…`;
    $('wallet-error').hidden = true;
    $('wallet-balance-time').textContent = '正在查詢鏈上餘額…';
    try {
      const results = await Promise.allSettled([
        request(`/api/balance?address=${encodeURIComponent(walletState.address)}`),
        walletRequest('/api/wallet/history'),
        walletRequest('/api/wallet'),
      ]);
      if (results[2].status === 'fulfilled') walletState.csrfToken = results[2].value.csrfToken;
      if (results[0].status === 'fulfilled') {
        const balance = results[0].value;
        $('wallet-balance').replaceChildren(document.createTextNode(balance.eth + ' '), node('small', nativeSymbol));
        $('wallet-balance-time').textContent = `更新於 ${time(balance.checkedAt)}`;
        $('funding-status').textContent = BigInt(balance.wei) > 0n
          ? `已查到 ${balance.eth} ${networkName} ${nativeSymbol}。可先預估費用，是否足夠仍以報價為準。`
          : `目前餘額為 0 ${nativeSymbol}。若剛領取，請稍後再查；代幣無法直接支付 gas。`;
      } else {
        $('wallet-balance').replaceChildren(document.createTextNode('— '), node('small', nativeSymbol));
        $('wallet-balance-time').textContent = '無法取得最新餘額';
        $('funding-status').textContent = '無法查到最新餘額，目前無法確認測試幣是否到帳，請稍後重試。';
        showWalletError('wallet-error', results[0].reason);
      }
      if (results[1].status === 'fulfilled') {
        renderHistory(results[1].value.transactions, results[1].value.refreshError || '', Boolean(results[1].value.canCreateTransaction));
      } else {
        renderHistory(historySnapshot, '無法更新紀錄，請稍後再試。', false);
        showWalletError('wallet-error', results[1].reason);
      }
      // Renew CSRF first; token reads have their own busy/error state and must not delay payment status.
      void refreshTokens().catch(error => showWalletError('token-error', error));
      if (flow?.pending) await reconcileFlow(results[1].status === 'fulfilled');
      if (document.body.dataset.view === 'vault-panel') {
        if ($('escrow-content').hidden) await refreshVault();
        else await refreshEscrow();
      }
    } catch (error) {
      showWalletError('wallet-error', error);
    } finally {
      button.disabled = false;
      $('check-funding').disabled = false;
      if (walletRefreshQueued) {
        walletRefreshQueued = false;
        void refreshWallet();
      }
    }
  }

  function renderHistory(transactions = historySnapshot, refreshError = historyRefreshError, allowNew = canCreateTransaction) {
    canCreateTransaction = Boolean(allowNew);
    historySnapshot = transactions || [];
    historyRefreshError = refreshError;
    const sent = historySnapshot.find(tx => tx.hash === $('send-feedback').dataset.hash);
    if (sent) showSent(sent);
    $('vault-history').replaceChildren();
    for (const tx of historySnapshot.filter(tx => (tx.action || '').startsWith('vault_')).slice(0, 5)) {
      const row = node('p');
      row.append(node('strong', actionLabels[tx.action] || '合約操作'), node('span', ` · ${tx.amount} ETH · `), node('span', transactionLabel(tx)), document.createTextNode(' '), explorer('tx', tx.hash));
      $('vault-history').append(row);
    }
    if (!$('vault-history').children.length) $('vault-history').append(node('p', '尚無合約操作紀錄，完成存入或取回後會顯示在這裡。', 'muted'));
    const term = $('history-search').value.trim().toLowerCase();
    transactions = historySnapshot.filter(tx => [tx.hash,tx.to,tx.orderId,tx.symbol,tx.amount,transactionActionLabel(tx),transactionLabel(tx),window.walletAddressLabel?.(tx.to)].some(value=>(String(value || '').toLowerCase().includes(term) || (window.FlowI18n?.t(String(value || '')) || String(value || '')).toLowerCase().includes(term))));
    const list = $('wallet-history');
    list.replaceChildren();
    if (historyRefreshError) list.append(node('p', historyRefreshError, 'error'));
    if (!transactions?.length) { list.append(node('p', '尚無交易。第一筆轉帳會出現在這裡。', 'muted')); return; }
    for (const tx of transactions) {
      const row = node('article', '', 'history-row');
      const summary = node('div', '', 'history-summary');
      const expanded = node('details', '', 'history-details');
      expanded.append(node('summary', '地址、收據與交易詳情'));
      const label = window.walletAddressLabel?.(tx.to);
      if (label) { const named = node('p', label); named.translate = false; summary.append(named); }
      const inspect = node('button','交易詳情','secondary');inspect.type='button';inspect.addEventListener('click',()=>{$('diagnostic-hash').value=tx.hash;location.hash='diagnostics-panel';$('diagnose-tx').click();});expanded.append(inspect);
      summary.append(node('strong', transactionActionLabel(tx)), node('p', time(tx.createdAt)));
      if (tx.orderId && tx.escrowBuyer) summary.append(escrowHistoryButton(tx));
      expanded.append(node('p', `${tx.action === 'approve' ? '被授權地址' : '收款人'} ${tx.to}`, 'mono'), explorer('tx', tx.hash));
      const amount = node('div', '', 'history-amount');
      amount.append(node('strong', `${tx.amount} ${tx.symbol}`), document.createElement('br'), node('span', transactionLabel(tx), `state-badge${tx.state === 'succeeded' ? '' : tx.state === 'reverted' ? ' danger' : ' warning'}`));
      if (tx.nonceConsumed) summary.append(node('p', nonceNotice(), 'warning'));
      if (tx.replacedBy) expanded.append(node('p', '已收錄的替代交易：'), explorer('tx', tx.replacedBy));
      if (tx.finalized) amount.append(node('p', '已達鏈上終局性'));
      else if (tx.confirmations) amount.append(node('p', `${tx.confirmations} 次確認`));
      if (tx.feeEth) amount.append(node('p', `實際費用 ${tx.feeEth} ${nativeSymbol}`));
      if (tx.action !== 'eth' && tx.state === 'succeeded') expanded.append(node('p', (tx.action || '').startsWith('vault_') ? '收據顯示執行成功；請至智慧合約頁更新餘額。' : '收據顯示執行成功；代幣實際移動請核對合約紀錄。'));
      if (!tx.nonceConsumed && (tx.state === 'broadcast_unknown' || tx.state === 'submitted' || tx.state === 'pending')) {
        const retry = node('button', '重新廣播原交易', 'secondary');
        retry.type = 'button';
        retry.addEventListener('click', async () => {
          retry.disabled = true;
          try {
            const data = await walletRequest('/api/wallet/retry', { hash: tx.hash });
            showSent(data);
            await refreshWallet();
          } catch (error) { showWalletError('wallet-error', error); }
          finally { retry.disabled = false; }
        });
        summary.append(retry);
        for (const [action, label] of [['speedup', '加速交易'], ['cancel', '取消交易']]) {
          const button = node('button', label, 'secondary');
          button.type = 'button';
          button.addEventListener('click', async () => {
            button.disabled = true;
            try { openConfirmation(await walletRequest('/api/wallet/quote', {action, hash:tx.hash})); }
            catch (error) { showWalletError('wallet-error', error); }
            finally { button.disabled = false; }
          });
          summary.append(button);
        }
      }
      row.append(summary, amount, expanded);
      list.append(row);
    }
  }

  function showSent(data) {
    $('send-feedback').dataset.hash = data.hash;
    $('send-feedback').hidden = false;
    $('send-feedback').replaceChildren(node('strong', transactionLabel(data)), node('p', data.hash, 'mono'), explorer('tx', data.hash), node('p', data.nonceConsumed ? nonceNotice() : '已記錄交易雜湊。更新交易紀錄以確認收錄結果；廣播成功不等於交易執行成功。'));
  }

  async function loadWallet() {
    $('wallet-loading').hidden = false;
    try {
      walletState = await walletRequest('/api/wallet');
      accounts = await walletRequest('/api/wallet/accounts');
      if (accounts.find(item => item.id === (accountPrefix.split('/')[2] || ''))?.archived) {
        location.replace(accountURL(accounts.find(item => !item.archived).id));
        return;
      }
      renderAccountSelect();
      $('account-select').value = accountPrefix.split('/')[2] || '';
      $('wallet-setup').hidden = walletState.exists;
      $('keystore-restore').hidden = walletState.exists;
      $('wallet-dashboard').hidden = !walletState.exists;
      $('nav-send').hidden = !walletState.exists;
      $('nav-history').hidden = !walletState.exists;
      $('nav-exchange').hidden = !walletState.exists;
      $('nav-vault').hidden = !walletState.exists || walletState.chainId !== 11155111;
      $('vault-panel').hidden = !walletState.exists || walletState.chainId !== 11155111;
      if (walletState.exists) {
        $('wallet-address').textContent = walletState.address;
        $('claim-native').textContent = `領取 ${walletState.chainId === 11155111 ? '0.001' : walletState.chainId === 80002 ? '0.1' : '0.0001'} 測試 ${nativeSymbol}`;
        $('claim-usdc').hidden = walletState.chainId !== 11155111;
        $('activity-csv').href = networkPrefix + '/api/wallet/activity?format=csv&page=1';
        $('wallet-explorer').href = `${explorerURL}/address/${encodeURIComponent(walletState.address)}`;
        $('exchange-panel').hidden = walletState.chainId !== 11155111;
        $('nav-exchange').hidden = walletState.chainId !== 11155111;
        $('quick-exchange').hidden = walletState.chainId !== 11155111;
        $('prepare-first-wrap').hidden = walletState.chainId !== 11155111;
        $('claim-test-eth').hidden = ![11155111,80002].includes(walletState.chainId);
        if (walletState.chainId === 80002) { $('claim-test-eth').href='https://faucet.polygon.technology/'; $('claim-test-eth').textContent='領取測試 POL ↗'; }
        $('faucet-status').hidden = walletState.chainId !== 11155111;
        $('first-transaction').hidden = walletState.chainId !== 11155111;
        $('receive-network').textContent = `僅接收 ${networkName} 的 ${nativeSymbol} 與代幣。`;
        if (walletState.chainId === 11155111) { if(flow) { $('exchange-action').value=flow.direction; $('exchange-amount').value=flow.amount; } updateExchangeAction(); renderFlow(); }
        await refreshWallet();
        await refreshActivity();
      }
    } catch (error) { showWalletError('wallet-error', error); }
    finally { $('wallet-loading').hidden = true; }
  }

  for (const mode of ['create', 'import']) {
    $(`choose-${mode}`).addEventListener('click', () => {
      setupMode = mode;
      for (const choice of ['create', 'import']) {
        $(`choose-${choice}`).classList.toggle('selected', choice === mode);
        $(`choose-${choice}`).setAttribute('aria-pressed', String(choice === mode));
      }
      $('mnemonic-input-group').hidden = mode !== 'import';
      $('import-mnemonic').required = mode === 'import';
      $('import-mnemonic').value = '';
      $('setup-submit').textContent = mode === 'import' ? '還原測試錢包' : '建立錢包';
      $('setup-error').hidden = true;
    });
  }

  $('wallet-setup-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    $('setup-error').hidden = true;
    const passwordLength = Array.from($('setup-password').value).length;
    if (passwordLength < 12 || passwordLength > 128) {
      showWalletError('setup-error', new Error('密碼長度必須介於 12 至 128 字元'));
      $('setup-password').focus();
      return;
    }
    if ($('setup-password').value !== $('confirm-password').value) {
      showWalletError('setup-error', new Error('兩次密碼不同，請重新確認。'));
      $('confirm-password').focus();
      return;
    }
    const mode = setupMode;
    const button = $('setup-submit');
    if (button.disabled) return;
    button.disabled = true;
    $('choose-create').disabled = true;
    $('choose-import').disabled = true;
    button.textContent = '正在加密金鑰，請稍候…';
    try {
      const body = { password: $('setup-password').value };
      if (mode === 'import') body.mnemonic = $('import-mnemonic').value.trim();
      const result = await walletRequest(`/api/wallet/${mode}`, body);
      form.reset();
      $('wallet-setup').hidden = true;
      if (result.mnemonic) {
        $('mnemonic-words').replaceChildren(...result.mnemonic.split(' ').map(word => node('li', word)));
        $('mnemonic-backup').hidden = false;
        $('backup-confirmed').focus();
      } else await loadWallet();
    } catch (error) {
      $('setup-password').value = '';
      $('confirm-password').value = '';
      $('import-mnemonic').value = '';
      showWalletError('setup-error', error);
    } finally {
      button.disabled = false;
      $('choose-create').disabled = false;
      $('choose-import').disabled = false;
      button.textContent = setupMode === 'import' ? '還原測試錢包' : '建立錢包';
    }
  });
  $('backup-confirmed').addEventListener('change', () => { $('finish-backup').disabled = !$('backup-confirmed').checked; });
  $('finish-backup').addEventListener('click', async () => {
    if (!$('backup-confirmed').checked) return;
    $('mnemonic-words').replaceChildren();
    $('mnemonic-backup').hidden = true;
    await loadWallet();
  });
  window.addEventListener('beforeunload', event => {
    if (!$('mnemonic-backup').hidden) { event.preventDefault(); event.returnValue = ''; }
  });

  $('copy-wallet-address').addEventListener('click', async () => {
    try { await navigator.clipboard.writeText(walletState.address); $('copy-wallet-address').textContent = '已複製'; }
    catch { showWalletError('wallet-error', new Error('無法複製，請選取上方完整地址手動複製。')); }
    setTimeout(() => { $('copy-wallet-address').textContent = '複製地址'; }, 2500);
  });
  $('claim-test-eth').addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(walletState.address);
      $('faucet-status').textContent = '地址已複製。請在 Google 水龍頭貼上地址並申請；完成後按「我已領取，查詢餘額」。';
    } catch {
      $('faucet-status').textContent = '無法自動複製，請手動複製上方收款地址，在 Google 水龍頭貼上並申請。';
    }
  });
  for (const [id, asset] of [['claim-native','native'],['claim-usdc','usdc']]) {
    $(id).addEventListener('click', () => window.claimTestTokens({
      buttons: [$('claim-native'), $('claim-usdc')], status: $('test-funding-status'),
      body: {chainId: walletState.chainId, asset, address: walletState.address},
      explorer: explorerURL, prefix: networkPrefix, refresh: refreshWallet,
    }));
  }
  $('history-search').addEventListener('input',()=>renderHistory());
  window.addEventListener('contacts-updated',()=>renderHistory());
  $('refresh-wallet').addEventListener('click', refreshWallet);
  $('check-funding').addEventListener('click', refreshWallet);
  $('prepare-self-transfer').addEventListener('click', () => {
    if (!walletState?.exists || sending || $('send-confirmation').open) return;
    $('send-asset').value = 'eth';
    $('send-action').value = 'transfer';
    $('send-to').value = walletState.address;
    $('send-amount').value = '0.000001';
    $('send-error').hidden = true;
    updateSendAction();
    $('send-panel').scrollIntoView({block: 'start'});
    $('send-amount').focus({preventScroll: true});
  });
  $('prepare-first-wrap').addEventListener('click', () => {
    if (!walletState?.exists || sending || $('send-confirmation').open) return;
    $('exchange-action').value = 'wrap';
    $('exchange-amount').value = '0.000001';
    updateExchangeAction();
    $('exchange-panel').scrollIntoView({block: 'start'});
    $('exchange-amount').focus({preventScroll: true});
  });
  $('backup-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    $('backup-feedback').textContent = '正在驗證密碼…';
    try {
      const data = await walletRequest('/api/wallet/backup', { password: $('backup-password').value });
      const url = URL.createObjectURL(new Blob([JSON.stringify(data)], { type: 'application/json' }));
      const link = node('a');
      link.href = url;
      link.download = `multi-chain-wallet-evm-${walletState.address}.json`;
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      $('backup-feedback').textContent = '已要求瀏覽器下載加密備份，請妥善保存檔案與密碼。';
    } catch (error) { $('backup-feedback').textContent = errorMessage(error); }
    finally { $('backup-password').value = ''; button.disabled = false; }
  });

  function renderTokens() {
    try { const previous = JSON.parse(localStorage.getItem(tokenStorageKey) || '[]'); const addresses = [...new Set([...(Array.isArray(previous) ? previous : []), ...tokens.keys()])].filter(address => /^0x[0-9a-fA-F]{40}$/.test(address)).slice(-20); localStorage.setItem(tokenStorageKey, JSON.stringify(addresses)); } catch {}
      $('token-list').replaceChildren();
      const selected = $('send-asset').value;
      $('send-asset').replaceChildren(new Option(networkName + ' ' + nativeSymbol, 'eth'));
      for (const item of tokens.values()) {
        const row = node('div', '', 'token-row');
        const symbol = node('strong', item.symbol); symbol.translate = false;
        row.append(symbol, node('p', item.balance, 'token-balance mono'));
        if (item.stale) row.append(node('p', '餘額未更新，顯示上次查詢結果。', 'error'));
        row.append(explorer('token', item.contract));
        $('token-list').append(row);
        $('send-asset').append(Object.assign(new Option(`${item.symbol} · ${item.contract.slice(0, 8)}…`, item.contract.toLowerCase()), {translate:false}));
      }
      $('send-asset').value = [...$('send-asset').options].some(option => option.value === selected) ? selected : 'eth';
    updateSendAction();
  }
  function updateSendAction() {
    const token = $('send-asset').value !== 'eth';
    const tokenInfo = tokens.get($('send-asset').value.toLowerCase());
    $('token-action-group').hidden = !token;
    const approve = token && $('send-action').value === 'approve';
    $('send-to-label').textContent = approve ? '被授權地址（spender）' : '收款地址';
    $('send-amount-label').textContent = token && !tokenInfo?.trustedMetadata ? '最小單位數量（整數）' : '金額';
    $('send-asset-hint').textContent = token && !tokenInfo?.trustedMetadata ? `自訂代幣直接簽署這個最小單位整數；請依可信來源核對合約與精度。手續費以 ${nativeSymbol} 支付。` : (approve ? `只授權指定數量；輸入 0 代表撤銷。手續費以 ${nativeSymbol} 支付。` : `手續費另以 ${nativeSymbol} 支付。`);
    if (tokens.get($('send-asset').value.toLowerCase())?.stale) $('send-asset-hint').textContent += ' 此代幣餘額未更新，預估費用時會重新查核。';
  }
  $('send-asset').addEventListener('change', updateSendAction);
  $('send-action').addEventListener('change', updateSendAction);
  $('send-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    button.textContent = '正在預估與模擬…';
    $('send-error').hidden = true;
    quote = undefined;
    try {
      const token = $('send-asset').value !== 'eth';
      const tokenInfo = token ? tokens.get($('send-asset').value.toLowerCase()) : undefined;
      const inputAmount = $('send-amount').value.trim();
      const data = await walletRequest('/api/wallet/quote', { action: token ? $('send-action').value : 'eth', to: $('send-to').value.trim(), amount: token && !tokenInfo?.trustedMetadata ? '' : inputAmount, amountRaw: token && !tokenInfo?.trustedMetadata ? inputAmount : '', contract: token ? $('send-asset').value : '' });
      openConfirmation(data);
    } catch (error) { showWalletError('send-error', error); }
    finally { button.disabled = false; button.textContent = '預估費用並核對'; }
  });
  let escrowInfo;
  // Preserve only the polling target across transient errors; operations still require fresh escrowInfo.
  let escrowPollingContract = '';
  let escrowOrder;
  let escrowBusy = false;
  let escrowQuery = 0;
  const escrowStates = {none:'尚未付款',funded:'款項由合約保管',released:'已放款給收款人',refunded:'已退回付款人'};
  const escrowDraftKey = 'multi-chain-wallet:escrow-draft:' + networkPrefix;

  function escrowUnits(value) {
    if (!/^[0-9]+(?:\.[0-9]{1,6})?$/.test(value)) throw new Error('付款金額請輸入最多 6 位小數的正數');
    const [whole, fraction = ''] = value.split('.');
    const raw = BigInt(whole) * 1000000n + BigInt(fraction.padEnd(6,'0'));
    if (raw <= 0n || raw >= (1n << 256n)) throw new Error('付款金額請輸入最多 6 位小數的正數');
    return raw;
  }
  function saveEscrowDraft() {
    try { sessionStorage.setItem(escrowDraftKey, JSON.stringify({reference:$('escrow-reference').value,seller:$('escrow-seller').value,amount:$('escrow-amount').value,buyer:$('escrow-lookup-buyer').value,lookup:$('escrow-lookup-reference').value})); } catch {}
  }
  try {
    const draft = JSON.parse(sessionStorage.getItem(escrowDraftKey) || 'null');
    if (draft) for (const [key,id] of Object.entries({reference:'escrow-reference',seller:'escrow-seller',amount:'escrow-amount',buyer:'escrow-lookup-buyer',lookup:'escrow-lookup-reference'})) {
      if (typeof draft[key] === 'string') $(id).value = draft[key].slice(0,66);
    }
  } catch {}
  function setEscrowBusy(busy) {
    escrowBusy = busy;
    for (const id of ['escrow-fund-preview','escrow-lookup','escrow-release','escrow-refund','escrow-refresh','escrow-reference','escrow-seller','escrow-amount','escrow-lookup-buyer','escrow-lookup-reference']) $(id).disabled = busy;
    for (const button of document.querySelectorAll('button[data-order-id]')) button.disabled = busy;
  }
  function renderEscrowHistory() {
    const rows = historySnapshot.filter(tx => tx.escrowContract?.toLowerCase() === escrowInfo?.contract?.toLowerCase() && tx.orderId || tx.action === 'approve' && tx.to?.toLowerCase() === escrowInfo?.contract?.toLowerCase()).slice(0,5);
    $('escrow-history').replaceChildren();
    for (const tx of rows) {
      const row = node('p');
      row.append(node('strong',transactionActionLabel(tx)),node('span',` · ${tx.amount} USDC · `),node('span',transactionLabel(tx)),document.createTextNode(' '),explorer('tx',tx.hash));
      if (tx.orderId && tx.escrowBuyer) {
        const reference = node('span',orderKeyLabel(tx.orderId));
        reference.translate = false;
        row.append(document.createTextNode(' · '),reference,document.createTextNode(' '),escrowHistoryButton(tx));
      }
      $('escrow-history').append(row);
    }
    if (!rows.length) $('escrow-history').append(node('p','完成授權或付款後，交易紀錄會顯示在這裡。','muted'));
  }
  function orderKeyLabel(reference) {
    return /^0x[0-9a-fA-F]{64}$/.test(reference) ? reference.slice(0,10) + '…' + reference.slice(-6) : reference;
  }
  function escrowHistoryButton(tx) {
    const button = node('button','查看訂單','secondary');
    button.type = 'button';
    button.dataset.orderId = tx.orderId;
    button.addEventListener('click',async () => {
      if (sending || escrowBusy) return;
      button.disabled = true;
      try {
        // Load the configured contract before using a record from this account's history.
        await selectContractTab(true);
        location.hash = 'vault-panel';
        if (!escrowInfo?.enabled || tx.escrowContract.toLowerCase() !== escrowInfo.contract.toLowerCase()) {
          showWalletError('escrow-error',new Error('這筆訂單屬於其他託管合約，無法在目前合約操作。'));
          return;
        }
        $('escrow-lookup-buyer').value = tx.escrowBuyer;
        $('escrow-lookup-reference').value = tx.orderId;
        await lookupEscrow();
      } finally { button.disabled = false; }
    });
    return button;
  }
  async function refreshEscrow() {
    if (escrowBusy || !walletState?.exists || walletState.chainId !== 11155111) return;
    setEscrowBusy(true);
    escrowInfo = undefined;
    escrowOrder = undefined;
    $('escrow-enabled').hidden = true;
    $('escrow-order').hidden = true;
    $('escrow-error').hidden = true;
    $('escrow-status').textContent = '正在更新付款託管狀態…';
    try {
      escrowInfo = await walletRequest('/api/wallet/escrow');
      escrowPollingContract = escrowInfo.enabled ? escrowInfo.contract : '';
      $('escrow-status').textContent = escrowInfo.enabled ? '' : '付款託管尚未開放。';
      $('escrow-enabled').hidden = !escrowInfo.enabled;
      $('escrow-contract-link').replaceChildren();
      if (escrowInfo.enabled) {
        $('escrow-contract-link').append(explorer('address',escrowInfo.contract));
        $('escrow-balance').textContent = `可用餘額：${escrowInfo.balance} USDC · 已授權：${escrowInfo.allowance} USDC`;
        if (!$('escrow-lookup-buyer').value) $('escrow-lookup-buyer').value = walletState.address;
        renderEscrowHistory();
      }
    } catch (error) {
      $('escrow-status').textContent = '暫時無法讀取付款狀態，請重試。';
      showWalletError('escrow-error',error);
    } finally { setEscrowBusy(false); }
    if (escrowInfo?.enabled && $('escrow-lookup-reference').value) await lookupEscrow(false);
  }
  function showEscrowOrder(order, focus) {
    escrowOrder = order;
    $('escrow-order').hidden = false;
    $('escrow-order-state').textContent = escrowStates[order.state] || '狀態待確認';
    $('escrow-order-details').replaceChildren(details([['訂單編號',order.orderId],['原付款人',order.buyer]]));
    $('escrow-release').hidden = $('escrow-refund').hidden = true;
    if (order.state === 'none') {
      $('escrow-role-hint').textContent = '找不到已付款的訂單，請確認付款人地址和訂單編號。';
    } else {
      $('escrow-new').open = false;
      $('escrow-search').open = false;
      $('escrow-order-details').append(details([['收款人',order.seller],['金額',`${order.amount} USDC`],['鏈上確認',order.finalized ? '已最終確認' : '已收錄，仍待最終確認']]));
      const buyer = order.buyer.toLowerCase() === walletState.address.toLowerCase();
      const seller = order.seller.toLowerCase() === walletState.address.toLowerCase();
      if (order.state === 'funded') {
        $('escrow-release').hidden = !buyer;
        $('escrow-refund').hidden = !seller;
        $('escrow-role-hint').textContent = buyer ? '你是付款人。確認對方已完成約定，再放款給收款人。退款要由收款人操作。' : seller ? '你是收款人。可將全額退給原付款人。要收款，請等付款人放款。' : '這個錢包只能查看。請切換到付款人或收款人的錢包。';
      } else $('escrow-role-hint').textContent = '這筆訂單已完成，不能再放款或退款。';
    }
    if (focus) $('escrow-order').focus();
  }
  async function lookupEscrow(focus = true) {
    if (escrowBusy || !escrowInfo?.enabled || !$('escrow-lookup-form').reportValidity()) return;
    const generation = ++escrowQuery;
    setEscrowBusy(true);
    escrowOrder = undefined;
    $('escrow-order').hidden = true;
    $('escrow-error').hidden = true;
    saveEscrowDraft();
    try {
      const order = await walletRequest('/api/wallet/escrow/order?' + new URLSearchParams({buyer:$('escrow-lookup-buyer').value.trim(),orderId:$('escrow-lookup-reference').value.trim()}));
      if (generation === escrowQuery) showEscrowOrder(order,focus);
    } catch (error) { if (generation === escrowQuery) showWalletError('escrow-error',error); }
    finally { setEscrowBusy(false); }
  }
  async function quoteEscrow(action) {
    if (escrowBusy || sending || !escrowInfo?.enabled) return;
    if (action === 'escrow_fund' && !$('escrow-fund-form').reportValidity()) return;
    setEscrowBusy(true);
    $('escrow-error').hidden = true;
    try {
      let requestBody;
      let approval;
      if (action === 'escrow_fund') {
        const seller = $('escrow-seller').value.trim();
        if ([walletState.address,escrowInfo.contract].some(address=>address.toLowerCase()===seller.toLowerCase()) || /^0x0{40}$/i.test(seller)) throw new Error('收款人須為另一個錢包地址');
        const amount = $('escrow-amount').value.trim();
        const raw = escrowUnits(amount);
        // Refresh allowances before deciding whether to approve; never infer payment from approval.
        const current = await walletRequest('/api/wallet/escrow');
        escrowPollingContract = current.enabled ? current.contract : '';
        if (!current.enabled) throw new Error('付款託管尚未開放。');
        escrowInfo = current;
        const allowance = current.allowance === '0' ? 0n : escrowUnits(current.allowance);
        const balance = current.balance === '0' ? 0n : escrowUnits(current.balance);
        if (balance < raw) throw new Error('測試 USDC 餘額不足，請先領取測試幣。');
        // A duplicate reference is rejected even before an unnecessary approval.
        const existing = await walletRequest('/api/wallet/escrow/order?' + new URLSearchParams({buyer:walletState.address,orderId:$('escrow-reference').value.trim()}));
        if (existing.state !== 'none') throw new Error('此訂單編號已付款，請查詢原訂單。');
        if (allowance < raw) {
          const revoke = allowance > 0n;
          requestBody = {action:'approve',to:current.contract,contract:current.token,amount:revoke?'0':amount,amountRaw:revoke?'0':String(raw)};
          approval = {orderId:$('escrow-reference').value.trim(),buyer:walletState.address,revoke};
        } else requestBody = {action,to:$('escrow-seller').value.trim(),amount,orderId:$('escrow-reference').value.trim(),buyer:walletState.address};
        saveEscrowDraft();
      } else {
        if (!escrowOrder || escrowOrder.state !== 'funded') throw new Error('請先查詢款項仍由合約保管的訂單。');
        requestBody = {action,to:action==='escrow_release'?escrowOrder.seller:escrowOrder.buyer,amount:'',orderId:escrowOrder.orderId,buyer:escrowOrder.buyer};
      }
      const data = await walletRequest('/api/wallet/quote',requestBody);
      if (approval) data.escrowApproval = approval;
      openConfirmation(data);
    } catch (error) { showWalletError('escrow-error',error); }
    finally { setEscrowBusy(false); }
  }
  function selectContractTab(escrow) {
    $('vault-eth-content').hidden = escrow;
    $('escrow-content').hidden = !escrow;
    for (const [id,selected] of [['contract-tab-vault',!escrow],['contract-tab-escrow',escrow]]) {
      $(id).setAttribute('aria-selected',String(selected));
      $(id).tabIndex = selected ? 0 : -1;
    }
    if (escrow) return refreshEscrow();
  }
  for (const id of ['contract-tab-vault','contract-tab-escrow']) {
    $(id).addEventListener('click',()=>selectContractTab(id==='contract-tab-escrow'));
    $(id).addEventListener('keydown',event=>{
      if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return;
      event.preventDefault();
      const escrow = event.key === 'End' || event.key !== 'Home' && id === 'contract-tab-vault';
      selectContractTab(escrow);$(escrow?'contract-tab-escrow':'contract-tab-vault').focus();
    });
  }
  $('escrow-fund-form').addEventListener('submit',event=>{event.preventDefault();quoteEscrow('escrow_fund');});
  $('escrow-fund-form').addEventListener('input',saveEscrowDraft);
  $('escrow-lookup-form').addEventListener('submit',event=>{event.preventDefault();lookupEscrow();});
  $('escrow-lookup-form').addEventListener('input',()=>{escrowQuery++;escrowOrder=undefined;$('escrow-order').hidden=true;saveEscrowDraft();});
  $('escrow-release').addEventListener('click',()=>quoteEscrow('escrow_release'));
  $('escrow-refund').addEventListener('click',()=>quoteEscrow('escrow_refund'));
  $('escrow-refresh').addEventListener('click',refreshWallet);

  async function refreshVault() {
    if (vaultBusy || !walletState?.exists || walletState.chainId !== 11155111) return;
    vaultBusy = true;
    vaultEnabled = false;
    $('vault-deposit').disabled = $('vault-withdraw').disabled = $('vault-amount').disabled = $('vault-preview').disabled = true;
    $('vault-refresh').disabled = true;
    $('vault-error').hidden = true;
    $('vault-notice').hidden = false;
    $('vault-status').textContent = '正在更新合約餘額…';
    $('vault-status-hint').hidden = true;
    $('vault-wallet-balance').textContent = '—';
    $('vault-deposited-balance').textContent = '—';
    $('vault-contract-view').replaceChildren();
    try {
      const vault = await walletRequest('/api/wallet/vault');
      if (!vault.enabled) {
        $('vault-status').textContent = '此環境尚未開放合約操作';
        $('vault-status-hint').textContent = '合約尚未設定，目前無法存入或取回 ETH。';
        $('vault-status-hint').hidden = false;
        return;
      }
      const balance = await request(`/api/balance?address=${encodeURIComponent(walletState.address)}`);
      $('vault-wallet-balance').textContent = `${balance.eth} ETH`;
      $('vault-deposited-balance').textContent = `${vault.balance} ETH`;
      const link = explorer('address', vault.contract);
      link.textContent = '在 Etherscan 查看合約 ↗';
      link.title = vault.contract;
      $('vault-contract-view').append(link);
      if (BigInt(balance.wei) === 0n) $('vault-status').textContent = '錢包還沒有測試 ETH，請先領取以支付存入或取回的手續費。';
      else if (BigInt(vault.balanceRaw) === 0n) $('vault-status').textContent = '此錢包尚未存入 ETH。可先試著存入一小筆，再取回錢包。';
      else $('vault-notice').hidden = true;
      vaultEnabled = true;
    } catch (error) {
      $('vault-status').textContent = '暫時無法讀取餘額';
      $('vault-status-hint').textContent = '請按「更新餘額與紀錄」重試，確認餘額後再操作。';
      $('vault-status-hint').hidden = false;
      showWalletError('vault-error', error);
    } finally {
      vaultBusy = false;
      $('vault-refresh').disabled = false;
      $('vault-form').hidden = $('vault-summary').hidden = $('vault-history-section').hidden = !vaultEnabled;
      $('vault-deposit').disabled = $('vault-withdraw').disabled = $('vault-amount').disabled = $('vault-preview').disabled = !vaultEnabled;
    }
  }
  async function quoteVault() {
    if (!vaultEnabled || vaultBusy || sending) return;
    if (!$('vault-form').reportValidity()) return;
    vaultBusy = true;
    $('vault-deposit').disabled = $('vault-withdraw').disabled = $('vault-amount').disabled = $('vault-preview').disabled = $('vault-refresh').disabled = true;
    $('vault-preview').textContent = '正在預估費用…';
    $('vault-error').hidden = true;
    try {
      const action = $('vault-withdraw').checked ? 'vault_withdraw' : 'vault_deposit';
      const data = await walletRequest('/api/wallet/quote', {action, to:walletState.address, amount:$('vault-amount').value.trim()});
      openConfirmation(data);
    } catch (error) { showWalletError('vault-error', error); }
    finally {
      vaultBusy = false;
      $('vault-deposit').disabled = $('vault-withdraw').disabled = $('vault-amount').disabled = $('vault-preview').disabled = !vaultEnabled;
      $('vault-refresh').disabled = false;
      $('vault-preview').textContent = '預估費用並核對';
    }
  }
  $('vault-form').addEventListener('change', event => {
    if (event.target.name !== 'vault-action') return;
    const withdraw = $('vault-withdraw').checked;
    $('vault-direction').textContent = withdraw ? '智慧合約 → 目前錢包' : '目前錢包 → 智慧合約';
    $('vault-amount-label').textContent = withdraw ? '取回金額（ETH）' : '存入金額（ETH）';
    $('vault-amount-hint').textContent = withdraw ? '只能取回此錢包存入的 ETH，款項會回到同一個錢包地址。' : '存入後，這筆 ETH 會記在目前錢包地址的合約餘額中。';
    $('vault-error').hidden = true;
  });
  $('vault-form').addEventListener('submit', event => { event.preventDefault(); quoteVault(); });
  $('vault-refresh').addEventListener('click', refreshWallet);
  window.addEventListener('wallet-view', event => { if (event.detail === 'vault-panel') { if ($('escrow-content').hidden) refreshVault(); else refreshEscrow(); } });

  function openConfirmation(data) {
    if ($('send-confirmation').open || sending) return;
    quote = data;
    $('confirm-title').textContent = data.action === 'vault_deposit' ? '確認存入合約' : data.action === 'vault_withdraw' ? '確認取回錢包' : '確認這筆交易';
    const entries = [['操作', actionLabels[data.action] || '資產操作'], ['網路', networkName], ['發送地址', data.from], [data.action === 'approve' ? '被授權地址' : '收款地址', data.to], ['數量', `${data.amount} ${data.symbol}`], ['執行費用上限', `${data.maxFeeEth} ${nativeSymbol}`], [data.rollupFeeEth ? '預估總扣款（含費用預留）' : '最多扣除 ' + nativeSymbol, `${data.totalEth} ${nativeSymbol}`], ['報價有效至', time(data.expiresAt)]];
    if (data.rollupFeeEth) entries.push(['L1／營運費預留', `${data.rollupFeeEth} ETH（估算含緩衝；上鏈費用仍可能變動）`]);
    if (data.escrow) {
      const escrowAction = data.escrow.action || data.action;
      $('confirm-title').textContent = data.action === 'speedup' ? '確認加速原交易' : escrowAction === 'escrow_fund' ? '確認付款至合約' : escrowAction === 'escrow_release' ? '確認放款給收款人' : '確認全額退款';
      if (data.action === 'speedup') entries.push(['原操作',actionLabels[escrowAction]]);
      entries.splice(2,2,['付款人',data.escrow.buyer],['收款人',data.escrow.seller]);
      entries.push(['訂單編號',data.escrow.orderId],['託管合約',data.contract],['代幣合約',data.escrow.token],['資金去向',escrowAction==='escrow_fund'?'目前錢包 → 託管合約':escrowAction==='escrow_release'?'託管合約 → 收款人':'託管合約 → 原付款人']);
      entries.push(['操作說明',escrowAction==='escrow_fund'?'款項會留在合約，等你放款給收款人，或由收款人退款。':'交易成功後會轉出全部款項，不能再放款或退款。']);
    } else if (data.escrowApproval) {
      $('confirm-title').textContent = data.escrowApproval.revoke ? '先撤銷舊授權' : '確認本次 USDC 授權';
      entries.push(['操作說明',data.escrowApproval.revoke?'先取消舊的授權，再授權這次要付的金額。':'這一步只授權，不會付款。授權成功後，再按「核對付款資料」。']);
    } else if (data.action === 'vault_deposit') entries.splice(2, 2, ['扣款錢包', data.from], ['智慧合約', data.contract]);
    else if (data.action === 'vault_withdraw') entries.splice(2, 2, ['智慧合約', data.contract], ['收款錢包', data.to]);
    else if (data.contract) entries.splice(4, 0, ['代幣合約', data.contract]);
    if (data.action === 'vault_withdraw') entries.push(['取回說明', '取回的 ETH 會回到目前錢包；錢包另付 Gas，最多扣除欄位僅為費用上限。']);
    if (data.exchange) {
      entries.push(['預估收到', `${data.exchange.expectedOut} ${data.exchange.symbolOut}`], ['最低收到', `${data.exchange.minimumOut} ${data.exchange.symbolOut}`], ['收到資產', data.exchange.tokenOut]);
      if (data.exchange.router) entries.push(['兌換合約', data.exchange.router], ['滑價', `${data.exchange.slippageBps / 100}%`], ['交易截止時間', time(data.exchange.deadline)]);
    }
    $('quote-details').replaceChildren(details(entries));
    $('confirmation-fee-hint').textContent=data.rollupFeeEth?'執行費有簽署上限；L1／營運費為預留估算，無法由這筆交易設定絕對上限。費用變動需重新預估。':'最高費用是上限，實際手續費依交易執行結果而定。報價逾時需重新預估。';
    if (data.action === 'speedup' || data.action === 'cancel') $('quote-details').append(node('p', '使用相同 Nonce 與較高手續費競爭收錄。原交易仍可能先成功；送出取消不代表已取消。', 'error'));
    const raw = document.createElement('details');
    raw.append(node('summary', '檢視實際簽署內容'), details([['Chain ID', walletState.chainId], ['最小單位', data.amountRaw], ['方法', data.method], ['Nonce', data.nonce], ['Gas limit', data.gasLimit], ['Max fee / gas', `${data.maxFeePerGas} wei`], ['Priority fee / gas', `${data.maxPriorityFeePerGas} wei`]]), node('pre', data.data || '0x'));
    if (data.exchange?.pool) raw.append(details([['交易池', data.exchange.pool]]));
    $('quote-details').append(raw);
    $('approval-warning').hidden = data.action !== 'approve';
    $('confirm-error').hidden = true;
    $('check-send-history').hidden = true;
    $('check-send-history').className = 'secondary';
    $('confirm-send-button').className = 'submit-button';
    $('cancel-send').textContent = '取消';
    $('confirm-send-button').textContent = `簽署並送出至 ${networkName}`;
    $('send-password').value = '';
    $('send-confirmation').showModal();
    $('cancel-send').focus();
  }
  $('cancel-send').addEventListener('click', () => { if (!sending) $('send-confirmation').close(); });
  $('send-confirmation').addEventListener('cancel', event => { if (sending) event.preventDefault(); });
  $('send-confirmation').addEventListener('close', () => { $('send-password').value = ''; quote = undefined; });
  async function completeSend(data, submittedQuote) {
    if (flow && submittedQuote.flowID === flow.id) { flow.pending = {quoteID:submittedQuote.id,hash:data.hash,kind:submittedQuote.flowKind}; saveFlow(); }
    $('send-confirmation').close();
    showSent(data);
    if (submittedQuote.escrow || submittedQuote.escrowApproval) {
      const payment = submittedQuote.escrow || submittedQuote.escrowApproval;
      $('escrow-lookup-buyer').value = payment.buyer;
      $('escrow-lookup-reference').value = payment.orderId;
      $('escrow-next-step').textContent = submittedQuote.escrowApproval ? '授權已送出。等紀錄顯示成功，再按「核對付款資料」付款。' : '交易已送出，請查看訂單狀態和付款紀錄。';
      saveEscrowDraft();
    }
    await refreshWallet();
    await refreshActivity();
    if (submittedQuote.escrow && !$('escrow-order').hidden) $('escrow-order').focus();
    if ((submittedQuote.action || '').startsWith('vault_') && !$('vault-history-section').hidden) {
      $('vault-history-section').tabIndex = -1;
      $('vault-history-section').focus();
    }
  }
  $('check-send-history').addEventListener('click',async () => {
    if (sending || !quote) return;
    const original = quote;
    sending = true;
    $('check-send-history').disabled = $('confirm-send-button').disabled = $('cancel-send').disabled = true;
    try {
      const history = await walletRequest('/api/wallet/history');
      renderHistory(history.transactions,history.refreshError || '',Boolean(history.canCreateTransaction));
      const transaction = history.transactions.find(tx => tx.quoteId === original.id);
      if (transaction) await completeSend(transaction,original);
      else showWalletError('confirm-error',new Error('尚未找到原交易，仍無法確認結果。請稍後再查，或重試原交易。'));
    } catch { showWalletError('confirm-error',new Error('暫時無法查詢原交易。請稍後再查，不要建立另一筆付款。')); }
    finally {
      sending = false;
      $('check-send-history').disabled = $('confirm-send-button').disabled = $('cancel-send').disabled = false;
    }
  });
  $('confirm-send-form').addEventListener('submit', async event => {
    event.preventDefault();
    if (sending || !quote) return;
    if (!quote.sendUncertain && Date.now() >= new Date(quote.expiresAt).getTime()) { showWalletError('confirm-error', new Error('報價已過期，請取消並重新預估。')); return; }
    sending = true;
    $('confirm-send-button').disabled = true;
    $('cancel-send').disabled = true;
    $('check-send-history').disabled = true;
    $('confirm-send-button').textContent = '正在簽署與廣播…';
    $('confirm-error').hidden = true;
    const submittedQuote = quote;
    try {
      if(flow && submittedQuote.flowID===flow.id){flow.pending={quoteID:submittedQuote.id,hash:'',kind:submittedQuote.flowKind};saveFlow();}
      const data = await walletRequest('/api/wallet/send', { quoteId: quote.id, password: $('send-password').value });
      if (!/^0x[0-9a-fA-F]{64}$/.test(data.hash)) throw new Error('無法讀取伺服器回應，請稍後再試。');
      await completeSend(data,submittedQuote);
    } catch (error) {
      if (error.code === 'send_rejected' && flow && flow.id === submittedQuote.flowID && flow.pending?.quoteID === submittedQuote.id) {
        flow.pending = undefined;
        saveFlow();
      }
      if (error.code === 'send_rejected') showWalletError('confirm-error',error);
      else {
        submittedQuote.sendUncertain = true;
        $('check-send-history').hidden = false;
        $('check-send-history').className = 'submit-button';
        $('confirm-send-button').className = 'secondary';
        $('cancel-send').textContent = '關閉';
        $('confirmation-fee-hint').textContent = '查詢不會送出交易。若需重試，會沿用原交易，不會建立另一筆付款。';
        showWalletError('confirm-error',new Error('交易結果待確認。請先查詢原交易，不要重新建立付款。'));
      }
    }
    finally {
      $('send-password').value = '';
      sending = false;
      renderFlow();
      $('confirm-send-button').disabled = false;
      $('cancel-send').disabled = false;
      $('check-send-history').disabled = false;
      $('confirm-send-button').textContent = submittedQuote.sendUncertain ? '重試原交易' : `簽署並送出至 ${networkName}`;
    }
  });
  async function refreshTokens() {
    if (tokensRefreshing) { tokenRefreshQueued = true; return; }
    tokensRefreshing = true;
    $('token-list').setAttribute('aria-busy', 'true');
    try {
      do {
        tokenRefreshQueued = false;
        let saved = [];
        try { const data = JSON.parse(localStorage.getItem(tokenStorageKey) || "[]"); if (Array.isArray(data)) saved = data.filter(address => /^0x[0-9a-fA-F]{40}$/.test(address)).slice(0,20); } catch {}
        const addresses = [...new Set([
          walletState.exchange?.weth, walletState.exchange?.usdc, ...saved,
          ...tokens.keys(),
        ].filter(Boolean).map(address => address.toLowerCase()))];
        let failed = false;
        // Coalesce refreshes during a read, then query again for changes after a send.
        for (const contract of addresses) {
          try { tokens.set(contract, await walletRequest('/api/wallet/token', {contract})); }
          catch {
            const token = tokens.get(contract);
            if (token) token.stale = true;
            failed = true;
          }
        }
        renderTokens();
        if (failed) showWalletError('token-error', new Error('部分代幣餘額無法更新，請稍後重試。'));
        else $('token-error').hidden = true;
      } while (tokenRefreshQueued);
    } finally {
      tokensRefreshing = false;
      $('token-list').setAttribute('aria-busy', 'false');
    }
  }

  function updateExchangeAction() {
    const action = $('exchange-action').value;
    const swap = ['weth-usdc','usdc-weth','eth-usdc','usdc-eth'].includes(action);
    $('swap-options').hidden = !swap;
    $('swap-status').textContent = '選好支付數量後，可先查詢是否需要授權。';
    $('exchange-error').hidden = true;
    $('pool-results').replaceChildren();
    const config = walletState?.exchange;
    if (!config) return;
    $('exchange-route').textContent = swap ? `Sepolia Router：${config.router}。收到的資產回到本錢包。` : `Sepolia WETH：${config.weth}。包裝／解包比率 1:1，另付 ETH gas。`;
  }
  $('exchange-action').addEventListener('change', updateExchangeAction);

  async function exchangeQuote(approval) {
    const button = approval === 'approve' ? $('swap-approve') : approval === 'revoke' ? $('swap-revoke') : $('exchange-submit');
    if (button.disabled) return;
    button.disabled = true;
    $('exchange-error').hidden = true;
    try {
      const action = $('exchange-action').value;
      const config = walletState.exchange;
      const input = action.startsWith('usdc-') ? config.usdc : config.weth;
      const output = action.startsWith('usdc-') ? config.weth : config.usdc;
      let payload;
      if (approval) payload = {action:'approve',to:config.router,contract:input,amount:approval === 'revoke' ? '0' : $('exchange-amount').value.trim()};
      else if (action === 'wrap' || action === 'unwrap') payload = {action,to:walletState.address,amount:$('exchange-amount').value.trim()};
      else payload = {action:'swap',to:walletState.address,contract:input,tokenOut:output,amount:$('exchange-amount').value.trim(),poolFee:Number($('swap-fee').value),slippageBps:Number($('swap-slippage').value)};
      if (!approval && ['eth-usdc','usdc-eth'].includes(action)) { await advanceFlow(); return; }
      if (payload.action === 'swap' && payload.poolFee === 0) payload.poolFee = await comparePools();
      const data = await walletRequest('/api/wallet/quote', payload);
      openConfirmation(data);
    } catch (error) { showWalletError('exchange-error', error); }
    finally { button.disabled = false; }
  }
  $('exchange-form').addEventListener('submit', event => {event.preventDefault();exchangeQuote();});
  $('swap-check').addEventListener('click', async () => {
    const button = $('swap-check');
    if (button.disabled) return;
    const action = $('exchange-action').value;
    const amount = $('exchange-amount').value.trim();
    const config = walletState.exchange;
    const contract = action.startsWith('usdc-') ? config.usdc : config.weth;
    button.disabled = true;
    $('swap-status').textContent = '正在查詢鏈上餘額與授權…';
    try {
      const token = await walletRequest('/api/wallet/token', {contract, spender: config.router});
      if ($('exchange-action').value !== action || $('exchange-amount').value.trim() !== amount) return;
      let next = '請填入支付數量，再查詢下一步。';
      const parts = amount.split('.');
      if (/^\d+(\.\d+)?$/.test(amount) && (parts[1] || '').length <= token.decimals) {
        const raw = BigInt(parts[0] + (parts[1] || '').padEnd(token.decimals, '0'));
        if (raw > 0n) {
          if (raw > BigInt(token.balanceRaw)) next = '支付餘額不足，請先收款或包裝 ETH。';
          else if (BigInt(token.allowanceRaw) >= raw) next = '額度足夠，可直接取得兌換報價。';
          else if (BigInt(token.allowanceRaw) > 0n) next = '額度不足；先撤銷既有授權，等待成功後授權本次數量。';
          else next = '請授權本次數量，等待交易成功後取得兌換報價。';
        }
      }
      $('swap-status').textContent = `可用 ${token.balance} ${token.symbol} · 目前授權 ${token.allowance} ${token.symbol}。${next} 每次送出前仍會重新查核。`;
    } catch (error) {
      if ($('exchange-action').value === action) $('swap-status').textContent = `查詢失敗：${error.message}`;
    } finally { button.disabled = false; }
  });
  $('exchange-amount').addEventListener('input', () => { $('pool-results').replaceChildren(); $('swap-status').textContent = '數量已變更，請重新查詢餘額與授權。'; });
  $('swap-approve').addEventListener('click', () => exchangeQuote('approve'));
  $('swap-revoke').addEventListener('click', () => exchangeQuote('revoke'));

  let comparisonGeneration = 0;
  async function comparePools() {
    comparisonGeneration += 1;
    const generation = comparisonGeneration;
    const action=$('exchange-action').value, amount=$('exchange-amount').value.trim(), config=walletState.exchange;
    const input=action.startsWith('usdc-')?config.usdc:config.weth, output=action.startsWith('usdc-')?config.weth:config.usdc;
    $('pool-results').textContent='正在比較四個費率的鏈上報價…';
    const result=await walletRequest('/api/wallet/exchange/pools',{contract:input,tokenOut:output,amount});
    if (generation!==comparisonGeneration || action!==$('exchange-action').value || amount!==$('exchange-amount').value.trim()) throw new Error('輸入已變更，請重新比較。');
    $('pool-results').replaceChildren(node('p','依預估收到數量排序選池；未扣除 gas，各池查詢時間可能不同。送出前會重新報價與模擬。'));
    for(const pool of result.pools) $('pool-results').append(node('p',`${pool.fee/10000}% · ${pool.error || pool.output+' '+result.symbol}${pool.fee===result.bestFee?' · 本次輸出最高':''}`));
    if(!result.bestFee)throw new Error('目前沒有可用的交易池報價。');
    return result.bestFee;
  }
  $('compare-pools').addEventListener('click',async()=>{
    const button=$('compare-pools');if(button.disabled)return;button.disabled=true;
    try{await comparePools();}catch(error){showWalletError('exchange-error',error);}finally{button.disabled=false;}
  });
  function saveFlow() {
    try { if(flow)sessionStorage.setItem(flowKey,JSON.stringify(flow));else sessionStorage.removeItem(flowKey); }
    catch { $('exchange-error').textContent='瀏覽器無法保存引導進度；請保留交易雜湊，重新整理後從歷史確認。';$('exchange-error').hidden=false; }
    renderFlow();
  }
  function renderFlow(message) {
    const active=flow && flow.phase!=='done';
    $('exchange-action').disabled=Boolean(active);
    $('exchange-amount').readOnly=Boolean(active);
    $('swap-approve').disabled=Boolean(active);
    $('swap-revoke').disabled=Boolean(active);
    $('flow-stop').hidden=!flow;
    $('flow-stop').disabled=sending;
    const phases={wrap:'包裝 ETH 成為 WETH',swap:'查核授權並兌換',unwrap:'將本次收到的 WETH 解包成 ETH',done:'已完成本次引導；收據仍可在交易紀錄核對'};
    $('exchange-workflow').textContent=message || (flow ? `${flow.direction==='eth-usdc'?'ETH → USDC':'USDC → ETH'} · ${flow.pending?'等待 '+(flow.pending.hash||'原報價')+' 的鏈上收據':phases[flow.phase]}` : '選擇 ETH ↔ USDC 時，系統會依序準備必要交易；每筆皆需獨立確認與簽署。');
    $('exchange-submit').textContent=flow?.pending?'更新進度':active?'繼續下一步並核對': '取得鏈上報價並核對';
  }
  $('flow-stop').addEventListener('click',()=>{if(!sending&&!$('send-confirmation').open){flow=undefined;saveFlow();}});
  async function reconcileFlow(historyFresh = false) {
    if(!flow?.pending)return;
    const current=flow,pending=current.pending;
    try {
      if (!historyFresh) {
        const history = await walletRequest('/api/wallet/history').catch(error => {
          renderHistory(historySnapshot, '無法更新紀錄，請稍後再試。', false);
          throw error;
        });
        if (flow !== current || current.pending !== pending) return;
        renderHistory(history.transactions, history.refreshError || '', Boolean(history.canCreateTransaction));
      }
      if(!pending.hash){const known=historySnapshot.find(tx=>tx.quoteId===pending.quoteID);if(!known){renderFlow('尚未找到原報價的交易紀錄。請更新進度，或在原確認視窗重試同一筆報價。');return;}pending.hash=known.hash;saveFlow();}
      const original = historySnapshot.find(tx => tx.hash === pending.hash);
      if (original?.nonceConsumed) { renderFlow(nonceNotice()); return; }
      let effectiveHash = pending.hash;
      let cancelled = false;
      if (original?.replacedBy) {
        const replacement = historySnapshot.find(tx => tx.hash === original.replacedBy);
        if (!replacement) { renderFlow('正在等待替代交易紀錄，請稍後更新。'); return; }
        cancelled = replacement.action === 'cancel' ||
          (replacement.action === 'speedup' && replacement.amount === '0' && replacement.to?.toLowerCase() === walletState.address.toLowerCase());
        if (!cancelled && (replacement.action !== 'speedup' ||
            ['to', 'amount', 'symbol'].some(key => replacement[key] !== original[key]))) {
          throw new Error('替代交易與原步驟內容不同，請核對交易紀錄。');
        }
        effectiveHash = replacement.hash;
      }
      const tx=await request('/api/transactions/'+encodeURIComponent(effectiveHash));
      if(flow!==current || current.pending!==pending)return;
      if(tx.state==='reverted'){current.pending=undefined;saveFlow();renderFlow('此步驟執行失敗，已消耗測試 gas。可結束引導或重新預估。');return;}
      if(tx.state!=='succeeded'){renderFlow();return;}
      if (cancelled) {
        current.pending = undefined;
        saveFlow();
        renderFlow('原步驟已取消，可結束引導或重新預估。');
        return;
      }
      if(pending.kind==='wrap')current.phase='swap';
      if(pending.kind==='swap'){
        if(current.direction==='eth-usdc')current.phase='done';
        else {
          const activity=await request('/api/watch/activity?'+new URLSearchParams({address:walletState.address,hash:effectiveHash}));
          if(activity.state!=='succeeded')throw new Error('尚未取得可驗證的兌換收支。');
          const raw=activity.movements.filter(m=>m.kind==='receive'&&m.asset.toLowerCase()===walletState.exchange.weth.toLowerCase()).reduce((sum,m)=>sum+BigInt(m.raw),0n);
          if(raw<=0n)throw new Error('未找到本次收到的 WETH，請核對收據後手動解包。');
          const digits=raw.toString().padStart(19,'0');current.unwrapAmount=digits.slice(0,-18)+'.'+digits.slice(-18);current.phase='unwrap';
        }
      }
      if(pending.kind==='unwrap')current.phase='done';
      current.pending=undefined;saveFlow();
    }catch(error){renderFlow('尚無法確認此步驟結果，保留原交易，請稍後更新。'+errorMessage(error));}
  }
  async function advanceFlow() {
    if(flow?.phase==='done') { flow=undefined;saveFlow(); }
    if(!flow){
      const direction=$('exchange-action').value,amount=$('exchange-amount').value.trim();
      if(!/^\d+(\.\d+)?$/.test(amount)||!/[1-9]/.test(amount))throw new Error('請輸入大於 0 的數量。');
      flow={id:crypto.randomUUID(),direction,amount,phase:direction==='eth-usdc'?'wrap':'swap'};saveFlow();
    }
    if(flow.pending){await reconcileFlow();return;}
    const current=flow,config=walletState.exchange;
    let payload,kind;
    if(current.phase==='wrap'||current.phase==='unwrap'){
      kind=current.phase;payload={action:kind,to:walletState.address,amount:kind==='unwrap'?current.unwrapAmount:current.amount};
    }else{
      const input=current.direction==='usdc-eth'?config.usdc:config.weth,output=current.direction==='usdc-eth'?config.weth:config.usdc;
      const token=await walletRequest('/api/wallet/token',{contract:input,spender:config.router});
      const parts=current.amount.split('.');if((parts[1]||'').length>token.decimals)throw new Error('支付數量超過代幣精度。');
      const raw=BigInt(parts[0]+(parts[1]||'').padEnd(token.decimals,'0'));
      if(BigInt(token.balanceRaw)<raw)throw new Error('來源代幣餘額不足。');
      if(BigInt(token.allowanceRaw)<raw){
        kind=BigInt(token.allowanceRaw)>0n?'revoke':'approve';payload={action:'approve',contract:input,to:config.router,amount:kind==='revoke'?'0':current.amount};
      }else{
        kind='swap';const fee=Number($('swap-fee').value)||await comparePools();
        payload={action:'swap',to:walletState.address,contract:input,tokenOut:output,amount:current.amount,poolFee:fee,slippageBps:Number($('swap-slippage').value)};
      }
    }
    const data=await walletRequest('/api/wallet/quote',payload);
    if(flow!==current)throw new Error('引導已變更，請重新報價。');
    data.flowID=current.id;data.flowKind=kind;openConfirmation(data);
  }
  setInterval(()=>{if((flow?.pending || historySnapshot.some(tx => ((tx.action || '').match(/^(vault_|escrow_)/) || tx.escrowAction || tx.action === 'approve' && escrowPollingContract && tx.to?.toLowerCase() === escrowPollingContract.toLowerCase()) && ['submitted','pending','broadcast_unknown','receipt_unavailable','reorg_detected'].includes(tx.state))) && !document.hidden && !sending)refreshWallet();},10000);

  function displayAsset(raw, asset) {
    const config = walletState.exchange;
    let decimals, symbol;
    if (asset === nativeSymbol) { decimals = 18; symbol = nativeSymbol; }
    else if (asset.toLowerCase() === (config.weth || '').toLowerCase()) { decimals = 18; symbol = 'WETH'; }
    else if (asset.toLowerCase() === (config.usdc || '').toLowerCase()) { decimals = 6; symbol = '測試 USDC'; }
    else return `${raw} 最小單位 · ${asset}`;
    const negative = raw.startsWith('-');
    const digits = (negative ? raw.slice(1) : raw).padStart(decimals + 1, '0');
    const fraction = digits.slice(-decimals).replace(/0+$/, '');
    return `${negative ? '-' : ''}${digits.slice(0, -decimals)}${fraction ? '.' + fraction : ''} ${symbol}`;
  }

  async function refreshActivity() {
    if (activityLoading || !walletState?.exists) return;
    activityLoading = true;
    $('activity-refresh').disabled = true;
    $('activity-prev').disabled = true;
    $('activity-next').disabled = true;
    $('activity-error').hidden = true;
    $('activity-list').replaceChildren(node('p', '正在查核本頁鏈上收據…', 'muted'));
    $('activity-totals').replaceChildren();
    try {
      const data = await walletRequest(`/api/wallet/activity?page=${activityPage}`);
      $('activity-csv').href = networkPrefix + `/api/wallet/activity?format=csv&page=${activityPage}`;
      $('activity-page').textContent = `第 ${data.page} / ${data.pages} 頁 · 已收錄 ${data.totalTransactions} 筆`;
      $('activity-prev').disabled = data.page <= 1;
      $('activity-next').disabled = data.page >= data.pages;
      $('activity-list').replaceChildren();
      if (!data.transactions.length) $('activity-list').append(node('p', '尚無已收錄的收支；未收錄不代表沒有鏈上交易。可至區塊瀏覽器核對。', 'muted'));
      if (data.incomplete) showWalletError('activity-error', new Error('部分交易尚未確認或無法查核，不列入本頁合計。'));
      for (const total of data.totals) {
        const box = node('div', '', 'activity-totals');
        box.append(node('strong', total.asset === nativeSymbol ? `本頁 ${nativeSymbol} 收支` : `本頁代幣 ${total.asset}`), node('p', `收入 ${displayAsset(total.receivedRaw,total.asset)} · 支出 ${displayAsset(total.sentRaw,total.asset)}`),node('p',`手續費 ${displayAsset(total.feeRaw,total.asset)} · 淨變動 ${displayAsset(total.netRaw,total.asset)}`));
        $('activity-totals').append(box);
      }
      $('activity-updated').textContent = `最後更新：${time(new Date().toISOString())}`;
      const kinds = {receive:'收款',send:'付款',fee:'手續費'};
      for (const tx of data.transactions) {
        const row = node('article', '', 'history-row');
        row.append(node('strong', stateLabels[tx.state] || '無法查核'),explorer('tx',tx.hash));
        if (tx.blockTime) row.append(node('p',`${time(tx.blockTime)} · 區塊 ${tx.block}`));
        if (tx.error) row.append(node('p',tx.error,'error'));
        if (!tx.movements.length) row.append(node('p','尚無可列入的資產變動。授權操作本身不算代幣支出。'));
        for (const movement of tx.movements) {
          const item = node('div', '', `activity-move ${movement.kind}`);
          item.append(node('strong',`${kinds[movement.kind]} ${displayAsset(movement.raw,movement.asset)}`));
          if (movement.kind !== 'fee') item.append(node('p', `對方地址：${movement.counterparty}`));
          const raw = node('details');
          raw.append(node('summary', '鏈上紀錄詳情'), details([['紀錄來源', movement.evidence], ['原始數量', movement.raw], ['資產', movement.asset]]));
          item.append(raw);
          row.append(item);
        }
        $('activity-list').append(row);
      }
    } catch (error) { $('activity-list').replaceChildren(); showWalletError('activity-error',error); }
    finally { activityLoading = false; $('activity-refresh').disabled = false; }
  }
  $('activity-refresh').addEventListener('click', async () => {
    if (activityLoading || $('activity-refresh').disabled) return;
    $('activity-refresh').disabled = true;
    await refreshWallet();
    await refreshActivity();
  });
  window.addEventListener('wallet-view', async event => {
    if (walletState?.exists && !activityLoading && ['history-panel', 'activity-panel'].includes(event.detail)) { await refreshWallet(); await refreshActivity(); }
  });
  $('activity-prev').addEventListener('click',()=>{if(!activityLoading){activityPage -= 1;refreshActivity();}});
  $('activity-next').addEventListener('click',()=>{if(!activityLoading){activityPage += 1;refreshActivity();}});
  $('activity-import-form').addEventListener('submit', async event => {
    event.preventDefault(); const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true; $('activity-error').hidden = true;
    try {
      const result = await walletRequest('/api/wallet/activity/import',{hash:$('activity-hash').value.trim()});
      $('activity-feedback').textContent = `已查核並收錄 ${result.hash}。重複匯入不會重複計帳。`;
      activityPage = 1; await refreshActivity();
    } catch (error) {showWalletError('activity-error',error);}
    finally {button.disabled = false;}
  });
  $('activity-sync-form').addEventListener('submit', async event => {
    event.preventDefault(); const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true; $('activity-error').hidden = true;
    $('activity-feedback').textContent = '正在掃描區塊，請稍候…';
    try {
      const raw = $('activity-from').value.trim();
      const from = raw ? Number(raw) : 0;
      if (!Number.isSafeInteger(from) || from < 0) throw new Error('請輸入有效的區塊號碼。');
      const contracts = [...tokens.keys()];
      let result;
      for (let offset = 0; offset < Math.max(1, contracts.length); offset += 20) {
        const batch = await walletRequest('/api/wallet/activity/sync', {
          from: result ? result.from : from,
          contracts: contracts.slice(offset, offset + 20),
        });
        if (!result && from === 0) $('activity-from').value = String(batch.from);
        result = result ? {...result, to: Math.min(result.to, batch.to), added: result.added + batch.added} : batch;
      }
      $('activity-feedback').textContent = `已同步區塊 ${result.from}–${result.to}，新增 ${result.added} 筆。可保留下一個起始區塊繼續同步，或清空改查最近區塊。`;
      $('activity-from').value = String(result.to + 1);
      activityPage = 1; await refreshActivity();
    } catch (error) {$('activity-feedback').textContent = '';showWalletError('activity-error',error);}
    finally {button.disabled = false;}
  });

  $('keystore-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const file = $('keystore-file').files[0];
      const password = $('keystore-new-password').value;
      if (!file || file.size > 8192) throw new Error('請選擇不超過 8 KB 的加密備份。');
      if (Array.from(password).length < 12 || Array.from(password).length > 128) throw new Error('新密碼需為 12–128 字元。');
      if (password !== $('keystore-confirm-password').value) throw new Error('兩次新密碼不一致。');
      await walletRequest('/api/wallet/import-keystore', {keystore: JSON.parse(await file.text()), password: $('keystore-password').value, newPassword: password});
      $('keystore-form').reset();
      await loadWallet();
    } catch (error) { $('keystore-feedback').textContent = errorMessage(error); }
    finally { $('keystore-password').value = ''; $('keystore-new-password').value = ''; $('keystore-confirm-password').value = ''; button.disabled = false; }
  });
  $('password-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const password = $('new-password').value;
      if (Array.from(password).length < 12 || Array.from(password).length > 128) throw new Error('新密碼需為 12–128 字元。');
      if (password !== $('new-password-confirm').value) throw new Error('兩次新密碼不一致。');
      await walletRequest('/api/wallet/password', {password: $('old-password').value, newPassword: password});
      $('password-feedback').textContent = '密碼已更新，請重新下載加密備份。';
    } catch (error) { $('password-feedback').textContent = errorMessage(error); }
    finally { $('password-form').reset(); button.disabled = false; }
  });
  $('account-select').addEventListener('change', () => { const id = $('account-select').value; location.href = (id ? '/accounts/' + id : '') + networkSuffix + '/' + (location.hash || ''); });
  function accountURL(id) { return (id ? '/accounts/' + id : '') + networkSuffix + '/' + (location.hash || ''); }

  function renderAccountSelect() {
    $('account-select').replaceChildren(...accounts.filter(item => !item.archived).map(item => {
      const address = item.address ? item.address.slice(0, 6) + '…' + item.address.slice(-4) : '待完成設定';
      const option = new Option((item.name || '主要錢包') + ' · ' + address, item.id);
      option.translate = false;
      return option;
    }));
    $('account-select').value = accountPrefix.split('/')[2] || '';
    window.dispatchEvent(new CustomEvent('wallet-accounts', {detail: accounts}));
  }

  function renderAccountList() {
    const list = $('account-list');
    list.replaceChildren();
    for (const item of accounts.filter(item => $('show-archived-accounts').checked || !item.archived)) {
      const row = node('article', '', 'account-row');
      const info = node('div', '', 'account-info');
      const name = node('strong', item.name || '主要錢包');
      name.translate = false;
      info.append(name, node('p', item.address || '待完成設定', 'mono'));
      const current = item.id === (accountPrefix.split('/')[2] || '');
      info.append(node('span', item.archived ? '已封存' : current ? '目前使用' : '可切換', 'muted'));
      const actions = node('div', '', 'account-row-actions');
      if (!item.archived) {
        const open = node('a', item.address ? '使用錢包' : '繼續設定', 'secondary');
        open.href = accountURL(item.id);
        actions.append(open);
      }
      const rename = node('button', '更名', 'secondary');
      rename.type = 'button';
      rename.addEventListener('click', () => editAccount('rename', item));
      const archive = node('button', item.archived ? '還原' : '封存', 'secondary');
      archive.type = 'button';
      archive.addEventListener('click', () => editAccount(item.archived ? 'restore' : 'archive', item));
      if (!sharedDemo) actions.append(rename, archive);
      row.append(info, actions);
      list.append(row);
    }
  }

  function editAccount(mode, item = null) {
    accountEdit = {mode, item};
    $('account-list-view').hidden = true;
    $('account-editor').hidden = false;
    $('account-feedback').hidden = true;
    $('account-name').value = item?.name || '';
    $('account-name').readOnly = mode === 'archive' || mode === 'restore';
    const title = {create: '新增錢包', rename: '更改錢包名稱', archive: '封存錢包', restore: '還原錢包'}[mode];
    $('account-editor-title').textContent = title;
    $('save-account').textContent = mode === 'create' ? '下一步：設定錢包' : title;
    $('account-editor-description').textContent = mode === 'create'
      ? '先命名，下一步建立新錢包或匯入現有錢包。每個錢包分開保存金鑰與紀錄。'
      : mode === 'archive' ? '封存後會從切換清單隱藏，金鑰與紀錄仍會保留。這不會刪除或轉移鏈上資產。'
      : mode === 'restore' ? '還原後會重新出現在錢包切換清單。' : '名稱只用於本機辨識，不會改變錢包地址。';
    if (sharedDemo && mode === 'create') {
      $('account-password').value = '';
      $('account-password-confirm').value = '';
      $('save-account').textContent = '建立測試錢包';
      $('account-editor-description').textContent = '名稱及地址會公開。請自行保管密碼；建立後可在設定與備份下載加密備份。';
    }
    (mode === 'archive'  || mode === 'restore' ? $('save-account') : $('account-name')).focus();
  }

  $('manage-accounts').addEventListener('click', async () => {
    $('account-list-view').hidden = false;
    $('account-editor').hidden = true;
    $('account-feedback').hidden = true;
    renderAccountList();
    $('account-manager').showModal();
    try { accounts = await walletRequest('/api/wallet/accounts'); renderAccountList(); }
    catch (error) { showWalletError('account-feedback', error); }
  });
  $('close-account-manager').addEventListener('click', () => $('account-manager').close());
  $('show-archived-accounts').addEventListener('change', renderAccountList);
  $('add-account').addEventListener('click', () => editAccount('create'));
  $('cancel-account-edit').addEventListener('click', () => {
    $('account-editor').hidden = true;
    $('account-list-view').hidden = false;
    $('account-feedback').hidden = true;
    $('add-account').focus();
  });
  $('account-editor').addEventListener('submit', async event => {
    event.preventDefault();
    if ($('save-account').disabled) return;
    const {mode, item} = accountEdit;
    $('save-account').disabled = true;
    $('cancel-account-edit').disabled = true;
    $('close-account-manager').disabled = true;
    $('account-feedback').hidden = true;
    try {
      const name = $('account-name').value.trim();
      if (mode === 'create') {
        const body = {name};
        if (sharedDemo) {
          if ($('account-password').value !== $('account-password-confirm').value) throw new Error('兩次密碼不同');
          body.password = $('account-password').value;
        }
        const created = await walletRequest('/api/wallet/accounts', body);
        if (sharedDemo) { $('account-password').value = ''; $('account-password-confirm').value = ''; }
        location.href = accountURL(created.id);
        return;
      }
      const updated = await walletRequest('/api/wallet/accounts/update', {
        id: item.id, name, archived: mode === 'archive' ? true : mode === 'restore' ? false : item.archived,
      });
      accounts = accounts.map(account => account.id === updated.id ? updated : account);
      if (mode === 'archive' && item.id === (accountPrefix.split('/')[2] || '')) {
        location.href = accountURL(accounts.find(account => !account.archived).id);
        return;
      }
      renderAccountSelect();
      renderAccountList();
      $('account-editor').hidden = true;
      $('account-list-view').hidden = false;
      $('account-feedback').textContent = '已儲存';
      $('account-feedback').hidden = false;
      $('add-account').focus();
    } catch (error) { showWalletError('account-feedback', error); }
    finally {
      $('save-account').disabled = false;
      $('cancel-account-edit').disabled = false;
      $('close-account-manager').disabled = false;
    }
  });
  $('account-manager').addEventListener('cancel', event => {
    if ($('save-account').disabled) event.preventDefault();
  });
  let scanTimer;
  let scanRefreshing = false;
  let scanTokenOffset = 0;
  async function refreshScan() {
    if (!walletState?.exists || scanRefreshing) return;
    scanRefreshing = true;
    try {
      const state = await walletRequest('/api/wallet/scan');
      $('scan-progress').textContent = `${state.enabled ? '同步已啟用' : '同步已暫停'} · 起點 ${state.start} · 下一區塊 ${state.next} · finalized ${state.finalized}${state.error ? ' · ' + state.error : ''}`;
      const candidates = state.tokens || [];
      const offset = scanTokenOffset % (candidates.length || 1);
      const discovered = [...candidates.slice(offset), ...candidates.slice(0,offset)].filter(contract => !tokens.has(contract.toLowerCase())).slice(0,20);
      // Rotate failed candidates so they cannot keep later discoveries out of every batch.
      if (discovered.length) scanTokenOffset = (candidates.indexOf(discovered.at(-1)) + 1) % candidates.length;
      for (const contract of discovered) {
        try { const token = await walletRequest('/api/wallet/token',{contract}); tokens.set(contract.toLowerCase(),token); }
        catch { /* Nonstandard metadata does not turn a discovery candidate into a trusted asset. */ }
      }
      if (discovered.length) renderTokens();
    } catch (error) { $('scan-progress').textContent = errorMessage(error); }
    finally { scanRefreshing = false; clearTimeout(scanTimer); scanTimer = setTimeout(refreshScan,15000); }
  }
  $('auto-scan-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const input = $('auto-scan-start').value.trim();
      if (input && (!/^[0-9]+$/.test(input) || !Number.isSafeInteger(Number(input)))) throw new Error('起始區塊無效');
      await walletRequest('/api/wallet/scan',{enabled:true,...(input ? {start:Number(input)} : {})});
      await refreshScan();
    } catch (error) { showWalletError('activity-error',error); }
    finally { button.disabled = false; }
  });
  $('stop-scan').addEventListener('click', async () => {
    $('stop-scan').disabled = true;
    try { await walletRequest('/api/wallet/scan',{enabled:false}); await refreshScan(); }
    catch (error) { showWalletError('activity-error',error); }
    finally { $('stop-scan').disabled = false; }
  });
  loadWallet().then(refreshScan);
})();
