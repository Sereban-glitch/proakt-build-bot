/**
 * services/mock.js — демо-данные для запуска вне Telegram (?demo=1).
 * Датасет повторяет сценарии из README: ЖК Сонячний, акты, прайс,
 * упущенная выгода. Нужен только для превью и разработки UI.
 */

const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const now = Date.now();
const daysAgo = (d, h = 12) => new Date(now - d * 86400000 - h * 3600000).toISOString();

const objects = [
  { id: 1, name: 'ЖК Сонячний, кв. 45', customer: 'Иван Петренко', status: 'active',
    created_at: daysAgo(40), acts: 3, total: 41700, paid: 20000, photos: 4 },
  { id: 2, name: 'ул. Шевченко, 12, офис 3', customer: 'ООО «Ремонт+»', status: 'active',
    created_at: daysAgo(21), acts: 1, total: 14500, paid: 14500, photos: 1 },
  { id: 3, name: 'Частный дом, Бортничи', customer: 'Ольга Ковальчук', status: 'active',
    created_at: daysAgo(9), acts: 1, total: 0, paid: 0, photos: 2 },
];

const acts = [
  { id: 10, act_no: 3, object_id: 1, object_name: 'ЖК Сонячний, кв. 45', customer: 'Иван Петренко',
    total: 15300, paid: 0, photos: 1, date: daysAgo(2) },
  { id: 9, act_no: 2, object_id: 1, object_name: 'ЖК Сонячний, кв. 45', customer: 'Иван Петренко',
    total: 14400, paid: 10000, photos: 2, date: daysAgo(8) },
  { id: 8, act_no: 1, object_id: 1, object_name: 'ЖК Сонячний, кв. 45', customer: 'Иван Петренко',
    total: 12000, paid: 10000, photos: 1, date: daysAgo(20) },
  { id: 7, act_no: 1, object_id: 2, object_name: 'ул. Шевченко, 12, офис 3', customer: 'ООО «Ремонт+»',
    total: 14500, paid: 14500, photos: 0, date: daysAgo(11) },
  { id: 6, act_no: 1, object_id: 3, object_name: 'Частный дом, Бортничи', customer: 'Ольга Ковальчук',
    total: 0, paid: 0, photos: 2, date: daysAgo(4) },
];

const linesByAct = {
  10: [
    { pos: 1, name: 'штукатурка стен', qty: 45, unit: 'м²', price: 260, sum: 11700 },
    { pos: 2, name: 'грунтовка стен', qty: 45, unit: 'м²', price: 25, sum: 1125 },
    { pos: 3, name: 'демонтаж перегородки', qty: 1, unit: '', price: 2475, sum: 2475 },
  ],
  9: [
    { pos: 1, name: 'шлифовка стен штукатурки', qty: 45, unit: 'м²', price: 50, sum: 2250 },
    { pos: 2, name: 'шпаклёвка в 2 слоя', qty: 90, unit: 'м²', price: 120, sum: 10800 },
    { pos: 3, name: 'электрика, разводка', qty: 1, unit: 'компл', price: 1350, sum: 1350 },
  ],
  8: [
    { pos: 1, name: 'стяжка пола', qty: 32, unit: 'м²', price: 300, sum: 9600 },
    { pos: 2, name: 'демонтаж старой стяжки', qty: 32, unit: 'м²', price: 75, sum: 2400 },
  ],
  7: [
    { pos: 1, name: 'покраска стен', qty: 120, unit: 'м²', price: 90, sum: 10800 },
    { pos: 2, name: 'покраска потолка', qty: 74, unit: 'м²', price: 50, sum: 3700 },
  ],
  6: [],
};

const paymentsByAct = {
  10: [],
  9: [{ id: 31, act_id: 9, amount: 10000, note: '', at: daysAgo(7, 10) }],
  8: [
    { id: 22, act_id: 8, amount: 10000, note: '', at: daysAgo(19) },
  ],
  7: [
    { id: 25, act_id: 7, amount: 4500, note: 'аванс', at: daysAgo(14) },
    { id: 29, act_id: 7, amount: 10000, note: '', at: daysAgo(5) },
  ],
  6: [],
};

