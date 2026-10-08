'use strict';
(() => {
  const $ = id => document.getElementById(id);
  let state,
    quote,
    sending = false,
    tokenRequestSequence = 0;
  const labels = {
    expired_unconfirmed: '已過期，固化鏈未查到收據',
    broadcast_unknown: '廣播結果待確認',
    submitted: '已廣播',
    processed: '已處理，等待確認',
    confirmed: '已確認，等待 finalized',
    finalized: '已 finalized',
    execution_failed: '鏈上執行失敗'
  };
  const text = (tag, value) => {
    const el = document.createElement(tag);
    el.textContent = value;
    return el;
  };
  const tokenSymbol = value => {
    const el = text('span', value);
    el.translate = false;
    return el;
  };
  const tokenAmount = (tag, amount, symbol) => {
    const el = text(tag, amount + ' ');
    el.append(tokenSymbol(symbol));
    return el;
  };
  async function api(path, body) {
    const response = await fetch('/tron/api/' + path, {
      signal: AbortSignal.timeout(45000),
      ...(body === undefined
        ? {}
        : {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'X-Wallet-CSRF': state.csrfToken },
            body: JSON.stringify(body)
          })
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || '操作失敗');
    return data;
  }
  function error(err) {
    $('tron-error').textContent =
      err.name === 'TimeoutError' ? '查詢逾時，結果未知；請先更新原交易紀錄。' : err.message;
  }
  async function refresh() {
    if (!state?.exists || $('tron-refresh').disabled) return;
    $('tron-refresh').disabled = true;
    $('tron-error').textContent = '';
    const results = await Promise.allSettled([
      api('balance?address=' + encodeURIComponent(state.address)),
      api('history'),
      api('status')
    ]);
    if (results[2].status === 'fulfilled') state.csrfToken = results[2].value.csrfToken;
    if (results[0].status === 'fulfilled') {
      $('tron-balance').textContent = results[0].value.trx + ' TRX';
      $('tron-balance-time').textContent =
        `可用 Bandwidth ${results[0].value.bandwidth} · Energy ${results[0].value.energy} · ${results[0].value.active ? '帳戶已啟用' : '尚未啟用，請先領取 TRX'}`;
    } else {
      $('tron-balance').textContent = '—';
      $('tron-balance-time').textContent = '無法取得最新餘額';
      error(results[0].reason);
    }
    if (results[1].status === 'fulfilled') {
      $('tron-history').replaceChildren();
      for (const tx of results[1].value.slice().reverse()) {
        const row = text('article', '');
        row.className = 'history-row';
        const link = text('a', tx.signature);
        link.href = 'https://shasta.tronscan.org/#/transaction/' + encodeURIComponent(tx.signature);
        link.target = '_blank';
        link.rel = 'noopener noreferrer';
        if (tx.feeTrx) row.append(text('p', '實際費用 ' + tx.feeTrx + ' TRX'));
        if (tx.result) row.append(text('p', '執行結果 ' + tx.result));
        row.append(
          tokenAmount('strong', tx.amount, tx.symbol),
          text('p', tx.to),
          text('p', (labels[tx.state] || tx.state) + (tx.finalized ? ' · 終局確認' : '')),
          link
        );
        if (!tx.finalized && tx.state !== 'expired_unconfirmed') {
          const retry = text('button', '重新廣播原交易');
          retry.className = 'secondary';
          retry.addEventListener('click', async () => {
            retry.disabled = true;
            try {
              await api('retry', { signature: tx.signature });
              await refresh();
            } catch (err) {
              error(err);
            } finally {
              retry.disabled = false;
            }
          });
          row.append(retry);
        }
        $('tron-history').append(row);
      }
    } else {
      error(results[1].reason);
      $('tron-history').prepend(text('p', '未能更新收據；以下若有紀錄，僅為上次查核結果。'));
    }
    $('tron-refresh').disabled = false;
  }
  async function load() {
    try {
      state = await api('status');
      $('tron-setup').hidden = state.exists;
      $('tron-dashboard').hidden = !state.exists;
      if (state.exists) {
        $('tron-address').textContent = state.address;
        await refresh();
      }
    } catch (err) {
      error(err);
    }
  }
  $('tron-create').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    if ($('tron-password').value !== $('tron-password-confirm').value) {
      error(new Error('兩次密碼不同'));
      return;
    }
    button.disabled = true;
    try {
      const result = await api('create', {
        mnemonic: $('tron-mnemonic').value.trim(),
        password: $('tron-password').value
      });
      $('tron-create').reset();
      if (result.mnemonic) {
        $('tron-setup').hidden = true;
        $('tron-backup-words').hidden = false;
        $('tron-words').replaceChildren(
          ...result.mnemonic.split(' ').map(word => text('li', word))
        );
      } else await load();
    } catch (err) {
      error(err);
    } finally {
      $('tron-password').value = '';
      $('tron-password-confirm').value = '';
      $('tron-mnemonic').value = '';
      button.disabled = false;
    }
  });
  $('tron-backed-up').addEventListener('change', () => {
    $('tron-finish').disabled = !$('tron-backed-up').checked;
  });
  $('tron-finish').addEventListener('click', () => {
    if (!$('tron-backed-up').checked) return;
    $('tron-words').replaceChildren();
    $('tron-backup-words').hidden = true;
    load();
  });
  window.addEventListener('beforeunload', event => {
    if (!$('tron-backup-words').hidden) {
      event.preventDefault();
      event.returnValue = '';
    }
  });
  $('tron-claim').addEventListener('click', () =>
    window.claimTestTokens({
      buttons: [$('tron-claim')],
      status: $('tron-funding-status'),
      body: {},
      tron: true,
      refresh
    })
  );
  $('tron-refresh').addEventListener('click', refresh);
  function clearTokenInfo() {
    tokenRequestSequence++;
    $('tron-token-info').textContent = '';
    $('tron-token-query').disabled = false;
  }
  $('tron-self').addEventListener('click', () => {
    $('tron-contract').value = '';
    $('tron-amount-label').textContent = '數量';
    clearTokenInfo();
    $('tron-to').value = '';
    $('tron-amount').value = '0.000001';
    $('tron-to').focus();
  });
  $('tron-copy').addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(state.address);
      $('tron-copy').textContent = '已複製';
    } catch {
      error(new Error('請手動複製完整地址'));
    }
  });
  $('tron-contract').addEventListener('input', () => {
    clearTokenInfo();
    $('tron-amount-label').textContent = $('tron-contract').value.trim()
      ? '最小單位數量（整數）'
      : '數量';
  });
  $('tron-transfer').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const contract = $('tron-contract').value.trim(),
        input = $('tron-amount').value.trim();
      quote = await api('quote', {
        to: $('tron-to').value.trim(),
        amount: contract ? '' : input,
        amountRaw: contract ? input : '',
        contract
      });
      $('tron-quote').replaceChildren(
        ...[
          ['網路', 'TRON Shasta'],
          ['收款人', quote.to],
          ['數量', quote.amount],
          ['預估費用', quote.feeTrx + ' TRX'],
          ['有效至', new Date(quote.expiresAt).toLocaleString(document.documentElement.lang)],
          ['Energy 用量估算', quote.energy],
          ['Bandwidth 預留', quote.bandwidth],
          ['Energy fee_limit', quote.feeLimitTrx + ' TRX'],
          ['代幣合約', quote.contract || '原生 TRX'],
          ['最小單位', quote.amountRaw || '—'],
          ['代幣精度', quote.decimals]
        ].map(([k, v]) => {
          const row = text('p', k + '：' + v + (k === '數量' ? ' ' : ''));
          if (k === '數量') row.append(tokenSymbol(quote.symbol));
          return row;
        })
      );
      $('tron-sign-error').textContent = '';
      $('tron-confirm').showModal();
    } catch (err) {
      error(err);
    } finally {
      button.disabled = false;
    }
  });
  $('tron-cancel').addEventListener('click', () => {
    if (!sending) $('tron-confirm').close();
  });
  $('tron-confirm').addEventListener('cancel', event => {
    if (sending) event.preventDefault();
  });
  $('tron-confirm').addEventListener('close', () => {
    $('tron-sign-password').value = '';
    quote = undefined;
  });
  $('tron-sign').addEventListener('submit', async event => {
    event.preventDefault();
    if (sending || !quote) return;
    sending = true;
    const button = event.currentTarget.querySelector('button');
    button.disabled = true;
    $('tron-cancel').disabled = true;
    try {
      const result = await api('send', { id: quote.id, password: $('tron-sign-password').value });
      $('tron-confirm').close();
      const link = text('a', result.signature);
      link.href =
        'https://shasta.tronscan.org/#/transaction/' + encodeURIComponent(result.signature);
      link.target = '_blank';
      link.rel = 'noopener noreferrer';
      link.translate = false;
      const row = text('div', '');
      row.className = 'history-row';
      row.append(link);
      $('tron-send-result').replaceChildren(
        text('p', '送出時狀態：' + (labels[result.state] || result.state)),
        row
      );
      $('tron-send-result').hidden = false;
      await refresh();
    } catch (err) {
      $('tron-sign-error').textContent = err.message;
    } finally {
      $('tron-sign-password').value = '';
      sending = false;
      button.disabled = false;
      $('tron-cancel').disabled = false;
    }
  });
  $('tron-export').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const data = await api('backup', { password: $('tron-backup-password').value });
      const url = URL.createObjectURL(
        new Blob([JSON.stringify(data)], { type: 'application/json' })
      );
      const link = text('a', '');
      link.href = url;
      link.download = 'multi-chain-wallet-tron-shasta-' + state.address + '.json';
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (err) {
      error(err);
    } finally {
      $('tron-backup-password').value = '';
      button.disabled = false;
    }
  });
  $('tron-restore').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const file = $('tron-restore-file').files[0];
      if (!file || file.size > 8192) throw new Error('請選擇 8 KB 以內的備份檔');
      if ($('tron-restore-new').value !== $('tron-restore-confirm').value)
        throw new Error('兩次新密碼不同');
      await api('restore', {
        backup: JSON.parse(await file.text()),
        password: $('tron-restore-password').value,
        newPassword: $('tron-restore-new').value
      });
      await load();
    } catch (err) {
      error(err);
    } finally {
      $('tron-restore').reset();
      button.disabled = false;
    }
  });
  $('tron-password-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      if ($('tron-new-password').value !== $('tron-new-confirm').value)
        throw new Error('兩次新密碼不同');
      await api('password', {
        password: $('tron-old-password').value,
        newPassword: $('tron-new-password').value
      });
      $('tron-error').textContent = '密碼已更新，請重新下載備份；舊備份仍使用舊密碼。';
    } catch (err) {
      error(err);
    } finally {
      $('tron-password-form').reset();
      button.disabled = false;
    }
  });
  $('tron-token-query').addEventListener('click', async () => {
    const button = $('tron-token-query');
    if (button.disabled) return;
    const requestSequence = ++tokenRequestSequence;
    const contract = $('tron-contract').value.trim();
    const isCurrent = () =>
      requestSequence === tokenRequestSequence && contract === $('tron-contract').value.trim();
    button.disabled = true;
    $('tron-token-info').textContent = '';
    try {
      const token = await api('token?' + new URLSearchParams({ address: state.address, contract }));
      if (isCurrent())
        $('tron-token-info').replaceChildren(
          document.createTextNode(`${token.balance} `),
          tokenSymbol(token.symbol),
          document.createTextNode(` · ${token.decimals} 位小數 · ${token.contract}`)
        );
    } catch (err) {
      if (isCurrent()) $('tron-token-info').textContent = err.message;
    } finally {
      if (isCurrent()) button.disabled = false;
    }
  });
  setInterval(() => {
    if (!document.hidden && !sending) refresh();
  }, 15000);
  load();
})();
