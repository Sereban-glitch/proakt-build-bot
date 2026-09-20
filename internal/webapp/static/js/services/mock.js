/**
 * services/mock.js — демо-данные для запуска вне Telegram (?demo=1).
 * Главный сценарий собран из файлов Виталия по объекту «Парковый 2»:
 * 12 актов, смета на 190 383,50 ₴ и рабочий прайс. Личные данные скрыты,
 * а оплаты и сроки намеренно помечены как демонстрационные.
 */

const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const now = Date.now();
const daysAgo = (d, h = 12) => new Date(now - d * 86400000 - h * 3600000).toISOString();

const objects = [
  { id: 1, name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', status: 'active',
    created_at: daysAgo(126), acts: 12, total: 549526, paid: 505000, photos: 8, progress: 91 },
  { id: 2, name: 'Квартира · новый расчёт', customer: 'Заказчик (демо)', status: 'active',
    created_at: daysAgo(12), acts: 0, total: 0, paid: 0, photos: 1, progress: 8 },
  { id: 3, name: 'Частный дом · фасад', customer: 'Заказчик (демо)', status: 'active',
    created_at: daysAgo(58), acts: 2, total: 94500, paid: 94500, photos: 6, progress: 100 },
];

const acts = [
  { id: 112, act_no: 12, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 64309, paid: 19783, photos: 3, date: daysAgo(2) },
  { id: 111, act_no: 11, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 55651, paid: 55651, photos: 2, date: daysAgo(9) },
  { id: 110, act_no: 10, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 50023, paid: 50023, photos: 1, date: daysAgo(16) },
  { id: 109, act_no: 9, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 48127, paid: 48127, photos: 0, date: daysAgo(23) },
  { id: 108, act_no: 8, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 48340, paid: 48340, photos: 0, date: daysAgo(31) },
  { id: 107, act_no: 7, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 46026, paid: 46026, photos: 1, date: daysAgo(39) },
  { id: 106, act_no: 6, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 41024, paid: 41024, photos: 0, date: daysAgo(47) },
  { id: 105, act_no: 5, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 39470, paid: 39470, photos: 0, date: daysAgo(55) },
  { id: 104, act_no: 4, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 40720, paid: 40720, photos: 0, date: daysAgo(64) },
  { id: 103, act_no: 3, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 34296, paid: 34296, photos: 1, date: daysAgo(73) },
  { id: 102, act_no: 2, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 42560, paid: 42560, photos: 0, date: daysAgo(83) },
  { id: 101, act_no: 1, object_id: 1, object_name: 'Парковый 2 · квартира', customer: 'Заказчик · данные скрыты', total: 38980, paid: 38980, photos: 0, date: daysAgo(94) },
  { id: 202, act_no: 2, object_id: 3, object_name: 'Частный дом · фасад', customer: 'Заказчик (демо)', total: 52500, paid: 52500, photos: 3, date: daysAgo(21) },
  { id: 201, act_no: 1, object_id: 3, object_name: 'Частный дом · фасад', customer: 'Заказчик (демо)', total: 42000, paid: 42000, photos: 3, date: daysAgo(43) },
];