const photosByObject = {
  1: [
    { id: 101, object_id: 1, act_id: 9, act_no: 2, caption: 'электрика, до штукатурки', created_at: daysAgo(8, 15), demo: true },
    { id: 102, object_id: 1, act_id: 9, act_no: 2, caption: '', created_at: daysAgo(7, 18), demo: true },
    { id: 103, object_id: 1, act_id: 10, act_no: 3, caption: 'стяжка залита', created_at: daysAgo(2, 9), demo: true },
    { id: 104, object_id: 1, act_id: null, act_no: 0, caption: 'склад материалов', created_at: daysAgo(1, 11), demo: true },
  ],
  2: [{ id: 105, object_id: 2, act_id: 7, act_no: 1, caption: 'покраска окончена', created_at: daysAgo(10), demo: true }],
  3: [
    { id: 106, object_id: 3, act_id: 6, act_no: 1, caption: 'фасад, начало работ', created_at: daysAgo(4), demo: true },
    { id: 107, object_id: 3, act_id: null, act_no: 0, caption: '', created_at: daysAgo(3), demo: true },
  ],
};

const catalog = [
  { name: 'грунтовка стен', unit: 'м²', price: 25 },
  { name: 'демонтаж перегородки', unit: '', price: 2475 },
  { name: 'монтаж гипсокартона', unit: 'м²', price: 210 },
  { name: 'покраска стен', unit: 'м²', price: 90 },
  { name: 'покраска потолка', unit: 'м²', price: 50 },
  { name: 'стяжка пола', unit: 'м²', price: 300 },
  { name: 'шлифовка стен штукатурки', unit: 'м²', price: 50 },
  { name: 'шпаклёвка в 2 слоя', unit: 'м²', price: 120 },
  { name: 'штукатурка стен', unit: 'м²', price: 260 },
  { name: 'электрика, разводка', unit: 'компл', price: 1350 },
];

const stats = {
  objects: 3, acts: 5, total: 56200, paid: 34500, photos: 7,
  zero_acts: 1, unpaid_acts: 2,
};

function parseUnit(u) {
  const s = (u || '').toLowerCase().trim();
  if (s === 'м2' || s === 'м²') return 'м²';
  if (s === 'мп' || s === 'м.п' || s === 'м.п.') return 'м.п';
  if (s === 'шт') return 'шт';
  return s;
}

