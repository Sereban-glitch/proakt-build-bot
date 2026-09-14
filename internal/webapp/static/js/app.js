/**
 * app.js — мини-роутер на hash: #/dashboard, #/objects/3, #/acts/10 …
 *
 * Представление — ES-модуль с экспортом view({ root, params, navigate })
 * → HTMLElement | { el, cleanup }. Роутер монтирует/размонтирует экран,
 * синхронизирует системную кнопку «Назад» Telegram с глубиной маршрута.
 */

const routes = [];

/** register('objects/:id', loader) — ленивый import() представления. */
export function register(pattern, loader) {
  routes.push({ pattern, keys: pattern.split('/').filter(Boolean), loader });
}

function match(hash) {
  const path = hash.replace(/^#\/?/, '').split('?')[0];
  const segs = path.split('/').filter(Boolean);
  for (const r of routes) {
    if (r.keys.length !== segs.length) continue;
    const params = {};
    let ok = true;
    for (let i = 0; i < r.keys.length; i++) {
      const k = r.keys[i];
      if (k.startsWith(':')) params[k.slice(1)] = decodeURIComponent(segs[i]);
      else if (k !== segs[i]) { ok = false; break; }
    }
    if (ok) return { route: r, params, segs };
  }
  return null;
}

export function navigate(hash) {
  if (location.hash === hash) return;
  location.hash = hash;
}

/**
 * run({ app, showBack }) — слушает hashchange и монтирует представления.
 * showBack(fnOrNull) — регистрация системной кнопки «Назад», возвращает
 * функцию отписки.
 */
export function run({ app, showBack }) {
  let screenCleanup = null;
  let backCleanup = null;

  const teardown = () => {
    if (screenCleanup) { try { screenCleanup(); } catch { /* ignore */ } screenCleanup = null; }
    if (backCleanup) { try { backCleanup(); } catch { /* ignore */ } backCleanup = null; }
  };

  const render = async () => {
    teardown();
    app.replaceChildren();

    const m = match(location.hash || '#/dashboard');
    if (!m) {
      navigate('#/dashboard');
      return;
    }

    // глубина > 1 — детальный экран: включаем системную «Назад»
    if (m.segs.length > 1) backCleanup = showBack(() => history.back());
    else backCleanup = showBack(null);

    try {
      const mod = await m.route.loader();
      const ctx = { root: app, params: m.params, navigate };
      const res = await mod.view(ctx);
      const el = res?.el ?? res;
      if (el) app.appendChild(el);
      if (res?.cleanup) screenCleanup = res.cleanup;
    } catch (err) {
      console.error('[router]', err);
      const box = document.createElement('div');
      box.className = 'empty';
      box.innerHTML = `<div class="empty-title">Не получилось открыть экран</div>`;
      app.appendChild(box);
    }
    window.scrollTo({ top: 0 });
  };

  window.addEventListener('hashchange', render);
  return render();
}
