/**
 * services/tg.js — единственное место, которое знает о Telegram.WebApp SDK.
 *
 * Остальные модули импортируют только этот фасад: смена SDK или демо-режим
 * не затронет ни один компонент. Все методы безопасны вне Telegram
 * (обычный браузер → demo-режим, хаптика/кнопки превращаются в no-op).
 */

const wa = typeof window !== 'undefined' ? window.Telegram?.WebApp : null;

/** Живём ли мы внутри Telegram с реальной подписью initData. */
export const native = Boolean(wa && wa.initData && wa.initDataUnsafe?.user?.id);

/** Открыто в браузере (не в Telegram) — включаем demo-режим с мок-данными. */
export const isDemo = !native;

/** Сырой initData для подписи API-запросов (пусто вне Telegram). */
export const initData = native ? String(wa.initData) : '';

/** Пользователь из initDataUnsafe (может быть null вне Telegram). */
export const user = native
  ? {
      id: wa.initDataUnsafe.user.id,
      name: [wa.initDataUnsafe.user.first_name, wa.initDataUnsafe.user.last_name]
        .filter(Boolean).join(' ') || wa.initDataUnsafe.user.username || '',
      username: wa.initDataUnsafe.user.username || '',
    }
  : { id: 0, name: 'Демо-мастер', username: 'demo' };

let ready = false;

/** ready() + expand() — стандартный старт TMA: валидная высота, полный экран. */
export function init() {
  if (!wa || ready) return;
  try {
    wa.ready();
    wa.expand();
    // цвет служебных интерфейсов Telegram — под наш фон
    if (wa.setHeaderColor) wa.setHeaderColor('bg_color');
    if (wa.setBackgroundColor) wa.setBackgroundColor('bg_color');
  } catch { /* старые клиенты — молча */ }
  ready = true;
}

/** Актуальная стабильная высота вьюпорта (см. viewportStableHeight). */
export function stableHeight() {
  if (wa && wa.viewportStableHeight) return wa.viewportStableHeight;
  return window.innerHeight;
}

// --- тема ------------------------------------------------------------

const FALLBACK_DARK = {
  bg_color: '#17212b', text_color: '#f2f5f7', hint_color: '#7c8894',
  link_color: '#58b0e3', button_color: '#2ecc71', button_text_color: '#06281a',
  secondary_bg_color: '#0e1621',
};

/** Текущие themeParams с фолбэком (вне Telegram params пустые). */
export function themeParams() {
  if (native && wa.themeParams && Object.keys(wa.themeParams).length) {
    return { ...FALLBACK_DARK, ...wa.themeParams };
  }
  return FALLBACK_DARK;
}

/** Текущая схема: 'dark' | 'light'. */
export function colorScheme() {
  if (native && wa.colorScheme) return wa.colorScheme;
  return 'dark';
}

/** Подписка на смену темы/схемы. Возвращает функцию отписки. */
export function onThemeChanged(fn) {
  if (!native || !wa.onEvent) return () => {};
  const handler = () => fn(themeParams(), colorScheme());
  wa.onEvent('themeChanged', handler);
  return () => wa.offEvent && wa.offEvent('themeChanged', handler);
}

// --- хаптика ----------------------------------------------------------

function haptic(style, params) {
  if (!native || !wa.HapticFeedback) return;
  try {
    wa.HapticFeedback.impactOccurred(style, params);
  } catch { /* ignore */ }
}

/** Пресеты хаптики для всего приложения. */
export const haptics = {
  tap: () => haptic('light'),
  medium: () => haptic('medium'),
  rigid: () => haptic('rigid'),
  select: () => {
    if (native && wa.HapticFeedback) {
      try { wa.HapticFeedback.selectionChanged(); } catch { /* ignore */ }
    }
  },
  success: () => notify('success'),
  warn: () => notify('warning'),
  error: () => notify('error'),
};

function notify(type) {
  if (!native || !wa.HapticFeedback) return;
  try { wa.HapticFeedback.notificationOccurred(type); } catch { /* ignore */ }
}

// --- BackButton / MainButton ------------------------------------------

const backHandlers = new Set();

if (native && wa.BackButton) {
  wa.onEvent('backButtonClicked', () => {
    for (const fn of backHandlers) fn();
  });
}

/** Показать/скрыть системную кнопку «Назад». Возвращает функцию снятия колбэка. */
export function showBackButton(onClick) {
  if (!native || !wa.BackButton) return () => {};
  backHandlers.add(onClick);
  wa.BackButton.show();
  return () => {
    backHandlers.delete(onClick);
    if (backHandlers.size === 0) wa.BackButton.hide();
  };
}

