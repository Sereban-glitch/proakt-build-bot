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

    // --- добавить: парсер / прайс / шаблон ---
    const addCard = Card({ className: 'section' });
    const addTitle = document.createElement('div');
    addTitle.className = 'card-title';
    addTitle.textContent = 'Добавить позиции';
    addCard.appendChild(addTitle);

    const addWrap = document.createElement('div');
    addWrap.style.cssText = 'display:flex;flex-direction:column;gap:8px;padding:0 14px 14px;';
    const parseF = Field({ label: 'Строкой — как в чате бота', placeholder: 'шпаклёвка 45 м² 140 или шлифовка 45 м' });
    addWrap.appendChild(parseF.el);
    const parseHint = document.createElement('div');
    parseHint.className = 'parse-box';
    parseHint.style.display = 'none';
    addWrap.appendChild(parseHint);

    let parseTimer = null;
    parseF.input.addEventListener('input', () => {
      clearTimeout(parseTimer);
      const text = parseF.input.value.trim();
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
    secondRow.append(fromPrice, fromTpl);
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

  return {
    el,
    cleanup() {
      closingConfirmation(false);
    },
  };
}

// --- утилиты -----------------------------------------------------------------

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