const linesByAct = {
  112: [
    { pos: 1, name: 'занос материалов', qty: 23, unit: 'шт', price: 25, sum: 575 },
    { pos: 2, name: 'укрывка плёнкой перед малярными работами', qty: 140, unit: 'м²', price: 30, sum: 4200 },
    { pos: 3, name: 'защита примыканий плёнкой и лентой', qty: 51, unit: 'м.п', price: 30, sum: 1530 },
    { pos: 4, name: 'покраска стен валиком', qty: 104.9, unit: 'м²', price: 120, sum: 12588 },
    { pos: 5, name: 'покраска откосов и участков стен', qty: 115.8, unit: 'м.п', price: 120, sum: 13896 },
    { pos: 6, name: 'покраска потолка безвоздушным методом', qty: 90.4, unit: 'м²', price: 160, sum: 14464 },
    { pos: 7, name: 'покраска откосов потолка безвоздушно', qty: 106.6, unit: 'м.п', price: 160, sum: 17056 },
  ],
  111: [
    { pos: 1, name: 'подготовка и армирование откосов', qty: 1, unit: 'компл', price: 2495, sum: 2495 },
    { pos: 2, name: 'грунтовка стен и откосов под покраску', qty: 1, unit: 'компл', price: 5517, sum: 5517 },
    { pos: 3, name: 'укрывка поверхностей плёнкой и лентой', qty: 1, unit: 'компл', price: 8550, sum: 8550 },
    { pos: 4, name: 'нанесение грунт-краски на стены и откосы', qty: 1, unit: 'компл', price: 15449, sum: 15449 },
    { pos: 5, name: 'грунтовка и грунт-краска потолков', qty: 1, unit: 'компл', price: 23640, sum: 23640 },
  ],
  110: [
    { pos: 1, name: 'занос материалов', qty: 8, unit: 'шт', price: 50, sum: 400 },
    { pos: 2, name: 'шпаклёвка стен под покраску по стеклохолсту', qty: 64.9, unit: 'м²', price: 180, sum: 11682 },
    { pos: 3, name: 'шпаклёвка откосов под покраску', qty: 40.8, unit: 'м.п', price: 180, sum: 7344 },
    { pos: 4, name: 'шлифовка стен под покраску', qty: 64.9, unit: 'м²', price: 60, sum: 3894 },
    { pos: 5, name: 'шлифовка откосов под покраску', qty: 40.8, unit: 'м.п', price: 60, sum: 2448 },
    { pos: 6, name: 'шпаклёвка потолка под покраску', qty: 40.4, unit: 'м²', price: 240, sum: 9696 },
    { pos: 7, name: 'шпаклёвка откосов потолка', qty: 36.6, unit: 'м.п', price: 240, sum: 8784 },
    { pos: 8, name: 'шлифовка потолка под покраску', qty: 40.4, unit: 'м²', price: 75, sum: 3030 },
    { pos: 9, name: 'шлифовка откосов потолка', qty: 36.6, unit: 'м.п', price: 75, sum: 2745 },
  ],
  109: [{ pos: 1, name: 'Работы по акту №9 · архив Виталия', qty: 1, unit: 'компл', price: 48127, sum: 48127 }],
  108: [{ pos: 1, name: 'Работы по акту №8 · архив Виталия', qty: 1, unit: 'компл', price: 48340, sum: 48340 }],
  107: [{ pos: 1, name: 'Работы по акту №7 · архив Виталия', qty: 1, unit: 'компл', price: 46026, sum: 46026 }],
  106: [{ pos: 1, name: 'Работы по акту №6 · архив Виталия', qty: 1, unit: 'компл', price: 41024, sum: 41024 }],
  105: [{ pos: 1, name: 'Работы по акту №5 · архив Виталия', qty: 1, unit: 'компл', price: 39470, sum: 39470 }],
  104: [{ pos: 1, name: 'Работы по акту №4 · архив Виталия', qty: 1, unit: 'компл', price: 40720, sum: 40720 }],
  103: [{ pos: 1, name: 'Работы по акту №3 · архив Виталия', qty: 1, unit: 'компл', price: 34296, sum: 34296 }],
  102: [{ pos: 1, name: 'Работы по акту №2 · архив Виталия', qty: 1, unit: 'компл', price: 42560, sum: 42560 }],
  101: [{ pos: 1, name: 'Работы по акту №1 · архив Виталия', qty: 1, unit: 'компл', price: 38980, sum: 38980 }],
  202: [{ pos: 1, name: 'Покраска фасада (демо)', qty: 150, unit: 'м²', price: 350, sum: 52500 }],
  201: [{ pos: 1, name: 'Подготовка фасада (демо)', qty: 140, unit: 'м²', price: 300, sum: 42000 }],
};