/** Скрыть системную кнопку «Назад» и сбросить обработчики (смена маршрута). */
export function hideBackButton() {
  backHandlers.clear();
  if (native && wa.BackButton) {
    try { wa.BackButton.hide(); } catch { /* ignore */ }
  }
}

/** MainButton — для подтверждений в нижних листах. Возвращает контроллер. */
export function mainButton() {
  if (!native || !wa.MainButton) {
    return { show() {}, hide() {}, setText() {}, onClick() {}, offClick() {}, enable() {}, disable() {}, showProgress() {}, hideProgress() {} };
  }
  const mb = wa.MainButton;
  return {
    show(opts = {}) {
      if (opts.text) mb.setText(opts.text);
      if (opts.color) mb.setColor(opts.color);
      mb.show();
    },
    hide() { mb.hide(); },
    setText(t) { mb.setText(t); },
    setColor(c) { try { mb.setColor(c); } catch { /* ignore */ } },
    onClick(fn) { mb.onClick(fn); },
    offClick(fn) { mb.offClick(fn); },
    enable() { mb.enable(); },
    disable() { mb.disable(); },
    showProgress(leaveActive) { mb.showProgress(leaveActive); },
    hideProgress() { mb.hideProgress(); },
  };
}

// --- попапы / ссылки ----------------------------------------------------

export function showAlert(message) {
  if (native && wa.showAlert) wa.showAlert(message);
  else alert(message);
}

export function showConfirm(message) {
  return new Promise((resolve) => {
    if (native && wa.showConfirm) wa.showConfirm(message, resolve);
    else resolve(confirm(message));
  });
}

export function openLink(url, options) {
  if (native && wa.openLink) wa.openLink(url, options);
  else window.open(url, '_blank', 'noopener');
}

/** Зазор под шапку Telegram (contentSafeArea) на новых клиентах. */
export function topInsetPx() {
  try {
    if (wa?.contentSafeAreaInset?.top) return wa.contentSafeAreaInset.top;
  } catch { /* ignore */ }
  return 0;
}

// --- TMA 2026: защита контента и шеринг ------------------------------------

/**
 * disableVerticalSwipes() — Bot API 7.7+. На длинных экранах (смета на
 * 40 позиций) свайп вниз закрывает Mini App прямо во время скролла —
 * отключаем. Вне Telegram / на старых клиентах — no-op.
 */
export function disableVerticalSwipes() {
  try { wa?.disableVerticalSwipes?.(); } catch { /* ignore */ }
}

/**
 * closingConfirmation(on) — «Точно выйти?» при попытке закрыть Mini App,
 * пока в конструкторе сметы есть несохранённые правки (v0.5).
 */
export function closingConfirmation(on) {
  try {
    if (on) wa?.enableClosingConfirmation?.();
    else wa?.disableClosingConfirmation?.();
  } catch { /* ignore */ }
}

/**
 * shareURL(url, text) — системный шеринг: в Telegram открывает пересылку
 * в чаты, вне — navigator.share с фолбэком на копирование.
 * Возвращает 'shared' | 'copied' | 'cancelled'.
 */
export async function shareURL(url, text = '') {
  const tgShare = `https://t.me/share/url?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`;
  if (native && wa.openTelegramLink) {
    try { wa.openTelegramLink(tgShare); return 'shared'; } catch { /* фолбэк ниже */ }
  }
  if (navigator.share) {
    try { await navigator.share({ title: 'Смета', text, url }); return 'shared'; } catch { /* отмена */ }
  }
  try {
    await navigator.clipboard.writeText(url);
    return 'copied';
  } catch {
    return 'cancelled';
  }
}

/**
 * hapticEnabled() — для настроек: у части пользователей хаптика отключена
 * в Telegram; ложные вызовы безвредны, но для тонких анимаций полезно знать.
 */
export const hapticsAvailable = Boolean(native && wa?.HapticFeedback);

/**
 * backGestureFallback — hash-навигация в браузере не даёт системную
 * «Назад» Mini App; на детальных экранах TMA 2026 можно подписаться на
 * viewportChanged, чтобы контент не прыгал при сворачивании клавиатуры.
 * Возвращает функцию отписки.
 */
export function onViewportChanged(fn) {
  if (!native || !wa.onEvent) return () => {};
  const handler = () => fn({ height: wa.viewportHeight, stable: wa.viewportStableHeight, expanded: wa.isExpanded });
  wa.onEvent('viewportChanged', handler);
  return () => wa.offEvent && wa.offEvent('viewportChanged', handler);
}
