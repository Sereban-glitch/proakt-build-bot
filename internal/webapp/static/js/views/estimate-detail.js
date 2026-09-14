/**
 * views/estimate-detail.js — конструктор сметы (v0.5), сердце продукта.
 *
 * Что здесь живёт:
 *  • hero: итог с коэффициентом, кольцо прогресса «закрыто актами»;
 *  • сегмент коэффициента сложности (меняет цены всей сметы мгновенно);
 *  • чипы-фильтры: все / чистовые / скрытые;
 *  • позиции: done-переключение (закрыто актом), ↑↓, правка в листе, удаление;
 *  • добавление тремя способами: строка-парсер («шпаклёвка 45 140»),
 *    из прайса (поиск), из шаблона работ (одним тапом весь техцикл);
 *  • «Поделиться» — публичная ссылка заказчику, «XLSX» — файл в формате
 *    таблицы мастера.
 * Правки включают closingConfirmation — случайное закрытие Telegram не съест
 * черновик (TMA 2026).
 */

import { api } from '../services/api.js';
import { money, moneyShort } from '../services/format.js';
import { Card, Empty, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { Segmented, Ring, Chip, Stepper, flash } from '../components/ui/adv.js';
import { icon } from '../components/ui/icon.js';
import { toast, openSheet, Field, confirmSheet } from '../components/ui/feedback.js';
import { haptics, closingConfirmation, shareURL, openLink, disableVerticalSwipes } from '../services/tg.js';

const STATUS_LABEL = { draft: 'черновик', sent: 'отправлена', approved: 'согласована', done: 'закрыта' };
const COEFFS = [
  { value: 1, label: '×1.0' },
  { value: 1.1, label: '×1.1' },
  { value: 1.2, label: '×1.2' },
  { value: 1.3, label: '×1.3' },
  { value: 1.5, label: '×1.5' },
  { value: 2, label: '×2.0' },
];

/** guessHiddenMime — совместимое имя записи для MediaRecorder (v0.6). */
function pickRecorderMime() {
  for (const m of ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4', 'audio/ogg;codecs=opus']) {
    if (window.MediaRecorder && MediaRecorder.isTypeSupported && MediaRecorder.isTypeSupported(m)) return m;
  }
  return '';
}

export function view({ root, params }) {
  const estId = Number(params.id);
  let brief = null;
  let lines = [];
  let filter = 'all'; // all | visible | hidden
  let priceCache = []; // прайс для подстановки цен
  let dirty = false;   // несохранённых правок нет, пока тихо

  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Смета', subtitle: 'загрузка…', back: () => { history.back(); } }));
  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(4, 'card-sk'));
  el.appendChild(content);
  root.appendChild(el);
  disableVerticalSwipes();

  load().catch((e) => {
    content.replaceChildren();
    content.appendChild(Empty({ iconName: 'alert', title: 'Смета не открылась', sub: e.message }));
  });

  async function load() {
    const [det, price] = await Promise.all([api.estimate(estId), api.price().catch(() => ({ items: [] }))]);
    brief = det.estimate;
    lines = det.lines;
    priceCache = price.items || [];
    render();
  }

  function markDirty() {
    if (!dirty) {
      dirty = true;
      closingConfirmation(true);
    }
  }

  function totalsOf(list) {
    let total = 0, visible = 0, hidden = 0, done = 0;
    for (const l of list) {
      const s = l.qty * l.price * brief.coeff;
      total += s;
      if (l.hidden) hidden += s; else visible += s;
      if (l.done) done += s;
    }
    return { total, visible, hidden, done };
  }

  // --- полный рендер экрана ---------------------------------------------------
  function render() {
    content.replaceChildren();
    const t = totalsOf(lines);

    el.querySelector('header .subtitle').textContent = brief.title;

    // --- hero: кольцо прогресса + деньги ---
    const hero = document.createElement('div');
    hero.className = 'est-hero section';
    const ring = Ring({
      value: t.total > 0 ? t.done / t.total : 0,
      size: 84,
      stroke: 8,
      label: `${Math.round(t.total > 0 ? (t.done / t.total) * 100 : 0)}%`,
      sub: 'закрыто',
    });
    hero.appendChild(ring.el);

    const hv = document.createElement('div');
    hv.className = 'est-hv';
    const big = document.createElement('div');
    big.className = 'big num';
    big.textContent = money(t.total);
    const sub1 = document.createElement('div');
    sub1.className = 'sub';
    sub1.textContent = `${brief.object_name} · ${STATUS_LABEL[brief.status] || brief.status}`;
    const sub2 = document.createElement('div');
    sub2.className = 'sub num';
    sub2.textContent = `чистовые ${moneyShort(t.visible)} · скрытые ${moneyShort(t.hidden)}`;
    hv.append(big, sub1, sub2);
    hero.appendChild(hv);
    content.appendChild(hero);

    // --- панель управления: коэффициент + статус + действия ---
    const ctrl = Card({ className: 'section' });

    const coeffRow = document.createElement('div');
    coeffRow.className = 'coeff-row';
    const coeffIc = document.createElement('span');
    coeffIc.className = 'coeff-ic';
    coeffIc.appendChild(icon('coeff', 20));
    const coeffText = document.createElement('div');
    coeffText.style.cssText = 'flex:1;min-width:0;';
    const coeffTitle = document.createElement('div');
    coeffTitle.style.cssText = 'font-weight:700;font-size:13.5px;';
    coeffTitle.textContent = 'Коэффициент сложности';
    const coeffNote = document.createElement('div');
    coeffNote.className = 'coeff-note';
    coeffNote.textContent = 'потолки от 280 см, лестницы, сжатые сроки — цена умножается на всю смету';
    coeffText.append(coeffTitle, coeffNote);
    const seg = Segmented({
      options: COEFFS,
      value: brief.coeff,
      small: true,
      onChange: async (v) => {
        if (v === brief.coeff) return;
        markDirty();
        haptics.medium();
        try {
          brief = await api.patchEstimate(estId, { coeff: v });
          flash(big);
          render();
          toast(`Коэффициент ×${v} — все суммы пересчитаны`, { icon: '🧮' });
        } catch (e) {
          toast(e.message, { tone: 'danger' });
        }
      },
    });
    coeffRow.append(coeffIc, coeffText);
    content.appendChild(ctrl);
    ctrl.appendChild(coeffRow);
    ctrl.appendChild(seg.el);

    // действия: поделиться / xlsx / статус
    const actions = document.createElement('div');
    actions.style.cssText = 'display:flex;gap:8px;padding:2px 14px 14px;flex-wrap:wrap;';
    const mkBtn = (label, iconName, onClick, variant = 'secondary') => {
      const b = document.createElement('button');
      b.className = `btn btn-${variant}`;
      b.style.cssText = 'flex:1;min-width:120px;font-size:13.5px;';
      b.appendChild(icon(iconName, 17));
      const sp = document.createElement('span');
      sp.textContent = label;
      b.appendChild(sp);
      b.addEventListener('click', () => { haptics.tap(); onClick(); });
      return b;
    };
    actions.appendChild(mkBtn('Заказчику', 'share', doShare));
    actions.appendChild(mkBtn('Акт недели', 'acts', actWeekSheet));
    actions.appendChild(mkBtn('Диалог', 'message', commentsSheet));
    actions.appendChild(mkBtn('Excel', 'download', () => {
      const url = api.estimateXlsxURL(estId);
      if (String(url).startsWith('/api/')) {
        // живой режим: ссылка с подписью initData не пройдёт — открываем
        // через fetch → blob (заголовок ставится сервисом api.js? нет, прямой
        // <a> без заголовка → 401). Проще: window.open в том же контексте
        // не несёт заголовок. Поэтому качаем с заголовком в blob.
        fetchBlob(url).then((blobUrl) => {
          const a = document.createElement('a');
          a.href = blobUrl;
          a.download = `smeta_${estId}.xlsx`;
          a.click();
          setTimeout(() => URL.revokeObjectURL(blobUrl), 4000);
        }).catch((e) => toast(e.message, { tone: 'danger' }));
      } else {
        toast('В demo XLSX не генерится — только на сервере', { icon: '📊' });
      }
    }));
    actions.appendChild(mkBtn('Статус: ' + (STATUS_LABEL[brief.status] || brief.status), 'edit', statusSheet));
    ctrl.appendChild(actions);
    content.appendChild(ctrl);

    // --- позиции: фильтры + список ---
    const visibleList = lines.filter((l) => filter === 'all' || (filter === 'hidden' ? l.hidden : !l.hidden));
    const listCard = Card({ className: 'section' });
    const headRow = document.createElement('div');
    headRow.style.cssText = 'display:flex;align-items:center;gap:8px;padding:12px 14px 4px;flex-wrap:wrap;';
    const chips = [
      Chip({ label: `Все · ${lines.length}`, active: filter === 'all', onToggle: () => { filter = 'all'; render(); } }),
      Chip({ label: `Чистовые · ${lines.filter((l) => !l.hidden).length}`, active: filter === 'visible', tone: '', onToggle: () => { filter = 'visible'; render(); } }),
      Chip({ label: `Скрытые · ${lines.filter((l) => l.hidden).length}`, active: filter === 'hidden', tone: 'warn', onToggle: () => { filter = 'hidden'; render(); } }),
    ];
    for (const c of chips) headRow.appendChild(c.el);
    listCard.appendChild(headRow);

    if (!visibleList.length) {
      const em = Empty({
        iconName: 'estimate',
        title: lines.length ? 'В этом фильтре пусто' : 'Позиций пока нет',
        sub: lines.length ? 'Переключи фильтр' : 'Строкой «шпаклёвка 45 140», из прайса или шаблоном — на выбор',
      });
      em.style.padding = '18px 14px 22px';
      listCard.appendChild(em);
    } else {
      const listEl = document.createElement('div');
      for (const l of visibleList) listEl.appendChild(lineRow(l));
      listCard.appendChild(listEl);
    }

    // мини-итог по фильтру
    if (lines.length) {
      const ft = totalsOf(visibleList);
      const foot = document.createElement('div');
      foot.className = 'coeff-row';
      foot.style.borderTop = '1px solid var(--border)';
      const lbl = document.createElement('span');
      lbl.style.cssText = 'font-size:12.5px;color:var(--hint);flex:1;';
      lbl.textContent = filter === 'all' ? 'Всего по смете' : (filter === 'hidden' ? 'Скрытая подготовка' : 'Чистовые работы');
      const val = document.createElement('b');
      val.className = 'num';
      val.style.fontSize = '15px';
      val.textContent = money(ft.total);
      foot.append(lbl, val);
      listCard.appendChild(foot);
    }
    content.appendChild(listCard);

    // --- добавление: диктовка / парсер / прайс / шаблон / комната ---
    const addCard = Card({ className: 'section' });
    const addTitle = document.createElement('div');
    addTitle.className = 'card-title';
    addTitle.textContent = 'Добавить позиции';
    addCard.appendChild(addTitle);

    // v0.6: ДИКТОВКА — герой-сценарий. Крупная кнопка у микрофона:
    // «кухня, стены, штукатурка, примерно сорок квадратов» — строка готова.
    const micRow = document.createElement('button');
    micRow.className = 'btn btn-primary btn-block mic-btn';
    micRow.appendChild(icon('mic', 22));
    const micLbl = document.createElement('span');
    micLbl.textContent = 'Диктовать на объекте';
    micRow.appendChild(micLbl);
    micRow.addEventListener('click', () => { haptics.medium(); voiceSheet(); });
    const micWrap = document.createElement('div');
    micWrap.style.cssText = 'padding:0 14px 10px;';
    micWrap.appendChild(micRow);
    addCard.appendChild(micWrap);

    const addWrap = document.createElement('div');
    addWrap.style.cssText = 'display:flex;flex-direction:column;gap:8px;padding:0 14px 14px;';
    const parseF = Field({ label: 'Строкой — как в чате бота', placeholder: 'шпаклёвка 45 м² 140 или шлифовка 45 м' });
    addWrap.appendChild(parseF.el);
    // v0.6: умные подсказки из прайса — «меньше печатать»
    const suggestBox = document.createElement('div');
    suggestBox.className = 'suggest-box';
    suggestBox.style.display = 'none';
    addWrap.appendChild(suggestBox);
    const parseHint = document.createElement('div');
    parseHint.className = 'parse-box';
    parseHint.style.display = 'none';
    addWrap.appendChild(parseHint);

    let parseTimer = null;
    parseF.input.addEventListener('input', () => {
      clearTimeout(parseTimer);
      const text = parseF.input.value.trim();
      // подсказки: слова ≥2 символов без цифр → топ-5 совпадений прайса
      if (text.length >= 2 && !/\d/.test(text)) {
        const q = text.toLowerCase();
        const hits = priceCache.filter((p) => p.name.includes(q)).slice(0, 5);
        if (hits.length) {
          suggestBox.style.display = 'block';
          suggestBox.replaceChildren(...hits.map((p) => {
            const b = document.createElement('button');
            b.type = 'button';
            b.className = 'suggest-item';
            const n1 = document.createElement('span');
            n1.textContent = p.name;
            const n2 = document.createElement('b');
            n2.className = 'num';
            n2.textContent = `${trimNum(p.price)}${p.unit ? '/' + p.unit : ''}`;
            b.append(n1, n2);
            b.addEventListener('click', () => {
              haptics.select();
              parseF.input.value = p.name + ' ';
              suggestBox.style.display = 'none';
              parseF.input.focus();
            });
            return b;
          }));
        } else {
          suggestBox.style.display = 'none';
        }
      } else {
        suggestBox.style.display = 'none';
      }
      if (text.length < 3) { parseHint.style.display = 'none'; return; }
      parseTimer = setTimeout(async () => {
        try {
          const res = await api.parseLine(text);
          parseHint.style.display = 'block';
          if (res.suspicious) {
            parseHint.innerHTML = `<div class="parse-preview">🤔 ${res.hint || 'Похоже, несколько позиций без цен'}</div>`;
            return;
          }
          if (!res.lines.length) { parseHint.style.display = 'none'; return; }
          parseHint.innerHTML = res.lines.map((l) =>
            `<div class="parse-preview"><b>${escapeHtml(l.name)}</b> — ${l.qty} ${l.unit || ''} × ${l.price || 'цена?'} = ${money(l.sum || l.qty * l.price)}${l.price ? '' : ' <span style="color:var(--warn)">цена возьмётся из прайса</span>'}</div>`).join('');
        } catch { parseHint.style.display = 'none'; }
      }, 260);
    });
    parseF.input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        addParsed(parseF.input.value.trim());
      }
    });
    const addRow = document.createElement('button');
    addRow.className = 'btn btn-primary btn-block';
    addRow.textContent = 'Добавить строкой';
    addRow.addEventListener('click', () => addParsed(parseF.input.value.trim()));
    addWrap.appendChild(addRow);

    const secondRow = document.createElement('div');
    secondRow.style.cssText = 'display:flex;gap:8px;';
    const fromPrice = document.createElement('button');
    fromPrice.className = 'btn btn-secondary';
    fromPrice.style.cssText = 'flex:1;';
    fromPrice.appendChild(icon('search', 16));
    const sp1 = document.createElement('span');
    sp1.textContent = 'Из прайса';
    fromPrice.appendChild(sp1);
    fromPrice.addEventListener('click', () => { haptics.tap(); pricePicker(); });
    const fromTpl = document.createElement('button');
    fromTpl.className = 'btn btn-secondary';
    fromTpl.style.cssText = 'flex:1;';
    fromTpl.appendChild(icon('template', 16));
    const sp2 = document.createElement('span');
    sp2.textContent = 'Шаблон';
    fromTpl.appendChild(sp2);
    fromTpl.addEventListener('click', () => { haptics.tap(); templatePicker(); });
    const fromRoom = document.createElement('button');
    fromRoom.className = 'btn btn-secondary';
    fromRoom.style.cssText = 'flex:1;';
    fromRoom.appendChild(icon('room', 16));
    const sp3 = document.createElement('span');
    sp3.textContent = 'Комната';
    fromRoom.appendChild(sp3);
    fromRoom.addEventListener('click', () => { haptics.tap(); roomSheet(); });
    secondRow.append(fromPrice, fromTpl, fromRoom);
    addWrap.appendChild(secondRow);
    addCard.appendChild(addWrap);
    content.appendChild(addCard);

    // --- удалить смету ---
    const delWrap = document.createElement('div');
    delWrap.style.cssText = 'padding:0 0 12px;display:flex;justify-content:center;';
    const delBtn = document.createElement('button');
    delBtn.className = 'btn btn-ghost';
    delBtn.style.cssText = 'color:var(--danger);font-size:13px;';
    delBtn.textContent = 'Удалить смету целиком';
    delBtn.addEventListener('click', async () => {
      const ok = await confirmSheet({
        title: 'Удалить смету?',
        text: `«${brief.title}» на ${money(t.total)} и все позиции уйдут безвозвратно. Акты и оплаты объекта не трогаем.`,
        confirmLabel: 'Да, удалить смету',
      });
      if (!ok) return;
      try {
        await api.deleteEstimate(estId);
        haptics.success();
        toast('Смета удалена', { icon: '🗑' });
        location.hash = '#/estimates';
      } catch (e) {
        toast(e.message, { tone: 'danger' });
      }
    });
    delWrap.appendChild(delBtn);
    content.appendChild(delWrap);
  }

  // --- строка позиции ----------------------------------------------------------
  function lineRow(l) {
    const row = document.createElement('div');
    row.className = `est-line${l.hidden ? ' hiddenw' : ''}${l.done ? ' done' : ''}`;
    row.dataset.lineId = l.id;

    const name = document.createElement('div');
    name.className = 'ln-name';
    name.textContent = l.name;
    row.appendChild(name);

    const sum = document.createElement('div');
    sum.className = 'ln-sum';
    const base = l.qty * l.price;
    sum.textContent = money(base * brief.coeff);
    if (brief.coeff !== 1) {
      const baseEl = document.createElement('div');
      baseEl.style.cssText = 'font-size:10.5px;font-weight:500;color:var(--hint);text-align:right;';
      baseEl.textContent = money(base) + ' × ' + brief.coeff;
      sum.appendChild(baseEl);
    }
    row.appendChild(sum);

    const meta = document.createElement('div');
    meta.className = 'ln-meta';
    const bits = [];
    if (l.qty > 0) bits.push(`${trimNum(l.qty)} ${l.unit || ''} × ${trimNum(l.price)}`);
    else bits.push(`${trimNum(l.price)} за всё`);
    if (l.hidden) bits.push('скрытая подготовка');
    if (l.done) bits.push('✓ закрыто актом');
    if (l.note) bits.push(l.note);
    meta.textContent = bits.join(' · ');
    row.appendChild(meta);

    // v0.6: редактирование «на пальцах» — объём стрелками прямо в строке
    // (мастер поправит «примерно 40» на «42» одним тапом, не открывая лист)
    if (l.qty > 0) {
      const quick = document.createElement('div');
      quick.className = 'qty-quick';
      const mkQ = (label, delta) => {
        const b = document.createElement('button');
        b.type = 'button';
        b.className = 'qty-btn';
        b.textContent = label;
        b.addEventListener('click', async (e) => {
          e.stopPropagation();
          const next = Math.max(0, Math.round((l.qty + delta) * 100) / 100);
          if (next === l.qty) return;
          markDirty();
          haptics.tap();
          try {
            const updated = await api.patchEstLine(estId, l.id, { qty: next });
            Object.assign(l, updated && updated.qty !== undefined ? updated : { qty: next, sum: next * l.price });
            render();
          } catch (err) {
            toast(err.message, { tone: 'danger' });
          }
        });
        return b;
      };
      const step = l.unit ? 1 : 1;
      quick.appendChild(mkQ(`−${step} ${l.unit || ''}`.trim(), -step));
      quick.appendChild(mkQ(`+${step} ${l.unit || ''}`.trim(), +step));
      row.appendChild(quick);
    }

    const actions = document.createElement('div');
    actions.className = 'ln-actions';
    const mkIcon = (iconName, aria, cls, onClick) => {
      const b = document.createElement('button');
      b.className = `icon-mini ${cls || ''}`;
      b.setAttribute('aria-label', aria);
      b.title = aria;
      b.appendChild(icon(iconName, 16));
      b.addEventListener('click', (e) => { e.stopPropagation(); onClick(b); });
      return b;
    };
    actions.appendChild(mkIcon('image', 'Фото строки', '', () => linePhotoSheet(l)));
    actions.appendChild(mkIcon('done', 'Закрыто актом', l.done ? 'done-on' : '', async (b) => {
      markDirty();
      try {
        const updated = await api.patchEstLine(estId, l.id, { done: !l.done });
        Object.assign(l, updated.done !== undefined ? updated : { done: !l.done });
        if (updated && updated.qty !== undefined) Object.assign(l, updated);
        haptics.success();
        render();
      } catch (e) {
        toast(e.message, { tone: 'danger' });
      }
    }));
    actions.appendChild(mkIcon('up', 'Выше', '', async () => {
      try {
        await api.moveEstLine(estId, l.id, 'up');
        haptics.tap();
        await reloadLines();
        render();
      } catch { /* край списка */ }
    }));
    actions.appendChild(mkIcon('down', 'Ниже', '', async () => {
      try {
        await api.moveEstLine(estId, l.id, 'down');
        haptics.tap();
        await reloadLines();
        render();
      } catch { /* край списка */ }
    }));
    actions.appendChild(mkIcon('edit', 'Правка', '', () => editLineSheet(l)));
    actions.appendChild(mkIcon('eyeOff', l.hidden ? 'Сделать видимой' : 'Скрытая работа', '', async () => {
      markDirty();
      try {
        const updated = await api.patchEstLine(estId, l.id, { hidden: !l.hidden });
        if (updated && updated.qty !== undefined) Object.assign(l, updated); else l.hidden = !l.hidden;
        haptics.medium();
        render();
        toast(l.hidden ? 'Скрытая подготовка — заказчик увидит пояснение' : 'Позиция в чистовых работах', { icon: l.hidden ? ' ▪' : '👁' });
      } catch (e) {
        toast(e.message, { tone: 'danger' });
      }
    }));
    actions.appendChild(mkIcon('trash', 'Удалить позицию', 'danger', async () => {
      const ok = await confirmSheet({
        title: 'Убрать позицию?',
        text: `«${l.name}» на ${money(l.qty * l.price * brief.coeff)} уйдёт из сметы.`,
        confirmLabel: 'Убрать',
      });
      if (!ok) return;
      try {
        await api.deleteEstLine(estId, l.id);
        lines = lines.filter((x) => x.id !== l.id);
        haptics.success();
        render();
      } catch (e) {
        toast(e.message, { tone: 'danger' });
      }
    }));
    row.appendChild(actions);
    return row;
  }

  async function reloadLines() {
    const det = await api.estimate(estId);
    lines = det.lines;
  }

  // --- добавление: парсер / прайс / шаблон ---------------------------------------
  async function addParsed(text) {
    if (text.length < 3) {
      toast('Напиши позицию: «шпаклёвка 45 140»', { tone: 'danger', icon: '✍️' });
      return;
    }
    try {
      const res = await api.parseLine(text);
      if (res.suspicious) {
        toast(res.hint || 'Разбей на отдельные строки', { tone: 'danger', icon: '🤔' });
        return;
      }
      if (!res.lines.length) {
        toast('Не понял строку — название + числа', { tone: 'danger' });
        return;
      }
      markDirty();
      const prepared = res.lines.map((l) => {
        // цена из прайса, если строка пришла без цены
        let price = l.price;
        if (!price && priceCache.length) {
          const name = l.name.toLowerCase();
          const hit = priceCache.find((p) => p.name === name)
            || priceCache.find((p) => name.includes(p.name) || p.name.includes(name) && p.name.length >= 4);
          if (hit) price = hit.price;
        }
        return { name: l.name, qty: l.qty || 1, unit: l.unit || '', price: price || 0, hidden: guessHidden(l.name) };
      });
      const r = await api.addEstLinesBulk(estId, prepared);
      if (r.estimate) brief = { ...brief, ...r.estimate };
      parseF.input.value = '';
      parseHint.style.display = 'none';
      await reloadLines();
      haptics.success();
      render();
      toast(prepared.length === 1 ? 'Позиция добавлена' : `Добавлено позиций: ${prepared.length}`, { icon: '➕' });
    } catch (e) {
      toast(e.message, { tone: 'danger' });
    }
  }

  function pricePicker() {
    const search = Field({ label: 'Поиск по прайсу', placeholder: 'шпаклёвка…' });
    const results = document.createElement('div');
    results.style.cssText = 'max-height:46vh;overflow:auto;display:flex;flex-direction:column;gap:2px;margin-top:10px;';
    const renderList = (q) => {
      const query = (q || '').toLowerCase();
      const items = priceCache.filter((p) => !query || p.name.includes(query)).slice(0, 30);
      results.replaceChildren();
      if (!priceCache.length) {
        results.innerHTML = '<div class="muted" style="padding:10px;font-size:13px;">Прайс пуст — заполните его во вкладке «Прайс»</div>';
        return;
      }
      for (const p of items) {
        const row = document.createElement('button');
        row.type = 'button';
        row.className = 'cell';
        row.style.cssText = 'background:none;border:0;width:100%;font:inherit;text-align:left;border-bottom:1px solid var(--border);';
        const nm = document.createElement('div');
        nm.className = 'cell-title';
        nm.textContent = p.name;
        const rt = document.createElement('div');
        rt.className = 'cell-value num';
        rt.textContent = `${trimNum(p.price)} ${p.unit ? '/' + p.unit : ''}`;
        row.append(nm, rt);
        row.addEventListener('click', async () => {
          markDirty();
          try {
            await api.addEstLine(estId, {
              name: p.name, qty: 1, unit: p.unit, price: p.price, hidden: guessHidden(p.name),
            });
            haptics.success();
            await reloadLines();
            render();
            sheet.close();
            toast(`«${p.name}» — добавлена, количество поправь в строке`, { icon: '💡' });
          } catch (e) {
            toast(e.message, { tone: 'danger' });
          }
        });
        results.appendChild(row);
      }
    };
    search.input.addEventListener('input', () => renderList(search.input.value));
    renderList('');
    const sheet = openSheet({ title: '📁 Позиция из прайса', body: [search.el, results] });
  }

  async function templatePicker() {
    const list = document.createElement('div');
    list.style.cssText = 'max-height:52vh;overflow:auto;display:flex;flex-direction:column;gap:10px;';
    list.innerHTML = '<div class="muted" style="padding:8px;font-size:13px;">Загружаю шаблоны…</div>';
    const sheet = openSheet({ title: '🧰 Шаблоны работ', body: [list] });
    try {
      const res = await api.templates();
      list.replaceChildren();
      if (!res.items.length) {
        list.innerHTML = '<div class="muted" style="padding:8px;font-size:13px;">Шаблонов ещё нет — собери первый во вкладке «Шаблоны»</div>';
        return;
      }
      for (const tpl of res.items) {
        const c = document.createElement('button');
        c.type = 'button';
        c.style.cssText = 'text-align:left;background:var(--card);border:1px solid var(--border);border-radius:var(--radius-m);padding:12px;cursor:pointer;color:inherit;font:inherit;';
        const t1 = document.createElement('div');
        t1.style.cssText = 'font-weight:700;font-size:14px;margin-bottom:4px;';
        t1.textContent = tpl.name;
        const t2 = document.createElement('div');
        t2.style.cssText = 'font-size:12px;color:var(--hint);line-height:1.5;';
        const sum = tpl.lines.reduce((a, l) => a + l.price * l.qty, 0);
        t2.textContent = `${tpl.lines.length} позиций · скрытых ${tpl.lines.filter((l) => l.hidden).length} · базовая сумма ${money(sum)} (×коэффициент сметы)`;
        c.append(t1, t2);
        c.addEventListener('click', async () => {
          markDirty();
          try {
            const r = await api.addEstLinesBulk(estId, tpl.lines.map((l) => ({
              name: l.name, qty: l.qty, unit: l.unit || '', price: l.price, hidden: Boolean(l.hidden), note: '',
            })));
            if (r.estimate) brief = { ...brief, ...r.estimate };
            haptics.success();
            await reloadLines();
            render();
            sheet.close();
            toast(`Шаблон «${tpl.name}» — вставлено ${r.added} позиций`, { icon: '⚡' });
          } catch (e) {
            toast(e.message, { tone: 'danger' });
          }
        });
        list.appendChild(c);
      }
    } catch (e) {
      list.innerHTML = `<div class="muted" style="padding:8px;">${escapeHtml(e.message)}</div>`;
    }
  }

  // --- правка позиции ---------------------------------------------------------------
  function editLineSheet(l) {
    const nameF = Field({ label: 'Название', value: l.name });
    const qtySt = Stepper({ value: l.qty, step: 1, min: 0, suffix: l.unit || '' });
    const priceF = Field({ label: 'Цена за единицу, грн', type: 'text', inputmode: 'decimal', value: String(l.price).replace('.', ',') });
    const unitF = Field({ label: 'Единица (м² / м.п / шт — или пусто)', value: l.unit });
    const noteF = Field({ label: 'Примечание', value: l.note || '' });

    const save = document.createElement('button');
    save.className = 'btn btn-primary btn-block';
    save.textContent = 'Сохранить';
    save.addEventListener('click', async () => {
      markDirty();
      try {
        const patch = {
          name: nameF.input.value.trim(),
          qty: qtySt.value,
          price: parseFloat(priceF.input.value.replace(',', '.')) || 0,
          unit: unitF.input.value.trim(),
          note: noteF.input.value.trim(),
        };
        const updated = await api.patchEstLine(estId, l.id, patch);
        Object.assign(l, updated);
        haptics.success();
        await reloadLines();
        render();
        sheet.close();
        toast('Позиция обновлена', { icon: '✏️' });
      } catch (e) {
        toast(e.message, { tone: 'danger' });
      }
    });
    const sheet = openSheet({ title: '✏️ Правка позиции', body: [nameF.el, qtySt.el, priceF.el, unitF.el, noteF.el], footer: [save] });
  }

  // --- статус сметы --------------------------------------------------------------------
  function statusSheet() {
    const box = document.createElement('div');
    box.style.cssText = 'display:flex;flex-direction:column;gap:8px;';
    for (const st of ['draft', 'sent', 'approved', 'done']) {
      const b = document.createElement('button');
      b.className = 'btn ' + (brief.status === st ? 'btn-primary' : 'btn-secondary');
      b.textContent = STATUS_LABEL[st];
      b.addEventListener('click', async () => {
        try {
          brief = { ...brief, ...await api.patchEstimate(estId, { status: st }) };
          haptics.select();
          render();
          sheet.close();
          toast('Статус: ' + STATUS_LABEL[st], { icon: '🔖' });
        } catch (e) {
          toast(e.message, { tone: 'danger' });
        }
      });
      box.appendChild(b);
    }
    const sheet = openSheet({ title: '🔖 Статус сметы', body: [box] });
  }

  // --- шеринг заказчику -------------------------------------------------------------------
  async function doShare() {
    try {
      const { url } = await api.estimateShare(estId);
      const t = totalsOf(lines);
      const res = await shareURL(url, `Смета «${brief.title}» — ${money(t.total)}. Скрытые работы с фото: ${url}`);
      if (res === 'copied') toast('Ссылка скопирована — отправь заказчику', { icon: '🔗' });
    } catch (e) {
      toast(e.message, { tone: 'danger' });
    }
  }

  // --- v0.6: ДИКТОВКА (герой-сценарий) ------------------------------------------
  // MediaRecorder → base64 → POST /voice → транскрипт + позиции → предпросмотр.
  // Одна диктовка = блокнотная строка в 10 раз быстрее, чем руками.
  function voiceSheet() {
    if (!window.MediaRecorder || !navigator.mediaDevices?.getUserMedia) {
      toast('Запись не поддерживается в этом браузере — диктуй голосом в чат бота', { icon: '🎤' });
      return;
    }
    const status = document.createElement('div');
    status.className = 'rec-status';
    const dot = document.createElement('span');
    dot.className = 'rec-dot';
    const txt = document.createElement('div');
    txt.textContent = 'Нажми «Записать» и говори как обычно:\n«кухня, стены, штукатурка, примерно сорок квадратов»';
    status.append(dot, txt);

    const timer = document.createElement('div');
    timer.className = 'rec-timer num';
    timer.textContent = '0:00';

    const startBtn = document.createElement('button');
    startBtn.className = 'btn btn-primary btn-block';
    startBtn.textContent = '🎙 Записать';
    const result = document.createElement('div');
    result.style.cssText = 'display:flex;flex-direction:column;gap:6px;';

    const sheet = openSheet({ title: '🎤 Диктовка позиций', body: [status, timer, startBtn, result] });

    let recorder = null;
    let chunks = [];
    let recMime = pickRecorderMime();
    let t0 = 0;
    let tick = null;

    const stopUI = () => {
      clearInterval(tick);
      startBtn.disabled = false;
      startBtn.textContent = '🎙 Записать заново';
      dot.classList.remove('on');
    };

    startBtn.addEventListener('click', async () => {
      if (recorder && recorder.state === 'recording') {
        recorder.stop();
        return;
      }
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        chunks = [];
        recorder = new MediaRecorder(stream, recMime ? { mimeType: recMime } : undefined);
        recorder.ondataavailable = (e) => { if (e.data.size) chunks.push(e.data); };
        recorder.onstop = async () => {
          stream.getTracks().forEach((t) => t.stop());
          stopUI();
          const blob = new Blob(chunks, { type: recMime || 'audio/webm' });
          if (blob.size < 800) {
            toast('Слишком коротко — нажми и говори пару секунд', { tone: 'danger' });
            return;
          }
          txt.textContent = 'Распознаю…';
          result.replaceChildren();
          const sk = document.createElement('div');
          sk.className = 'skeleton card-sk';
          result.appendChild(sk);
          try {
            const b64 = await blobToBase64(blob);
            const res = await api.estimateVoice(estId, b64, blob.type || 'audio/webm');
            result.replaceChildren();
            txt.textContent = 'Распознал: «' + (res.transcript || '').slice(0, 140) + '»';
            if (!res.lines?.length) {
              toast('Позиций не нашёл — попробуй ещё раз', { tone: 'danger', icon: '🤔' });
              return;
            }
            const total = res.lines.reduce((a, l) => a + (l.sum || l.qty * l.price), 0);
            for (const l of res.lines) {
              const row = document.createElement('div');
              row.className = 'parse-preview';
              row.innerHTML = `<b>${escapeHtml(l.name)}</b> — ${l.qty} ${l.unit || ''} × ${l.price || 'цена из прайса'} = ${money(l.sum || l.qty * l.price)}`;
              result.appendChild(row);
            }
            const addAll = document.createElement('button');
            addAll.className = 'btn btn-primary btn-block';
            addAll.textContent = `Добавить ${res.lines.length} позиц. · ${money(total)}`;
            addAll.addEventListener('click', async () => {
              addAll.disabled = true;
              markDirty();
              try {
                const prepared = res.lines.map((l) => ({
                  name: l.name, qty: l.qty || 1, unit: l.unit || '', price: l.price || 0, hidden: guessHidden(l.name), note: '',
                }));
                const r = await api.addEstLinesBulk(estId, prepared);
                if (r.estimate) brief = { ...brief, ...r.estimate };
                haptics.success();
                await reloadLines();
                render();
                sheet.close();
                toast(`Продиктовано: +${prepared.length} позиций`, { icon: '🎤' });
              } catch (e2) {
                addAll.disabled = false;
                toast(e2.message, { tone: 'danger' });
              }
            });
            result.appendChild(addAll);
            if (res.missing?.length) {
              const warn = document.createElement('div');
              warn.className = 'muted';
              warn.style.cssText = 'font-size:12px;color:var(--warn);';
              warn.textContent = '⚠ Нет в прайсе: ' + res.missing.join(', ');
              result.appendChild(warn);
            }
          } catch (e2) {
            result.replaceChildren();
            txt.textContent = 'Не получилось расшифровать — попробуй ещё раз или введи строкой.';
          }
        };
        recorder.start();
        t0 = Date.now();
        tick = setInterval(() => {
          const s = Math.floor((Date.now() - t0) / 1000);
          timer.textContent = `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
        }, 250);
        dot.classList.add('on');
        startBtn.textContent = '⏹ Остановить';
        haptics.medium();
      } catch {
        toast('Нет доступа к микрофону — проверь разрешения', { tone: 'danger' });
      }
    });
  }

  // --- v0.6: типовые помещения с нормами ----------------------------------------
  // «Кухня 9 м²» → потолок = S, стены = периметр×высота, грунт/шпаклёвка
  // той же площади, плинтус = периметр. Мастер подправляет объёмы тапами.
  async function roomSheet() {
    const list = document.createElement('div');
    list.innerHTML = '<div class="muted" style="padding:8px;font-size:13px;">Загружаю помещения…</div>';
    const sheet = openSheet({ title: '🏠 Типовое помещение', body: [list] });
    let presets;
    try {
      presets = (await api.roomPresets()).items;
    } catch (e) {
      list.innerHTML = `<div class="muted" style="padding:8px;">${escapeHtml(e.message)}</div>`;
      return;
    }
    list.replaceChildren();
    for (const p of presets) {
      const b = document.createElement('button');
      b.type = 'button';
      b.style.cssText = 'text-align:left;background:var(--card);border:1px solid var(--border);border-radius:var(--radius-m);padding:12px;cursor:pointer;color:inherit;font:inherit;';
      const t1 = document.createElement('div');
      t1.style.cssText = 'font-weight:700;font-size:14px;';
      t1.textContent = p.name;
      const t2 = document.createElement('div');
      t2.style.cssText = 'font-size:12px;color:var(--hint);margin-top:2px;';
      t2.textContent = `${p.hint} · ${p.lines.length} работ`;
      b.append(t1, t2);
      b.addEventListener('click', () => {
        haptics.select();
        roomSizes(p);
      });
      list.appendChild(b);
    }

    function roomSizes(preset) {
      const areaF = Field({ label: 'Площадь пола, м²', inputmode: 'decimal', placeholder: '9' });
      const heightF = Field({ label: 'Высота, м (пусто — 2.7)', inputmode: 'decimal', placeholder: '2.7' });
      const perimF = Field({ label: 'Периметр, м (пусто — посчитаю ≈4·√S)', inputmode: 'decimal', placeholder: '12' });
      const go = document.createElement('button');
      go.className = 'btn btn-primary btn-block';
      go.textContent = 'Рассчитать и вставить';
      go.addEventListener('click', async () => {
        const area = parseFloat((areaF.input.value || '').replace(',', '.'));
        if (!area || area <= 0) { areaF.input.focus(); return; }
        const height = parseFloat((heightF.input.value || '').replace(',', '.')) || 2.7;
        const perim = parseFloat((perimF.input.value || '').replace(',', '.')) || 0;
        go.disabled = true;
        markDirty();
        try {
          const res = await api.roomApply(estId, preset.key, area, height, perim);
          if (res.estimate) brief = { ...brief, ...res.estimate };
          haptics.success();
          await reloadLines();
          render();
          sheet.close();
          toast(`«${preset.name}» — вставлено ${res.added} работ${res.missing_prices ? `, без цены: ${res.missing_prices}` : ''}`, { icon: '🏠' });
        } catch (e2) {
          go.disabled = false;
          toast(e2.message, { tone: 'danger' });
        }
      });
      // переиспользуем лист: заменяем содержимое
      list.replaceChildren(areaF.el, heightF.el, perimF.el, go);
      sheet.el.querySelector('h3').textContent = `🏠 ${preset.name} — размеры`;
    }
  }

  // --- v0.6: АКТ ИЗ СМЕТЫ ЗА НЕДЕЛЮ ------------------------------------------------
  // Отметил строки, сделанные за неделю → акт с теми же ценами → XLSX.
  // Еженедельная рутина Виталика: вместо 1–2 часов Excel — минута тапов.
  function actWeekSheet() {
    const open = lines.filter((l) => !l.done && (l.qty * l.price) > 0.009);
    if (!open.length) {
      toast('Незакрытых строк с суммой нет — отметь готовые или добавь новые', { icon: '✅' });
      return;
    }
    const selected = new Map(open.map((l) => [l.id, false]));
    const list = document.createElement('div');
    list.style.cssText = 'display:flex;flex-direction:column;gap:2px;max-height:46vh;overflow:auto;';
    let sum = 0;
    const totalEl = document.createElement('div');
    const mkRow = (l) => {
      const row = document.createElement('button');
      row.type = 'button';
      row.className = 'cell pick-line';
      row.style.cssText = 'background:none;border:0;width:100%;font:inherit;text-align:left;border-bottom:1px solid var(--border);padding:10px 4px;';
      const box = document.createElement('span');
      box.className = 'pick-box';
      const nm = document.createElement('div');
      nm.style.cssText = 'flex:1;min-width:0;';
      const t = document.createElement('div');
      t.className = 'cell-title';
      t.textContent = l.name;
      const m = document.createElement('div');
      m.className = 'cell-sub';
      m.textContent = `${trimNum(l.qty)} ${l.unit || ''} × ${trimNum(l.price)}`;
      nm.append(t, m);
      const sv = document.createElement('b');
      sv.className = 'cell-value num';
      sv.textContent = money(l.qty * l.price * brief.coeff);
      row.append(box, nm, sv);
      row.addEventListener('click', () => {
        const on = !selected.get(l.id);
        selected.set(l.id, on);
        row.classList.toggle('picked', on);
        box.classList.toggle('on', on);
        sum += (on ? 1 : -1) * l.qty * l.price * brief.coeff;
        totalEl.textContent = money(Math.max(0, sum));
        haptics.select();
      });
      return row;
    };
    for (const l of open) list.appendChild(mkRow(l));
    const hint = document.createElement('div');
    hint.className = 'muted';
    hint.style.cssText = 'font-size:12.5px;padding:6px 2px;';
    hint.textContent = 'Отметь, что сделано за неделю. Выбранные строки закроются актом — в следующий акт не попадут.';
    const go = document.createElement('button');
    go.className = 'btn btn-primary btn-block';
    go.textContent = 'Собрать акт · 0';
    totalEl.style.display = 'none';
    go.addEventListener('click', async () => {
      const ids = [...selected.entries()].filter(([, on]) => on).map(([id]) => id);
      if (!ids.length) { toast('Отметь хотя бы одну строку', { icon: '☑️' }); return; }
      go.disabled = true;
      try {
        const res = await api.estimateAct(estId, ids);
        haptics.success();
        await reloadLines();
        render();
        sheet.close();
        toast(`📋 Акт №${res.act.act_no} собран: ${money(res.act.total)} (${res.closed} строк)`, { icon: '📋' });
      } catch (e2) {
        go.disabled = false;
        toast(e2.message, { tone: 'danger' });
      }
    });
    const sheet = openSheet({ title: '📋 Акт за неделю', body: [list, hint, totalEl, go] });
  }

  // --- v0.6: фото строки --------------------------------------------------------
  // «Фотографируешь комнату на ходу — фото цепляется к последней строке»:
  // снимок живёт в фотоотчёте объекта и виден заказчику на странице сметы.
  function linePhotoSheet(l) {
    const pick = document.createElement('input');
    pick.type = 'file';
    pick.accept = 'image/*';
    pick.capture = 'environment';
    pick.style.display = 'none';
    const info = document.createElement('div');
    info.className = 'muted';
    info.style.cssText = 'font-size:13.5px;line-height:1.5;';
    info.textContent = `Снимок получит подпись «${l.name}» и появится в фотоотчёте объекта и на странице заказчика.`;
    const shoot = document.createElement('button');
    shoot.className = 'btn btn-primary btn-block';
    shoot.textContent = '📷 Сделать снимок';
    shoot.addEventListener('click', () => pick.click());
    pick.addEventListener('change', async () => {
      const f = pick.files?.[0];
      if (!f) return;
      shoot.disabled = true;
      try {
        await api.linePhoto(estId, l.id, f, '');
        haptics.success();
        toast('Фото прикреплено к строке', { icon: '📷' });
        sheet.close();
      } catch (e2) {
        shoot.disabled = false;
        toast(e2.message, { tone: 'danger' });
      }
    });
    const sheet = openSheet({ title: '📷 Фото к строке', body: [info, shoot, pick] });
  }

  // --- v0.6: диалог с заказчиком ----------------------------------------------------
  async function commentsSheet() {
    const list = document.createElement('div');
    list.innerHTML = '<div class="muted" style="padding:8px;font-size:13px;">Загружаю диалог…</div>';
    const sheet = openSheet({ title: '💬 Диалог с заказчиком', body: [list] });
    let items;
    try {
      items = (await api.comments(estId)).items;
    } catch (e) {
      list.innerHTML = `<div class="muted" style="padding:8px;">${escapeHtml(e.message)}</div>`;
      return;
    }
    list.replaceChildren();
    if (!items.length) {
      list.innerHTML = '<div class="muted" style="padding:6px;font-size:13px;">Пока пусто. Комментарии заказчика со страницы сметы появятся здесь — и наоборот.</div>';
    }
    for (const c of items) {
      const row = document.createElement('div');
      row.className = 'cmt-msg' + (c.author === 'client' ? ' client' : '');
      const who = document.createElement('div');
      who.className = 'cmt-who';
      who.textContent = c.author === 'client' ? 'Заказчик' : 'Мастер';
      const tx = document.createElement('div');
      tx.textContent = c.text;
      row.append(who, tx);
      list.appendChild(row);
    }
    const field = Field({ label: 'Ответить заказчику', placeholder: 'например: плитку поменять можно, посчитаю…' });
    const send = document.createElement('button');
    send.className = 'btn btn-primary btn-block';
    send.textContent = 'Отправить';
    send.addEventListener('click', async () => {
      const text = field.input.value.trim();
      if (!text) { field.input.focus(); return; }
      send.disabled = true;
      try {
        await api.addComment(estId, text);
        haptics.success();
        sheet.close();
        toast('Ответ отправлен — заказчик увидит на странице', { icon: '💬' });
      } catch (e2) {
        send.disabled = false;
        toast(e2.message, { tone: 'danger' });
      }
    });
    list.append(field.el, send);
  }

  return {
    el,
    cleanup() {
      closingConfirmation(false);
    },
  };
}

// --- утилиты -----------------------------------------------------------------

/** blobToBase64 — Blob → base64 (без data:-префикса) для POST /voice. */
function blobToBase64(blob) {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result);
      resolve(s.slice(s.indexOf(',') + 1));
    };
    r.onerror = reject;
    r.readAsDataURL(blob);
  });
}

function trimNum(v) {
  const n = Math.round(v * 100) / 100;
  return String(n);
}

function guessHidden(name) {
  const n = name.toLowerCase();
  return /грунт|шпакл|шлиф|армир|штроб|укрыв|заделк|стык|гипсокартон|сетк|уголок|профил|изоляц|оттяжк|обеспыл/.test(n);
}

async function fetchBlob(url) {
  const headers = {};
  const { initData } = await import('../services/tg.js');
  if (initData) headers['X-Telegram-Init-Data'] = initData;
  const res = await fetch(url, { headers });
  if (!res.ok) throw new Error('Не получилось скачать файл');
  return URL.createObjectURL(await res.blob());
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}
