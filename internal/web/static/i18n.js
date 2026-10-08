'use strict';
(() => {
  const catalog = window.FlowMessages;
  const originals = new WeakMap();
  const attributes = new WeakMap();
  const excluded = 'script,style,textarea,input,code,pre,[translate="no"],.mnemonic-grid';
  const escape = value => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const patterns = Object.entries(catalog).filter(([source]) => source.includes('${')).map(([source, values]) => {
    const slots = [...source.matchAll(/\$\{[^}]+\}/g)].map(match => match[0]);
    const parts = source.split(/\$\{[^}]+\}/g);
    // A pattern needs literal UI wording; arbitrary user data is never a message key.
    if (!parts.some(part => /[\u3400-\u9fff]/.test(part))) return null;
    return {regex:new RegExp('^' + parts.map(escape).join('(.+?)') + '$'), slots, values};
  }).filter(Boolean);
  let locale = document.documentElement.lang;
  function translate(value, depth = 0) {
    if (locale === 'zh-TW' || typeof value !== 'string' || depth > 3) return value;
    const trimmed = value.trim();
    const pair = Object.hasOwn(catalog, trimmed) ? catalog[trimmed] : null;
    if (pair && !trimmed.includes('${')) return value.replace(trimmed, pair[locale === 'en' ? 0 : 1]);
    for (const pattern of patterns) {
      const match = trimmed.match(pattern.regex);
      if (!match) continue;
      let result = pattern.values[locale === 'en' ? 0 : 1];
      // Replacements use a callback so dollar signs in data stay literal.
      result = result.replace(/\$\{[^}]+\}/g, slot => {
        const index = pattern.slots.indexOf(slot);
        return index >= 0 ? translate(match[index + 1], depth + 1) : slot;
      });
      return value.replace(trimmed, () => result);
    }
    // Composed status rows use these separators; addresses, amounts and symbols are unchanged.
    if (/[·：]/.test(value)) return value.split(/([·：])/).map(part => /[·：]/.test(part) ? (locale === 'en' && part === '：' ? ': ' : part) : translate(part, depth + 1)).join('');
    return value;
  }
  function localizeText(node) {
    if (!node.parentElement || node.parentElement.closest(excluded)) return;
    const prior = originals.get(node);
    const source = prior && node.nodeValue === prior.rendered ? prior.source : node.nodeValue;
    const rendered = translate(source);
    if (node.nodeValue !== rendered) node.nodeValue = rendered;
    originals.set(node, {source, rendered});
  }
  function walk(root) {
    if (root.nodeType === Node.TEXT_NODE) { localizeText(root); return; }
    if (root.nodeType !== Node.ELEMENT_NODE && root.nodeType !== Node.DOCUMENT_NODE) return;
    if (root.nodeType === Node.ELEMENT_NODE && root.closest(excluded)) return;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    for (let node = walker.nextNode(); node; node = walker.nextNode()) localizeText(node);
    const elements = root.querySelectorAll('[placeholder],[aria-label],[alt],[title]');
    const candidates = root.nodeType === Node.ELEMENT_NODE ? [root, ...elements] : elements;
    for (const el of candidates) {
      if (el.closest('[translate="no"]')) continue;
      let saved = attributes.get(el);
      if (!saved) { saved = {}; attributes.set(el, saved); }
      for (const key of ['placeholder','aria-label','alt','title']) {
        if (!el.hasAttribute(key)) continue;
        const current = el.getAttribute(key);
        const source = saved[key] && saved[key].rendered === current ? saved[key].source : current;
        const rendered = translate(source);
        if (current !== rendered) el.setAttribute(key, rendered);
        saved[key] = {source, rendered};
      }
    }
  }
  const observer = new MutationObserver(records => {
    observer.disconnect();
    const roots = new Set();
    for (const record of records) {
      if (record.type === 'childList') record.addedNodes.forEach(node => roots.add(node));
      else roots.add(record.target);
    }
    roots.forEach(walk);
    observe();
  });
  function observe() { observer.observe(document.documentElement, {subtree:true, childList:true, characterData:true, attributes:true, attributeFilter:['placeholder','aria-label','alt','title']}); }
  function refresh() { observer.disconnect(); walk(document.documentElement); observe(); }
  window.FlowI18n = {
    get locale() { return locale; },
    t: translate,
    refresh,
    setLocale(next) {
      if (!['zh-TW','en','zh-CN'].includes(next)) return;
      locale = next;
      document.documentElement.lang = locale;
      try { localStorage.setItem('multi-chain-wallet:locale',locale); } catch { /* Session-only preference. */ }
      refresh();
      window.dispatchEvent(new Event('localechange'));
    },
  };
  refresh();
})();
