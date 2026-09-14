/**
 * services/api.js — клиент REST API ПрорАКТА.
 *
 * Внутри Telegram подписывает каждый запрос заголовком
 * X-Telegram-Init-Data (сервер проверяет HMAC и скоупит данные по chat_id).
 * Вне Telegram (?demo=1 или открыт как сайт) прозрачно переключается на mock.
 */

import { initData, isDemo } from './tg.js';
import { mock } from './mock.js';

export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

async function request(path, { method = 'GET', body } = {}) {
  const headers = {};
  if (initData) headers['X-Telegram-Init-Data'] = initData;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  let res;
  try {
    res = await fetch('/api' + path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError(0, 'Нет связи с сервером — проверь интернет');
  }

  if (res.status === 401) {
    throw new ApiError(401, 'Авторизация не прошла — открой приложение из бота (кнопка 🧰)');
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, data?.error || `Ошибка сервера (${res.status})`);
  }
  return data;
}

/** Реальные эндпоинты (внутри Telegram). */
const live = {
  dashboard: () => request('/dashboard'),
  objects: () => request('/objects'),
  createObject: (name, customer) => request('/objects', { method: 'POST', body: { name, customer } }),
  archiveObject: (id) => request(`/objects/${id}/archive`, { method: 'POST' }),

  acts: () => request('/acts?limit=100'),
  act: (id) => request(`/acts/${id}`),
  deleteAct: (id) => request(`/acts/${id}`, { method: 'DELETE' }),
  addPayment: (actId, amount) => request(`/acts/${actId}/payments`, { method: 'POST', body: { amount } }),
  deletePayment: (payId) => request(`/payments/${payId}`, { method: 'DELETE' }),

  price: () => request('/price'),
  upsertPrice: (name, unit, price) => request('/price', { method: 'POST', body: { name, unit, price } }),
  deletePrice: (name) => request(`/price/${encodeURIComponent(name)}`, { method: 'DELETE' }),

  photos: (objectId) => request(`/photos?object_id=${objectId}`),
  photoURL: (photo) => `/api/photos/${photo.id}/file`,
  deletePhoto: (photoId) => request(`/photos/${photoId}`, { method: 'DELETE' }),

  // --- сметы (v0.5) ---
  estimates: (objectId) => request('/estimates' + (objectId ? `?object_id=${objectId}` : '')),
  estimate: (id) => request(`/estimates/${id}`),
  createEstimate: (objectId, title, coeff = 1, note = '') =>
    request('/estimates', { method: 'POST', body: { object_id: objectId, title, coeff, note } }),
  patchEstimate: (id, patch) => request(`/estimates/${id}`, { method: 'PATCH', body: patch }),
  deleteEstimate: (id) => request(`/estimates/${id}`, { method: 'DELETE' }),
  addEstLine: (id, line) => request(`/estimates/${id}/lines`, { method: 'POST', body: line }),
  addEstLinesBulk: (id, lines) => request(`/estimates/${id}/lines`, { method: 'POST', body: { lines } }),
  patchEstLine: (id, lineId, patch) => request(`/estimates/${id}/lines/${lineId}`, { method: 'PATCH', body: patch }),
  deleteEstLine: (id, lineId) => request(`/estimates/${id}/lines/${lineId}`, { method: 'DELETE' }),
  moveEstLine: (id, lineId, dir) => request(`/estimates/${id}/lines/${lineId}/move`, { method: 'POST', body: { dir } }),
  estimateShare: (id) => request(`/estimates/${id}/share`, { method: 'POST', body: {} }),
  estimateXlsxURL: (id) => `/api/estimates/${id}/xlsx`,
  parseLine: (text) => request('/parse-line', { method: 'POST', body: { text } }),

  // --- шаблоны работ (v0.5) ---
  templates: () => request('/templates'),
  upsertTemplate: (name, lines) => request('/templates', { method: 'POST', body: { name, lines } }),
  deleteTemplate: (id) => request(`/templates/${id}`, { method: 'DELETE' }),
};

/** Публичный API: demo → mock, иначе → живой бэкенд. */
export const api = isDemo ? mock : live;