export const mock = {
  async dashboard() {
    await wait(350);
    return {
      user: { chat_id: 777 },
      stats,
      price_count: catalog.length,
      debt_total: stats.total - stats.paid,
      debts: acts.filter((a) => a.total - a.paid > 0.009).slice(0, 10),
      recent_acts: acts.slice(0, 5),
    };
  },
  async objects() { await wait(280); return objects; },
  async createObject(name, customer) {
    await wait(420);
    const o = { id: Math.floor(Math.random() * 1000) + 200, name, customer,
      status: 'active', created_at: new Date().toISOString(), acts: 0, total: 0, paid: 0, photos: 0 };
    objects.unshift(o);
    return o;
  },
  async archiveObject(id) {
    await wait(360);
    const i = objects.findIndex((o) => o.id === id);
    const [o] = objects.splice(i, 1);
    return { archived: true, name: o.name };
  },
  async acts() { await wait(300); return acts; },
  async act(id) {
    await wait(320);
    const brief = acts.find((a) => a.id === id);
    if (!brief) throw new Error('act not found');
    const photos = (photosByObject[brief.object_id] || []).filter((p) => p.act_id === id);
    return {
      act: brief,
      lines: linesByAct[id] || [],
      payments: paymentsByAct[id] || [],
      photos,
    };
  },
  async deleteAct(id) {
    await wait(420);
    const i = acts.findIndex((a) => a.id === id);
    const [a] = acts.splice(i, 1);
    return { deleted: true, act: a };
  },
  async addPayment(actId, amount) {
    await wait(460);
    const a = acts.find((x) => x.id === actId);
    a.paid = Math.min(a.total, a.paid + amount);
    (paymentsByAct[actId] = paymentsByAct[actId] || []).push({
      id: Math.floor(Math.random() * 1000) + 300, act_id: actId, amount, note: '', at: new Date().toISOString(),
    });
    return { act: a };
  },
  async deletePayment(actId, payId) {
    await wait(380);
    const list = paymentsByAct[actId] || [];
    const i = list.findIndex((p) => p.id === payId);
    const [p] = list.splice(i, 1);
    const a = acts.find((x) => x.id === actId);
    if (a && p) a.paid = Math.max(0, a.paid - p.amount);
    return { deleted: true, payment: p };
  },
  async price() { await wait(260); return { items: [...catalog], count: catalog.length }; },
  async upsertPrice(name, unit, price) {
    await wait(420);
    const norm = name.trim().toLowerCase().replace(/ё/g, 'е');
    const i = catalog.findIndex((c) => c.name === norm);
    let prev = 0, existed = false;
    if (i >= 0) { prev = catalog[i].price; existed = true; catalog[i] = { name: norm, unit, price }; }
    else catalog.push({ name: norm, unit, price });
    catalog.sort((a, b) => a.name.localeCompare(b.name, 'ru'));
    return { item: { name: norm, unit, price }, prev, existed, changed: existed && prev !== price, count: catalog.length };
  },
  async deletePrice(name) {
    await wait(380);
    const i = catalog.findIndex((c) => c.name === name);
    if (i < 0) throw new Error('нет такой позиции');
    const [item] = catalog.splice(i, 1);
    return { deleted: true, item };
  },
  async photos(objectId) { await wait(260); return photosByObject[objectId] || []; },
  /** В demo файлы не живут на диске — рисуем SVG-заглушку с подписью. */
  photoURL(photo) {
    const label = encodeURIComponent(photo.caption || 'скрытые работы');
    const svg = `<svg xmlns='http://www.w3.org/2000/svg' width='480' height='480'>`
      + `<defs><linearGradient id='g' x1='0' y1='0' x2='1' y2='1'>`
      + `<stop offset='0' stop-color='#22303f'/><stop offset='1' stop-color='#17212b'/></linearGradient></defs>`
      + `<rect width='480' height='480' fill='url(#g)'/>`
      + `<text x='240' y='228' font-size='120' text-anchor='middle'>📷</text>`
      + `<text x='240' y='290' font-size='22' fill='#9fb0bf' text-anchor='middle' font-family='sans-serif'>${label}</text>`
      + `</svg>`;
    return 'data:image/svg+xml;utf8,' + encodeURIComponent(svg);
  },
  async deletePhoto(_objectId, photoId) {
    await wait(340);
    for (const list of Object.values(photosByObject)) {
      const i = list.findIndex((p) => p.id === photoId);
      if (i >= 0) { const [p] = list.splice(i, 1); return { deleted: true, photo: p }; }
    }
    return { deleted: true, photo: { id: photoId } };
  },
};

/* ============================================================
   v0.5 · сметы и шаблоны. Датасет повторяет живую смету мастера
   «Парковий 2» (малярные работы, 190 383 грн): техпоследовательность
   грунт → шпаклёвка → стеклохолст → покраска; часть позиций скрытая
   подготовка, коэффициент потолков 300 см = 1.30.
   ============================================================ */

const estimates = [
  {
    id: 3, object_id: 1, object_name: 'ЖК Сонячний, кв. 45',
    title: 'Малярные работы под покраску', status: 'approved', coeff: 1.3,
    note: 'Потолки 300 см — коэффициент сложности 1.30 (прайс-лист)',
    created_at: daysAgo(25),
  },
  {
    id: 2, object_id: 2, object_name: 'ул. Шевченко, 12, офис 3',
    title: 'Офис: стеклохолст + покраска', status: 'sent', coeff: 1,
    note: '', created_at: daysAgo(12),
  },
  {
    id: 1, object_id: 3, object_name: 'Частный дом, Бортничи',
    title: 'Фасад, подготовка', status: 'draft', coeff: 1,
    note: '', created_at: daysAgo(3),
  },
];