const paymentsByAct = {
  112: [{ id: 1121, act_id: 112, amount: 19783, note: 'частичная оплата · демо', at: daysAgo(1) }],
  111: [{ id: 1111, act_id: 111, amount: 55651, note: 'оплачено · демо', at: daysAgo(8) }],
  110: [{ id: 1101, act_id: 110, amount: 50023, note: 'оплачено · демо', at: daysAgo(15) }],
  109: [{ id: 1091, act_id: 109, amount: 48127, note: 'оплачено · демо', at: daysAgo(22) }],
  108: [{ id: 1081, act_id: 108, amount: 48340, note: 'оплачено · демо', at: daysAgo(30) }],
  107: [{ id: 1071, act_id: 107, amount: 46026, note: 'оплачено · демо', at: daysAgo(38) }],
  106: [{ id: 1061, act_id: 106, amount: 41024, note: 'оплачено · демо', at: daysAgo(46) }],
  105: [{ id: 1051, act_id: 105, amount: 39470, note: 'оплачено · демо', at: daysAgo(54) }],
  104: [{ id: 1041, act_id: 104, amount: 40720, note: 'оплачено · демо', at: daysAgo(63) }],
  103: [{ id: 1031, act_id: 103, amount: 34296, note: 'оплачено · демо', at: daysAgo(72) }],
  102: [{ id: 1021, act_id: 102, amount: 42560, note: 'оплачено · демо', at: daysAgo(82) }],
  101: [{ id: 1011, act_id: 101, amount: 38980, note: 'оплачено · демо', at: daysAgo(93) }],
  202: [{ id: 2021, act_id: 202, amount: 52500, note: '', at: daysAgo(20) }],
  201: [{ id: 2011, act_id: 201, amount: 42000, note: '', at: daysAgo(42) }],
};

const photosByObject = {
  1: [
    { id: 1001, object_id: 1, act_id: 112, act_no: 12, caption: 'До → грунтование → готовое основание', created_at: daysAgo(2, 9), demo: true, asset: 'assets/hidden-work-primer-report.webp' },
    { id: 1002, object_id: 1, act_id: 111, act_no: 11, caption: 'Грунт-краска на стенах', created_at: daysAgo(9, 15), demo: true },
    { id: 1003, object_id: 1, act_id: 111, act_no: 11, caption: 'Защита поверхностей плёнкой', created_at: daysAgo(9, 12), demo: true },
    { id: 1004, object_id: 1, act_id: 110, act_no: 10, caption: 'Шлифовка перед покраской', created_at: daysAgo(16, 10), demo: true },
    { id: 1005, object_id: 1, act_id: 107, act_no: 7, caption: 'Стеклохолст и армирование', created_at: daysAgo(39), demo: true },
    { id: 1006, object_id: 1, act_id: 103, act_no: 3, caption: 'Стыки гипсокартона с сеткой', created_at: daysAgo(73), demo: true },
    { id: 1007, object_id: 1, act_id: null, act_no: 0, caption: 'Материалы на объекте', created_at: daysAgo(4), demo: true },
    { id: 1008, object_id: 1, act_id: null, act_no: 0, caption: 'Общий вид объекта', created_at: daysAgo(1), demo: true },
  ],
  2: [{ id: 2001, object_id: 2, act_id: null, act_no: 0, caption: 'Замеры помещения · демо', created_at: daysAgo(3), demo: true }],
  3: [
    { id: 3001, object_id: 3, act_id: 201, act_no: 1, caption: 'Фасад, подготовка · демо', created_at: daysAgo(43), demo: true },
    { id: 3002, object_id: 3, act_id: 202, act_no: 2, caption: 'Фасад, готово · демо', created_at: daysAgo(21), demo: true },
  ],
};

