/**
 * hooks/useStore.js — 60-строчный реактивный стор без зависимостей.
 * Достаточно для Mini App: один глобальный срез + подписки по ключу.
 *
 *   const st = store({ user: null, objects: [] });
 *   st.set({ user: {...} });
 *   st.watch('user', (v) => ...);
 *   st.get('user');
 */

export function store(initial = {}) {
  let state = { ...initial };
  const watchers = new Map();

  return {
    get(key) {
      return key === undefined ? state : state[key];
    },
    set(patch) {
      const prev = state;
      state = { ...state, ...patch };
      for (const [key, fns] of watchers) {
        if (key in patch && prev[key] !== state[key]) {
          for (const fn of fns) {
            try { fn(state[key], prev[key]); } catch (e) { console.error('[store watcher]', e); }
          }
        }
      }
    },
    watch(key, fn) {
      if (!watchers.has(key)) watchers.set(key, new Set());
      watchers.get(key).add(fn);
      return () => watchers.get(key)?.delete(fn);
    },
  };
}

/**
 * guardLatest — обёртка для загрузки данных: гонка представлений
 * (быстрые тапы по табам) не даст старому ответу перезаписать новый экран.
 * Возвращает { run, stale }.
 */
export function guardLatest() {
  let token = 0;
  return {
    run(asyncFn) {
      const my = ++token;
      return asyncFn().then(
        (v) => ({ ok: my === token, value: v, stale: my !== token }),
        (e) => ({ ok: false, error: e, stale: my !== token, token: my }),
      );
    },
    get current() { return token; },
  };
}
