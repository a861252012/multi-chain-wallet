'use strict';
window.claimTestTokens = async ({buttons, status, body, solana = false, tron = false, explorer, prefix, refresh}) => {
  if (buttons.some(button => button.disabled)) return;
  buttons.forEach(button => { button.disabled = true; });
  status.replaceChildren(document.createTextNode('正在申請測試幣…'));
  try {
    const settingsResponse = await fetch('/api/faucet', {signal: AbortSignal.timeout(10000)});
    const settings = await settingsResponse.json();
    if (!settingsResponse.ok) throw new Error(settings.error || '無法讀取領幣設定');
    const response = await fetch('/api/faucet' + (solana ? '/solana' : tron ? '/tron' : ''), {
      method: 'POST', headers: {'Content-Type':'application/json', 'X-Wallet-CSRF':settings.csrfToken},
      body: JSON.stringify(body), signal: AbortSignal.timeout(55000),
    });
    let result;
    try { result = await response.json(); }
    catch { throw new Error('無法讀取領幣回應，結果未知。請先核對餘額與交易紀錄，不要直接重複申請。'); }
    if (!response.ok) throw new Error(result.error || '領取失敗，請稍後再試');
    const message = document.createElement('span');
    const link = document.createElement('a');
    link.textContent = '查看鏈上交易 ↗'; link.target = '_blank'; link.rel = 'noopener noreferrer';
    link.href = tron ? `https://shasta.tronscan.org/#/transaction/${encodeURIComponent(result.signature)}` : solana ? `https://explorer.solana.com/tx/${encodeURIComponent(result.signature)}?cluster=devnet` : `${explorer}/tx/${encodeURIComponent(result.hash)}`;
    message.textContent = result.reused ? '本小時已有同額發放紀錄，正在核對原交易。' : '申請已送出，等待鏈上結果。';
    status.replaceChildren(message, document.createTextNode(' '), link);
    await refresh();
    if (solana) {
      message.textContent = '空投已提交，請核對最新餘額與交易連結；尚未確認到帳。';
      return;
    }
    for (let attempt = 0; attempt < 12; attempt += 1) {
      const receiptResponse = await fetch(tron ? `/api/faucet/tron/${encodeURIComponent(result.signature)}` : `${prefix}/api/transactions/${encodeURIComponent(result.hash)}`, {signal:AbortSignal.timeout(10000)});
      if (receiptResponse.ok) {
        const receipt = await receiptResponse.json();
        if (receipt.state === 'succeeded' || receipt.state === 'finalized') { await refresh(); message.textContent = result.reused ? '本小時已領取，原交易已成功；未重複發放，已重新查詢餘額。' : '測試幣轉帳已在鏈上執行成功，已重新查詢餘額。'; return; }
        if (receipt.state === 'reverted' || receipt.state === 'execution_failed' || receipt.state === 'expired_unconfirmed') { message.textContent = '這筆發放交易執行失敗，未完成領取。'; return; }
      }
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
    message.textContent = '交易結果仍待確認；請保留此連結，稍後更新餘額。';
  } catch (error) {
    const message = error.name === 'TimeoutError' || error instanceof TypeError
      ? (solana ? '連線中斷或逾時，空投結果未知。請先更新餘額，不要連續申請。' : '連線中斷或逾時，領取結果未知。請先更新餘額；再次申請會優先查找原發放交易。')
      : error.message;
    // Keep an already returned transaction link available even when the receipt RPC fails.
    const link = status.querySelector('a');
    status.replaceChildren(document.createTextNode(message));
    if (link) status.append(document.createTextNode(' '), link);
  } finally {
    buttons.forEach(button => { button.disabled = false; });
  }
};