const catalog = [
  { name: 'шлифовка стен штукатурки перед шпаклёвкой', unit: 'м²', price: 60 },
  { name: 'грунтовка стен перед шпаклёвкой', unit: 'м²', price: 25 },
  { name: 'армировка стен стекловолоконной сеткой', unit: 'м²', price: 120 },
  { name: 'шпаклёвка стен под стеклохолст', unit: 'м²', price: 120 },
  { name: 'шлифовка стен под стеклохолст', unit: 'м²', price: 40 },
  { name: 'поклейка стеклохолста на стены', unit: 'м²', price: 110 },
  { name: 'шпаклёвка стен под покраску', unit: 'м²', price: 180 },
  { name: 'шлифовка стен под покраску', unit: 'м²', price: 60 },
  { name: 'нанесение грунт-краски на стены', unit: 'м²', price: 70 },
  { name: 'покраска стен валиком', unit: 'м²', price: 120 },
  { name: 'покраска стен безвоздушным методом', unit: 'м²', price: 140 },
  { name: 'грунтовка потолка перед шпаклёвкой', unit: 'м²', price: 30 },
  { name: 'заделка стыков гипсокартона потолка', unit: 'м²', price: 100 },
  { name: 'шпаклёвка потолка под стеклохолст', unit: 'м²', price: 150 },
  { name: 'шлифовка потолка под стеклохолст', unit: 'м²', price: 60 },
  { name: 'поклейка стеклохолста на потолок', unit: 'м²', price: 140 },
  { name: 'шпаклёвка потолка под покраску', unit: 'м²', price: 240 },
  { name: 'шлифовка потолка под покраску', unit: 'м²', price: 75 },
  { name: 'нанесение грунт-краски на потолок', unit: 'м²', price: 90 },
  { name: 'покраска потолка валиком', unit: 'м²', price: 140 },
  { name: 'покраска потолка безвоздушным методом', unit: 'м²', price: 160 },
  { name: 'укрывка плёнкой перед малярными работами', unit: 'м²', price: 30 },
  { name: 'заделка штроб', unit: 'м.п', price: 70 },
  { name: 'гидроизоляция брусов скрытой двери', unit: 'м.п', price: 90 },
  { name: 'штроба по периметру скрытой двери', unit: 'м.п', price: 110 },
  { name: 'армирование примыкания скрытой двери', unit: 'м.п', price: 350 },
  { name: 'монтаж откосов из гипсокартона', unit: 'м.п', price: 240 },
  { name: 'перфорированный пластиковый уголок', unit: 'м.п', price: 90 },
  { name: 'лента Straight Flex на углы', unit: 'м.п', price: 130 },
  { name: 'грунтовка двери скрытого монтажа', unit: 'шт', price: 240 },
  { name: 'покраска двери скрытого монтажа', unit: 'шт', price: 480 },
];

const stats = {
  objects: 3, acts: 14, total: 644026, paid: 599500, photos: 15,
  zero_acts: 1, unpaid_acts: 1,
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
    if (photo.asset) return photo.asset;
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
    id: 3, object_id: 1, object_name: 'Парковый 2 · квартира',
    title: 'Малярные работы · этап под покраску', status: 'approved', coeff: 1,
    note: 'Источник: файл «Парковый 2». 38 позиций с известным объёмом · 190 383,50 ₴. Неоценённая укрывка в итог не включена.',
    created_at: daysAgo(118),
  },
  {
    id: 2, object_id: 2, object_name: 'Квартира · новый расчёт',
    title: 'Стены и потолки · предварительный расчёт', status: 'sent', coeff: 1,
    note: 'Демонстрационная смета по актуальному прайсу Виталия.', created_at: daysAgo(8),
  },
  {
    id: 1, object_id: 3, object_name: 'Частный дом · фасад',
    title: 'Фасад · подготовка и покраска', status: 'done', coeff: 1,
    note: 'Демонстрационный закрытый объект.', created_at: daysAgo(58),
  },
];

