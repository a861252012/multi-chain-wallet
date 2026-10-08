'use strict';
(() => {
  const $ = id => document.getElementById(id);
  let state,
    quote,
    sending = false;
  const labels = {
    expired_unconfirmed: '已過期，查無鏈上紀錄',
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
  async function api(path, body) {
    const response = await fetch('/solana/api/' + path, {
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
    $('sol-error').textContent =
      err.name === 'TimeoutError' ? '查詢逾時，結果未知；請先更新原交易紀錄。' : err.message;
  }
  async function refresh() {
    if (!state?.exists || $('sol-refresh').disabled) return;
    $('sol-refresh').disabled = true;
    $('sol-error').textContent = '';
    const results = await Promise.allSettled([
      api('balance?address=' + encodeURIComponent(state.address)),
      api('history'),
      api('status')
    ]);
    if (results[2].status === 'fulfilled') state.csrfToken = results[2].value.csrfToken;
    if (results[0].status === 'fulfilled') {
      $('sol-balance').textContent = results[0].value.sol + ' SOL';
      $('sol-balance-time').textContent = '查詢 Slot ' + results[0].value.slot;
    } else {
      $('sol-balance').textContent = '—';
      $('sol-balance-time').textContent = '無法取得最新餘額';
      error(results[0].reason);
    }
    if (results[1].status === 'fulfilled') {
      $('sol-history').replaceChildren();
      for (const tx of results[1].value.slice().reverse()) {
        const row = text('article', '');
        row.className = 'history-row';
        const link = text('a', tx.signature);
        link.href =
          'https://explorer.solana.com/tx/' + encodeURIComponent(tx.signature) + '?cluster=devnet';
        link.target = '_blank';
        link.rel = 'noopener noreferrer';
        row.append(
          text('strong', tx.amount + ' SOL'),
          text('p', tx.to),
          text('p', (labels[tx.state] || tx.state) + (tx.finalized ? ' · 終局確認' : '')),
          link
        );
        if (
          !tx.finalized &&
          tx.state !== 'expired_unconfirmed' &&
          tx.state !== 'execution_failed'
        ) {
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
        $('sol-history').append(row);
      }
    } else {
      error(results[1].reason);
      $('sol-history').prepend(text('p', '未能更新收據；以下若有紀錄，僅為上次查核結果。'));
    }
    $('sol-refresh').disabled = false;
  }
  async function load() {
    try {
      state = await api('status');
      $('sol-setup').hidden = state.exists;
      $('sol-dashboard').hidden = !state.exists;
      if (state.exists) {
        $('sol-address').textContent = state.address;
        await refresh();
      }
    } catch (err) {
      error(err);
    }
  }
  $('sol-create').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    if ($('sol-password').value !== $('sol-password-confirm').value) {
      error(new Error('兩次密碼不同'));
      return;
    }
    button.disabled = true;
    try {
      const result = await api('create', {
        mnemonic: $('sol-mnemonic').value.trim(),
        password: $('sol-password').value
      });
      $('sol-create').reset();
      if (result.mnemonic) {
        $('sol-setup').hidden = true;
        $('sol-backup-words').hidden = false;
        $('sol-words').replaceChildren(...result.mnemonic.split(' ').map(word => text('li', word)));
      } else await load();
    } catch (err) {
      error(err);
    } finally {
      $('sol-password').value = '';
      $('sol-password-confirm').value = '';
      $('sol-mnemonic').value = '';
      button.disabled = false;
    }
  });
  $('sol-backed-up').addEventListener('change', () => {
    $('sol-finish').disabled = !$('sol-backed-up').checked;
  });
  $('sol-finish').addEventListener('click', () => {
    if (!$('sol-backed-up').checked) return;
    $('sol-words').replaceChildren();
    $('sol-backup-words').hidden = true;
    load();
  });
  window.addEventListener('beforeunload', event => {
    if (!$('sol-backup-words').hidden) {
      event.preventDefault();
      event.returnValue = '';
    }
  });
  $('sol-airdrop').addEventListener('click', () =>
    window.claimTestTokens({
      buttons: [$('sol-airdrop')],
      status: $('sol-funding-status'),
      body: {},
      solana: true,
      refresh
    })
  );
  $('sol-refresh').addEventListener('click', refresh);
  $('sol-self').addEventListener('click', () => {
    $('sol-to').value = state.address;
    $('sol-amount').value = '0.000001';
    $('sol-to').focus();
  });
  $('sol-copy').addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(state.address);
      $('sol-copy').textContent = '已複製';
    } catch {
      error(new Error('請手動複製完整地址'));
    }
  });
  $('sol-transfer').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      quote = await api('quote', {
        to: $('sol-to').value.trim(),
        amount: $('sol-amount').value.trim()
      });
      $('sol-quote').replaceChildren(
        ...[
          ['網路', 'Solana Devnet'],
          ['收款人', quote.to],
          ['數量', quote.amount + ' SOL'],
          ['預估費用', quote.feeSol + ' SOL'],
          ['有效至', new Date(quote.expiresAt).toLocaleString(document.documentElement.lang)],
          ['最晚有效區塊高度', quote.lastValidBlockHeight]
        ].map(([k, v]) => text('p', k + '：' + v))
      );
      $('sol-sign-error').textContent = '';
      $('sol-confirm').showModal();
    } catch (err) {
      error(err);
    } finally {
      button.disabled = false;
    }
  });
  $('sol-cancel').addEventListener('click', () => {
    if (!sending) $('sol-confirm').close();
  });
  $('sol-confirm').addEventListener('cancel', event => {
    if (sending) event.preventDefault();
  });
  $('sol-confirm').addEventListener('close', () => {
    $('sol-sign-password').value = '';
    quote = undefined;
  });
  $('sol-sign').addEventListener('submit', async event => {
    event.preventDefault();
    if (sending || !quote) return;
    sending = true;
    const button = event.currentTarget.querySelector('button');
    button.disabled = true;
    $('sol-cancel').disabled = true;
    try {
      const result = await api('send', { id: quote.id, password: $('sol-sign-password').value });
      $('sol-confirm').close();
      const link = text('a', result.signature);
      link.href =
        'https://explorer.solana.com/tx/' +
        encodeURIComponent(result.signature) +
        '?cluster=devnet';
      link.target = '_blank';
      link.rel = 'noopener noreferrer';
      link.translate = false;
      const row = text('div', '');
      row.className = 'history-row';
      row.append(link);
      $('sol-send-result').replaceChildren(
        text('p', '送出時狀態：' + (labels[result.state] || result.state)),
        row
      );
      $('sol-send-result').hidden = false;
      await refresh();
    } catch (err) {
      $('sol-sign-error').textContent = err.message;
    } finally {
      $('sol-sign-password').value = '';
      sending = false;
      button.disabled = false;
      $('sol-cancel').disabled = false;
    }
  });
  $('sol-export').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const data = await api('backup', { password: $('sol-backup-password').value });
      const url = URL.createObjectURL(
        new Blob([JSON.stringify(data)], { type: 'application/json' })
      );
      const link = text('a', '');
      link.href = url;
      link.download = 'multi-chain-wallet-solana-devnet-' + state.address + '.json';
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (err) {
      error(err);
    } finally {
      $('sol-backup-password').value = '';
      button.disabled = false;
    }
  });
  $('sol-restore').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      const file = $('sol-restore-file').files[0];
      if (!file || file.size > 8192) throw new Error('請選擇 8 KB 以內的備份檔');
      if ($('sol-restore-new').value !== $('sol-restore-confirm').value)
        throw new Error('兩次新密碼不同');
      await api('restore', {
        backup: JSON.parse(await file.text()),
        password: $('sol-restore-password').value,
        newPassword: $('sol-restore-new').value
      });
      await load();
    } catch (err) {
      error(err);
    } finally {
      $('sol-restore').reset();
      button.disabled = false;
    }
  });
  $('sol-password-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('button');
    if (button.disabled) return;
    button.disabled = true;
    try {
      if ($('sol-new-password').value !== $('sol-new-confirm').value)
        throw new Error('兩次新密碼不同');
      await api('password', {
        password: $('sol-old-password').value,
        newPassword: $('sol-new-password').value
      });
      $('sol-error').textContent = '密碼已更新，請重新下載備份；舊備份仍使用舊密碼。';
    } catch (err) {
      error(err);
    } finally {
      $('sol-password-form').reset();
      button.disabled = false;
    }
  });
  setInterval(() => {
    if (!document.hidden && !sending) refresh();
  }, 15000);
  load();
})();