const estLines = {
  3: [
    { id: 31, est_id: 3, pos: 1, name: 'укрывка окон гофрокартоном', qty: 12, unit: 'м.п', price: 25, hidden: true, done: true, note: '' },
    { id: 32, est_id: 3, pos: 2, name: 'заделка штроб', qty: 47, unit: 'м.п', price: 60, hidden: true, done: true, note: 'штробы под электрику' },
    { id: 33, est_id: 3, pos: 3, name: 'грунтовка стен перед шпаклёвкой', qty: 146.4, unit: 'м²', price: 25, hidden: true, done: true, note: '' },
    { id: 34, est_id: 3, pos: 4, name: 'армировка откосов стекловолоконной сеткой', qty: 18.5, unit: 'м.п', price: 80, hidden: true, done: true, note: '' },
    { id: 35, est_id: 3, pos: 5, name: 'шлифовка стен штукатурки перед шпаклёвкой', qty: 146.4, unit: 'м²', price: 50, hidden: true, done: true, note: '' },
    { id: 36, est_id: 3, pos: 6, name: 'шпаклёвка стен под стеклохолст', qty: 140.5, unit: 'м²', price: 140, hidden: true, done: true, note: '2 слоя' },
    { id: 37, est_id: 3, pos: 7, name: 'грунтовка стен перед поклейкой стеклохолста', qty: 75, unit: 'м²', price: 25, hidden: true, done: false, note: '' },
    { id: 38, est_id: 3, pos: 8, name: 'поклейка стеклохолста на стены', qty: 75, unit: 'м²', price: 120, hidden: false, done: false, note: '' },
    { id: 39, est_id: 3, pos: 9, name: 'шпаклёвка стен под покраску по стеклохолсту', qty: 75, unit: 'м²', price: 200, hidden: true, done: false, note: '' },
    { id: 40, est_id: 3, pos: 10, name: 'шлифовка шпаклёвки стен под покраску', qty: 75, unit: 'м²', price: 50, hidden: true, done: false, note: '' },
    { id: 41, est_id: 3, pos: 11, name: 'грунтовка стен под покраску', qty: 129.3, unit: 'м²', price: 25, hidden: true, done: false, note: '' },
    { id: 42, est_id: 3, pos: 12, name: 'нанесение грунт-краски на стены', qty: 129.3, unit: 'м²', price: 60, hidden: false, done: false, note: '' },
    { id: 43, est_id: 3, pos: 13, name: 'покраска стен безвоздушным методом', qty: 129.3, unit: 'м²', price: 120, hidden: false, done: false, note: '2 слоя, Dulux' },
  ],
  2: [
    { id: 21, est_id: 2, pos: 1, name: 'шлифовка стен штукатурки перед шпаклёвкой', qty: 60, unit: 'м²', price: 50, hidden: true, done: true, note: '' },
    { id: 22, est_id: 2, pos: 2, name: 'шпаклёвка стен под покраску', qty: 60, unit: 'м²', price: 180, hidden: true, done: false, note: '' },
    { id: 23, est_id: 2, pos: 3, name: 'покраска стен валиком', qty: 60, unit: 'м²', price: 120, hidden: false, done: false, note: '' },
  ],
  1: [],
};

function briefOf(e) {
  const lines = estLines[e.id] || [];
  let total = 0, visible = 0, hidden = 0, done = 0;
  for (const l of lines) {
    const s = l.qty * l.price * e.coeff;
    total += s;
    if (l.hidden) hidden += s; else visible += s;
    if (l.done) done += s;
  }
  return {
    id: e.id, object_id: e.object_id, object_name: e.object_name, title: e.title,
    status: e.status, coeff: e.coeff, note: e.note, share_token: e.share_token || '',
    lines: lines.length, total, visible, hidden, done_sum: done,
    hidden_share: total > 0 ? Math.round((hidden / total) * 100) : 0,
    created_at: e.created_at,
  };
}