const estLines = {
  3: [
    { id: 301, est_id: 3, pos: 1, name: 'укрывка окон гофрокартоном', qty: 12, unit: 'м.п', price: 25, hidden: true, done: true, note: '' },
    { id: 302, est_id: 3, pos: 2, name: 'заделка штроб', qty: 47, unit: 'м.п', price: 60, hidden: true, done: true, note: '' },
    { id: 303, est_id: 3, pos: 3, name: 'поклейка пенополистирола на верхний откос балкона', qty: 1, unit: 'шт', price: 100, hidden: true, done: true, note: '' },
    { id: 304, est_id: 3, pos: 4, name: 'грунтовка откосов перед штукатуркой', qty: 18.5, unit: 'м.п', price: 25, hidden: true, done: true, note: '' },
    { id: 305, est_id: 3, pos: 5, name: 'установка перфорированного пластикового уголка', qty: 9.8, unit: 'м.п', price: 80, hidden: true, done: true, note: '' },
    { id: 306, est_id: 3, pos: 6, name: 'армировка откосов стекловолоконной сеткой', qty: 18.5, unit: 'м.п', price: 80, hidden: true, done: true, note: '' },
    { id: 307, est_id: 3, pos: 7, name: 'отпуск дверных проёмов', qty: 5, unit: 'шт', price: 200, hidden: true, done: true, note: '' },
    { id: 308, est_id: 3, pos: 8, name: 'шлифовка стен штукатурки перед шпаклёвкой', qty: 146.4, unit: 'м²', price: 50, hidden: true, done: true, note: '' },
    { id: 309, est_id: 3, pos: 9, name: 'шлифовка откосов перед шпаклёвкой', qty: 97.7, unit: 'м.п', price: 50, hidden: true, done: true, note: '' },
    { id: 310, est_id: 3, pos: 10, name: 'грунтовка стен перед шпаклёвкой', qty: 146.4, unit: 'м²', price: 25, hidden: true, done: true, note: '' },
    { id: 311, est_id: 3, pos: 11, name: 'грунтовка откосов перед шпаклёвкой', qty: 97.7, unit: 'м.п', price: 25, hidden: true, done: true, note: '' },
    { id: 312, est_id: 3, pos: 12, name: 'шпаклёвка стен под стеклохолст', qty: 140.5, unit: 'м²', price: 140, hidden: true, done: true, note: '' },
    { id: 313, est_id: 3, pos: 13, name: 'шпаклёвка откосов под стеклохолст', qty: 97.7, unit: 'м.п', price: 140, hidden: true, done: true, note: '' },
    { id: 314, est_id: 3, pos: 14, name: 'шлифовка стен под стеклохолст', qty: 140.5, unit: 'м²', price: 50, hidden: true, done: true, note: '' },
    { id: 315, est_id: 3, pos: 15, name: 'шлифовка откосов под стеклохолст', qty: 97.7, unit: 'м.п', price: 50, hidden: true, done: true, note: '' },
    { id: 316, est_id: 3, pos: 16, name: 'грунтовка стен перед стеклохолстом', qty: 75, unit: 'м²', price: 25, hidden: true, done: true, note: '' },
    { id: 317, est_id: 3, pos: 17, name: 'грунтовка откосов перед стеклохолстом', qty: 79.7, unit: 'м.п', price: 25, hidden: true, done: true, note: '' },
    { id: 318, est_id: 3, pos: 18, name: 'поклейка стеклохолста на стены', qty: 75, unit: 'м²', price: 120, hidden: true, done: true, note: '' },
    { id: 319, est_id: 3, pos: 19, name: 'поклейка стеклохолста на откосы', qty: 79.7, unit: 'м.п', price: 120, hidden: true, done: true, note: '' },
    { id: 320, est_id: 3, pos: 20, name: 'шпаклёвка стен под покраску по стеклохолсту', qty: 75, unit: 'м²', price: 200, hidden: true, done: true, note: '' },
    { id: 321, est_id: 3, pos: 21, name: 'шпаклёвка откосов под покраску', qty: 79.7, unit: 'м.п', price: 200, hidden: true, done: true, note: '' },
    { id: 322, est_id: 3, pos: 22, name: 'шлифовка стен под покраску', qty: 75, unit: 'м²', price: 50, hidden: true, done: true, note: '' },
    { id: 323, est_id: 3, pos: 23, name: 'шлифовка откосов под покраску', qty: 79.7, unit: 'м.п', price: 50, hidden: true, done: true, note: '' },
    { id: 324, est_id: 3, pos: 24, name: 'грунтовка гипсовых панелей перед монтажом', qty: 5.9, unit: 'м²', price: 25, hidden: true, done: true, note: '' },
    { id: 325, est_id: 3, pos: 25, name: 'поклейка гипсовых панелей', qty: 5.9, unit: 'м²', price: 700, hidden: false, done: true, note: '' },
    { id: 326, est_id: 3, pos: 26, name: 'блок для розеток на гипсовых панелях', qty: 1, unit: 'шт', price: 500, hidden: true, done: true, note: '' },
    { id: 327, est_id: 3, pos: 27, name: 'шпаклёвка и шлифовка стыков гипсовых панелей', qty: 5.9, unit: 'м²', price: 500, hidden: true, done: true, note: '' },
    { id: 328, est_id: 3, pos: 28, name: 'грунтовка гипсовых панелей под покраску', qty: 5.9, unit: 'м²', price: 75, hidden: true, done: true, note: '' },
    { id: 329, est_id: 3, pos: 29, name: 'грунтовка стен под покраску', qty: 129.3, unit: 'м²', price: 25, hidden: true, done: true, note: '' },
    { id: 330, est_id: 3, pos: 30, name: 'грунтовка откосов под покраску', qty: 81.7, unit: 'м.п', price: 25, hidden: true, done: true, note: '' },
    { id: 331, est_id: 3, pos: 31, name: 'нанесение грунт-краски на гипсовые панели', qty: 5.9, unit: 'м²', price: 180, hidden: true, done: true, note: '' },
    { id: 332, est_id: 3, pos: 32, name: 'нанесение грунт-краски на стены', qty: 129.3, unit: 'м²', price: 60, hidden: true, done: true, note: '' },
    { id: 333, est_id: 3, pos: 33, name: 'нанесение грунт-краски на откосы', qty: 81.7, unit: 'м.п', price: 60, hidden: true, done: true, note: '' },
    { id: 334, est_id: 3, pos: 34, name: 'акрил на углы и разделение цветов', qty: 16.2, unit: 'м.п', price: 170, hidden: true, done: true, note: '' },
    { id: 335, est_id: 3, pos: 35, name: 'покраска гипсовых панелей безвоздушно', qty: 5.9, unit: 'м²', price: 360, hidden: false, done: true, note: '' },
    { id: 336, est_id: 3, pos: 36, name: 'покраска стен безвоздушным методом', qty: 129.3, unit: 'м²', price: 120, hidden: false, done: false, note: 'В работе' },
    { id: 337, est_id: 3, pos: 37, name: 'покраска откосов безвоздушным методом', qty: 81.7, unit: 'м.п', price: 120, hidden: false, done: false, note: 'Следующий этап' },
    { id: 338, est_id: 3, pos: 38, name: 'армировка примыкания балконного остекления', qty: 5.6, unit: 'м.п', price: 250, hidden: true, done: true, note: '' },
  ],
  2: [
    { id: 201, est_id: 2, pos: 1, name: 'шлифовка стен перед шпаклёвкой', qty: 80, unit: 'м²', price: 60, hidden: true, done: false, note: '' },
    { id: 202, est_id: 2, pos: 2, name: 'шпаклёвка стен под покраску', qty: 80, unit: 'м²', price: 180, hidden: true, done: false, note: '' },
    { id: 203, est_id: 2, pos: 3, name: 'покраска стен валиком', qty: 80, unit: 'м²', price: 120, hidden: false, done: false, note: '' },
  ],
  1: [
    { id: 101, est_id: 1, pos: 1, name: 'подготовка фасада', qty: 140, unit: 'м²', price: 300, hidden: true, done: true, note: 'Демо' },
    { id: 102, est_id: 1, pos: 2, name: 'покраска фасада', qty: 150, unit: 'м²', price: 350, hidden: false, done: true, note: 'Демо' },
  ],
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

  /* ============ v0.6: герой-сценарий, диалог, комнаты, Google Sheets ============ */

  /** demo-диктовка: имитирует транскрипт и позиции с ценами из demo-прайса. */
  async estimateVoice(_id, _audio, _mime) {
    await wait(1400);
    const transcript = 'кухня, стены, штукатурка, примерно сорок квадратов, грунтовка сорок, демонтаж перегородки две тысячи';
    const lines = [
      { name: 'штукатурка стен', qty: 40, unit: 'м²', price: 260, sum: 10400 },
      { name: 'грунтовка стен', qty: 40, unit: 'м²', price: 25, sum: 1000 },
      { name: 'демонтаж перегородки', qty: 1, unit: '', price: 2475, sum: 2475 },
    ];
    return { transcript, lines, missing: [] };
  },
  async estimateAct(id, lineIds) {
    await wait(900);
    const list = estLines[id] || [];
    const picked = lineIds ? list.filter((l) => lineIds.includes(l.id) && !l.done) : list.filter((l) => !l.done);
    if (!picked.length) throw new Error('закрывать нечего');
    const obj = objects.find((o) => o.id === (estimates.find((x) => x.id === id) || {}).object_id) || objects[0];
    const actNo = Math.max(...acts.filter((a) => a.object_id === obj.id).map((a) => a.act_no), 0) + 1;
    const brief = {
      id: Math.max(...acts.map((a) => a.id)) + 1, act_no: actNo, object_id: obj.id,
      object_name: obj.name, customer: obj.customer, total: 0, paid: 0, photos: 0, date: new Date().toISOString(),
    };
    let total = 0;
    for (const l of picked) {
      total += l.qty * l.price;
      l.done = true;
    }
    brief.total = total;
    linesByAct[brief.id] = picked.map((l, i) => ({ pos: i + 1, name: l.name, qty: l.qty, unit: l.unit, price: l.price, sum: l.qty * l.price }));
    acts.unshift(brief);
    return { act: brief, closed: picked.length, act_lines: linesByAct[brief.id], xlsx_url: `/api/acts/${brief.id}/xlsx` };
  },
  async priceSuggest(q) {
    await wait(120);
    const query = (q || '').toLowerCase();
    return { items: catalog.filter((c) => c.name.includes(query)).slice(0, 8) };
  },
  async comments(id) {
    await wait(240);
    return { items: (commentsByEst[id] = commentsByEst[id] || []).slice() };
  },
  async addComment(id, text) {
    await wait(320);
    const c = { id: Date.now(), est_id: id, author: 'master', text, created_at: new Date().toISOString() };
    (commentsByEst[id] = commentsByEst[id] || []).push(c);
    return c;
  },
  async linePhoto(_id, _lineId, _file, _caption) {
    await wait(600);
    return { saved: true, photos: [] };
  },
  async roomPresets() {
    await wait(200);
    return { items: [
      { key: 'room', name: 'Комната', hint: 'потолок, стены, пол — полный цикл', lines: [] },
      { key: 'kitchen', name: 'Кухня', hint: 'фартук, стены под плитку/покраску', lines: [] },
      { key: 'bath', name: 'Ванная', hint: 'гидроизоляция, плитка, сантехника', lines: [] },
      { key: 'wc', name: 'Туалет', hint: 'маленькое помещение — плитка и сантехника', lines: [] },
      { key: 'hall', name: 'Коридор / прихожая', hint: 'стены + пол, часто ламинат', lines: [] },
      { key: 'balcony', name: 'Балкон / лоджия', hint: 'утепление, мелкий цикл', lines: [] },
    ] };
  },
  async roomApply(estId, preset, area, height = 2.7, perimeter = 0) {
    await wait(700);
    const perim = perimeter || 4 * Math.sqrt(area);
    const norm = {
      room: ['штукатурка стен|м²|P*h|260', 'грунтовка стен|м²|P*h|25', 'шпаклёвка потолка в 2 слоя|м²|S|120', 'покраска потолка|м²|S|50', 'стяжка пола|м²|S|300', 'покраска стен|м²|P*h|90'],
      kitchen: ['штукатурка стен|м²|P*h|260', 'плитка на пол|м²|S|400', 'стяжка пола|м²|S|300'],
      bath: ['гидроизоляция пола|м²|S|120', 'плитка на пол|м²|S|400', 'плитка на стены|м²|P*h|450'],
      wc: ['гидроизоляция пола|м²|S|120', 'плитка на стены|м²|P*h|450'],
      hall: ['штукатурка стен|м²|P*h|260', 'ламинат, укладка|м²|S|180'],
      balcony: ['утепление стен|м²|P*h|150', 'гипсокартон, монтаж|м²|P*h|210'],
    };
    const src = norm[preset] || norm.room;
    const hiddenRe = /грунт|шпакл|шлиф|армир|штроб|гидроизол|гипсокартон|утепл|стяжк/;
    const added = src.map((row) => {
      const [name, unit, srcc, priceS] = row.split('|');
      let qty;
      if (srcc === 'S') qty = area;
      else if (srcc === 'P*h') qty = perim * height;
      else qty = 1;
      qty = Math.round(qty * 100) / 100;
      return {
        id: ++lineSeq, est_id: estId, pos: 0, name, qty, unit,
        price: parseFloat(priceS), sum: qty * parseFloat(priceS),
        hidden: hiddenRe.test(name), done: false, note: '',
      };
    });
    const list = (estLines[estId] = estLines[estId] || []);
    for (const l of added) { l.pos = list.length + 1; list.push(l); }
    return { added: added.length, missing_prices: 0, estimate: await mock.estimate(estId).then((r) => r.estimate), lines: added };
  },
  async priceSource() { await wait(150); return { connected: false }; },
  async priceSync() { await wait(1100); return { synced: 82, total: catalog.length + 82, duration_ms: 1050 }; },
});

/** диалоги смет (demo) — отдельно, чтобы Object.assign выше остался читаемым */
const commentsByEst = { 3: [
  { id: 1, est_id: 3, author: 'client', text: 'Почему подготовка стоит почти половину сметы?', created_at: daysAgo(2, 10) },
  { id: 2, est_id: 3, author: 'master', text: 'Грунт, шпаклёвка в 2 слоя и шлифовка — без них покраска облезет через год. Фото этапов на странице.', created_at: daysAgo(2, 12) },
] };
