/**
 * services/format.js — деньги, даты, множественные числа.
 * Формат денег совпадает с ботом: «11 700 грн» (узкий пробел).
 */

const NARROW = '\u202F'; // narrow no-break space

export function money(v, { withUnit = true, sign = false } = {}) {
  const n = Number(v) || 0;
  const neg = n < -0.009;
  const abs = Math.abs(n);
  const int = Math.floor(abs);
  const frac = Math.round((abs - int) * 100);
  let s = int.toLocaleString('ru-RU').replace(/\u00A0/g, NARROW);
  if (frac > 0) s += ',' + String(frac).padStart(2, '0');
  if (sign && !neg) s = '+' + s;
  if (neg) s = '−' + s;
  return withUnit ? `${s}${NARROW}грн` : s;
}

/** Короткий формат для метрик: 11 700 грн / 45 000 грн (без копеек). */
export function moneyShort(v) {
  return money(Math.round(Number(v) || 0));
}

const MONTHS_UK = ['янв','фев','мар','апр','мая','июн','июл','авг','сен','окт','ноя','дек'];

/** «31 авг» / «31 авг 2025». */
export function dateShort(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  const sameYear = d.getFullYear() === new Date().getFullYear();
  return `${d.getDate()}${NARROW}${MONTHS_UK[d.getMonth()]}${sameYear ? '' : NARROW + d.getFullYear()}`;
}

/** «31.08 15:28» — как бот подписывает фото. */
export function dateTime(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  const p = (x) => String(x).padStart(2, '0');
  return `${p(d.getDate())}.${p(d.getMonth() + 1)}${NARROW}${p(d.getHours())}:${p(d.getMinutes())}`;
}

/** Русские множественные: plural(2, 'акт','акта','актов'). */
export function plural(n, one, few, many) {
  const m10 = n % 10, m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return few;
  return many;
}

/** «2 акта», «5 актов» — n + правильная форма. */
export function pluralN(n, one, few, many) {
  return `${n}${NARROW}${plural(n, one, few, many)}`;
}

/** Единица позиции с знаком количества: «45 м²». */
export function qtyLine(qty, unit) {
  if (!unit) return fmtQty(qty);
  return `${fmtQty(qty)}${NARROW}${unit}`;
}

function fmtQty(q) {
  const n = Number(q) || 0;
  return n.toLocaleString('ru-RU', { maximumFractionDigits: 2 }).replace(/\u00A0/g, NARROW);
}