const templates = [
  {
    id: 1, name: 'Стена под покраску (полный цикл)',
    lines: [
      { name: 'грунтовка стен перед шпаклёвкой', qty: 1, unit: 'м²', price: 25, hidden: true },
      { name: 'шпаклёвка стен под покраску', qty: 1, unit: 'м²', price: 180, hidden: true },
      { name: 'шлифовка шпаклёвки стен под покраску', qty: 1, unit: 'м²', price: 50, hidden: true },
      { name: 'грунтовка стен под покраску', qty: 1, unit: 'м²', price: 25, hidden: true },
      { name: 'покраска стен безвоздушным методом', qty: 1, unit: 'м²', price: 120, hidden: false },
    ],
  },
  {
    id: 2, name: 'Потолок под покраску (коэфф. 1.3)',
    lines: [
      { name: 'грунтовка потолка перед шпаклёвкой', qty: 1, unit: 'м²', price: 30, hidden: true },
      { name: 'шпаклёвка потолка под покраску', qty: 1, unit: 'м²', price: 240, hidden: true },
      { name: 'шлифовка шпаклёвки потолка под покраску', qty: 1, unit: 'м²', price: 75, hidden: true },
      { name: 'покраска потолка безвоздушным методом', qty: 1, unit: 'м²', price: 160, hidden: false },
    ],
  },
  {
    id: 3, name: 'Подготовка под стеклохолст',
    lines: [
      { name: 'заделка штроб', qty: 1, unit: 'м.п', price: 60, hidden: true },
      { name: 'армировка откосов стекловолоконной сеткой', qty: 1, unit: 'м.п', price: 80, hidden: true },
      { name: 'шпаклёвка стен под стеклохолст', qty: 1, unit: 'м²', price: 140, hidden: true },
      { name: 'грунтовка стен перед поклейкой стеклохолста', qty: 1, unit: 'м²', price: 25, hidden: true },
      { name: 'поклейка стеклохолста на стены', qty: 1, unit: 'м²', price: 120, hidden: false },
    ],
  },
];

let tplSeq = 10;
let lineSeq = 100;

/** demo-парсер: тот же контракт, что POST /api/parse-line (для ?demo=1). */
function demoParse(text) {
  const suspicious = /(\D+\s+\d+(\.\d+)?\s+){2,}/.test(text) && !/\d[^\d]*\d[^\d]*\d/.test(text) && text.trim().split(/\s+/).length > 4;
  const out = [];
  for (const part of text.split(/[,;]/).map((s) => s.trim()).filter(Boolean)) {
    const m = part.match(/^(.+?)[\s]+(\d+(?:[.,]\d+)?)\s*(м²|м2|м\.?п\.?|шт)?\s*(\d+(?:[.,]\d+)?)?$/i);
    if (!m) continue;
    const name = m[1].trim();
    const qty = parseFloat(m[2].replace(',', '.'));
    let unit = (m[3] || '').toLowerCase();
    if (unit === 'м2') unit = 'м²';
    if (unit.startsWith('м.п') || unit === 'мп') unit = 'м.п';
    const price = m[4] !== undefined ? parseFloat(m[4].replace(',', '.')) : 0;
    if (!name || Number.isNaN(qty)) continue;
    // цена из demo-прайса, если не указана
    let p = price;
    if (!p) {
      const cat = catalog.find((c) => c.name.includes(name.toLowerCase()) || name.toLowerCase().includes(c.name));
      if (cat) p = cat.price;
    }
    out.push({ name, qty, unit, price: p, sum: qty * p });
  }
  return Promise.resolve({ suspicious, lines: out });
}

