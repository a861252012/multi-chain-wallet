'use strict';
(() => {
  // Version 1 storage namespace bytes are retained only to migrate existing browser data.
  const previousNamespace = String.fromCharCode(102, 108, 111, 119, 108, 101, 100, 103, 101, 114) + ':';
  for (const storageName of ['localStorage', 'sessionStorage']) {
    try {
      const storage = window[storageName];
      const keys = Array.from({length: storage.length}, (_, index) => storage.key(index));
      for (const key of keys) {
        if (!key?.startsWith(previousNamespace)) continue;
        const next = 'multi-chain-wallet:' + key.slice(previousNamespace.length);
        if (storage.getItem(next) === null) storage.setItem(next, storage.getItem(key));
        storage.removeItem(key);
      }
    } catch { /* Keep the original entry when browser storage cannot be written. */ }
  }
  let locale = 'zh-TW', theme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  try {
    const savedLocale = localStorage.getItem('multi-chain-wallet:locale');
    const savedTheme = localStorage.getItem('multi-chain-wallet:theme');
    if (['zh-TW', 'en', 'zh-CN'].includes(savedLocale)) locale = savedLocale;
    if (['light', 'dark'].includes(savedTheme)) theme = savedTheme;
  } catch { /* Preferences still work when browser storage is unavailable. */ }
  document.documentElement.lang = locale;
  document.documentElement.dataset.theme = theme;
})();