Object.assign(mock, {
  async estimates(objectId) {
    await wait(280);
    return estimates
      .filter((e) => !objectId || e.object_id === objectId)
      .map(briefOf);
  },
  async estimate(id) {
    await wait(320);
    const e = estimates.find((x) => x.id === id);
    if (!e) throw new Error('смета не найдена');
    return { estimate: briefOf(e), lines: [...(estLines[id] || [])] };
  },
  async createEstimate(objectId, title, coeff = 1, note = '') {
    await wait(420);
    const obj = objects.find((o) => o.id === objectId);
    if (!obj) throw new Error('объект не найден');
    const e = {
      id: Math.max(...estimates.map((x) => x.id)) + 1, object_id: objectId,
      object_name: obj.name, title, status: 'draft', coeff, note, created_at: new Date().toISOString(),
    };
    estimates.unshift(e);
    estLines[e.id] = [];
    return e;
  },
  async patchEstimate(id, patch) {
    await wait(300);
    const e = estimates.find((x) => x.id === id);
    if (!e) throw new Error('смета не найдена');
    if (patch.title) e.title = patch.title;
    if (patch.status) e.status = patch.status;
    if (patch.coeff) e.coeff = patch.coeff;
    if (patch.note !== undefined) e.note = patch.note;
    return briefOf(e);
  },
  async deleteEstimate(id) {
    await wait(380);
    const i = estimates.findIndex((x) => x.id === id);
    const [e] = estimates.splice(i, 1);
    delete estLines[id];
    return { deleted: true, estimate: briefOf(e) };
  },
  async addEstLine(id, line) {
    await wait(260);
    const list = (estLines[id] = estLines[id] || []);
    const l = {
      id: ++lineSeq, est_id: id, pos: list.length + 1,
      name: line.name, qty: line.qty, unit: parseUnit(line.unit), price: line.price,
      hidden: Boolean(line.hidden), done: false, note: line.note || '',
    };
    list.push(l);
    return l;
  },
  async addEstLinesBulk(id, lines) {
    await wait(420);
    for (const line of lines) await mock.addEstLine(id, line);
    return { added: lines.length, estimate: await mock.estimate(id).then((r) => r.estimate) };
  },
  async patchEstLine(id, lineId, patch) {
    await wait(240);
    const l = (estLines[id] || []).find((x) => x.id === lineId);
    if (!l) throw new Error('позиция не найдена');
    if (patch.name) l.name = patch.name;
    if (patch.qty !== undefined) l.qty = patch.qty;
    if (patch.price !== undefined) l.price = patch.price;
    if (patch.unit !== undefined) l.unit = parseUnit(patch.unit);
    if (patch.hidden !== undefined) l.hidden = patch.hidden;
    if (patch.done !== undefined) l.done = patch.done;
    if (patch.note !== undefined) l.note = patch.note;
    return { ...l };
  },
  async deleteEstLine(id, lineId) {
    await wait(300);
    const list = estLines[id] || [];
    const i = list.findIndex((x) => x.id === lineId);
    const [l] = list.splice(i, 1);
    list.forEach((x, j) => { x.pos = j + 1; });
    return { deleted: true, line: l };
  },
  async moveEstLine(id, lineId, dir) {
    await wait(200);
    const list = estLines[id] || [];
    const i = list.findIndex((x) => x.id === lineId);
    const j = dir === 'up' ? i - 1 : i + 1;
    if (i < 0 || j < 0 || j >= list.length) throw new Error('двигать некуда');
    [list[i], list[j]] = [list[j], list[i]];
    list.forEach((x, k) => { x.pos = k + 1; });
    return { moved: true };
  },
  async estimateShare(id) {
    await wait(360);
    const e = estimates.find((x) => x.id === id);
    e.share_token = Math.random().toString(16).slice(2).padEnd(32, '0');
    return { token: e.share_token, url: `https://demo.proakt.app/s/${e.share_token}` };
  },
  estimateXlsxURL: (id) => `/api/estimates/${id}/xlsx`,
  parseLine: (text) => wait(180).then(() => demoParse(text)),

  async templates() {
    await wait(240);
    return { items: templates.map((t) => ({ ...t, lines: [...t.lines] })), count: templates.length };
  },
  async upsertTemplate(name, lines) {
    await wait(380);
    const norm = name.trim();
    const i = templates.findIndex((t) => t.name === norm);
    if (i >= 0) { templates[i] = { ...templates[i], lines }; return templates[i]; }
    const t = { id: ++tplSeq, name: norm, lines };
    templates.push(t);
    return t;
  },
  async deleteTemplate(id) {
    await wait(300);
    const i = templates.findIndex((t) => t.id === id);
    if (i < 0) throw new Error('шаблон не найден');
    const [t] = templates.splice(i, 1);
    return { deleted: true, template: t };
  },
});
