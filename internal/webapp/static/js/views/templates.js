/**
 * views/templates.js — шаблоны работ (v0.5, «админка» мастера).
 * «Покраска комнаты под ключ» = готовый техцикл из прайса (грунт → шпаклёвка
 * → шлифовка → покраска). Один тап в конструкторе сметы — и все позиции
 * на месте. Здесь: список, сборка нового из прайса, удаление.
 */

import { api } from '../services/api.js';
import { money } from '../services/format.js';
import { Card, Empty, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { icon } from '../components/ui/icon.js';
import { toast, openSheet, Field, confirmSheet } from '../components/ui/feedback.js';
import { haptics } from '../services/tg.js';

export function view({ root }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Шаблоны работ', subtitle: 'техцикл одним тапом — без перебивания из Excel' }));

  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(3, 'card-sk'));
  el.appendChild(content);

  const fabWrap = document.createElement('div');
  fabWrap.className = 'fab-wrap';
  const fab = document.createElement('button');
  fab.className = 'fab';
  fab.setAttribute('aria-label', 'Новый шаблон');
  fab.appendChild(icon('plus', 24));
  fab.addEventListener('click', () => { haptics.medium(); builderSheet(); });
  fabWrap.appendChild(fab);
  el.appendChild(fabWrap);

  root.appendChild(el);

  let priceCache = [];

  api.templates().then(async (res) => {
    content.replaceChildren();
    if (!res.items.length) {
      content.appendChild(Empty({
        iconName: 'template',
        title: 'Шаблонов нет',
        sub: 'Собери первый: «Стена под покраску» — грунт, шпаклёвка, шлифовка, покраска. Дальше — вставляется в смету одним тапом.',
      }));
      return;
    }
    const priceRes = await api.price().catch(() => ({ items: [] }));
    priceCache = priceRes.items || [];
    for (const tpl of res.items) content.appendChild(tplCard(tpl));
  }).catch((e) => {
    content.replaceChildren();
    content.appendChild(Empty({ iconName: 'alert', title: 'Шаблоны не загрузились', sub: e.message }));
  });

  function tplCard(tpl) {
    const card = Card({ className: 'est-card section' });

    const top = document.createElement('div');
    top.className = 'est-top';
    const ic = document.createElement('span');
    ic.className = 'est-ic';
    ic.appendChild(icon('template', 20));
    const box = document.createElement('div');
    box.style.cssText = 'flex:1;min-width:0;';
    const nm = document.createElement('div');
    nm.className = 'est-name';
    nm.textContent = tpl.name;
    const meta = document.createElement('div');
    meta.className = 'est-obj';
    const sum = tpl.lines.reduce((a, l) => a + l.price * l.qty, 0);
    meta.textContent = `${tpl.lines.length} позиций · скрытых ${tpl.lines.filter((l) => l.hidden).length} · база ${money(sum)}`;
    box.append(nm, meta);
    top.append(ic, box);

    const delBtn = document.createElement('button');
    delBtn.className = 'icon-mini danger';
    delBtn.setAttribute('aria-label', 'Удалить шаблон');
    delBtn.appendChild(icon('trash', 16));
    delBtn.addEventListener('click', async () => {
      const ok = await confirmSheet({
        title: 'Удалить шаблон?',
        text: `«${tpl.name}» (${tpl.lines.length} позиций) уйдёт. Сметы, куда он уже вставлен, не трогаем.`,
        confirmLabel: 'Удалить шаблон',
      });
      if (!ok) return;
      try {
        await api.deleteTemplate(tpl.id);
        haptics.success();
        card.remove();
        toast('Шаблон удалён', { icon: '🗑' });
      } catch (e) {
        toast(e.message, { tone: 'danger' });
      }
    });
    top.appendChild(delBtn);
    card.appendChild(top);

    // состав — раскрытый список (не блины-аккордеоны: мастеру видно сразу)
    const list = document.createElement('div');
    list.style.cssText = 'border:1px solid var(--border);border-radius:var(--radius-m);overflow:hidden;';
    for (const l of tpl.lines) {
      const row = document.createElement('div');
      row.className = 'est-line' + (l.hidden ? ' hiddenw' : '');
      const n = document.createElement('div');
      n.className = 'ln-name';
      n.textContent = l.name;
      const s = document.createElement('div');
      s.className = 'ln-sum';
      s.textContent = `${money(l.price * l.qty)}`;
      const m = document.createElement('div');
      m.className = 'ln-meta';
      m.textContent = `${trimNum(l.qty)} ${l.unit || ''} × ${trimNum(l.price)}`;
      row.append(n, s, m);
      list.appendChild(row);
    }
    card.appendChild(list);
    return card;
  }

  // --- сборка шаблона из прайса -------------------------------------------------
  function builderSheet() {
    if (!priceCache.length) {
      api.price().then((r) => { priceCache = r.items || []; builderSheetOpen(); });
    } else {
      builderSheetOpen();
    }
  }

  function builderSheetOpen() {
    const nameF = Field({ label: 'Название шаблона', placeholder: 'Стена под покраску (полный цикл)' });
    const search = Field({ label: 'Добавить из прайса', placeholder: 'шпаклёвка…' });
    const picked = document.createElement('div');
    picked.style.cssText = 'display:flex;flex-direction:column;gap:6px;max-height:34vh;overflow:auto;';
    const results = document.createElement('div');
    results.style.cssText = 'max-height:26vh;overflow:auto;border:1px solid var(--border);border-radius:var(--radius-m);';

    let chosen = []; // {name, qty, unit, price, hidden}

    const renderChosen = () => {
      picked.replaceChildren();
      if (!chosen.length) {
        const hint = document.createElement('div');
        hint.className = 'muted';
        hint.style.cssText = 'font-size:12.5px;padding:6px 2px;';
        hint.textContent = 'Пока пусто — найди позиции в прайсе ниже. Порядок = техпоследовательность.';
        picked.appendChild(hint);
        return;
      }
      chosen.forEach((l, i) => {
        const row = document.createElement('div');
        row.className = 'est-line' + (l.hidden ? ' hiddenw' : '');
        const n = document.createElement('div');
        n.className = 'ln-name';
        n.textContent = `${i + 1}. ${l.name}`;
        const s = document.createElement('div');
        s.className = 'ln-sum';
        s.textContent = money(l.price * l.qty);
        const acts = document.createElement('div');
        acts.className = 'ln-actions';
        const mkMini = (ic, aria, fn, cls = '') => {
          const b = document.createElement('button');
          b.className = 'icon-mini ' + cls;
          b.setAttribute('aria-label', aria);
          b.appendChild(icon(ic, 14));
          b.addEventListener('click', fn);
          return b;
        };
        acts.appendChild(mkMini('up', 'Выше', () => {
          if (i === 0) return;
          [chosen[i - 1], chosen[i]] = [chosen[i], chosen[i - 1]];
          renderChosen();
        }));
        acts.appendChild(mkMini('down', 'Ниже', () => {
          if (i >= chosen.length - 1) return;
          [chosen[i + 1], chosen[i]] = [chosen[i], chosen[i + 1]];
          renderChosen();
        }));
        acts.appendChild(mkMini(l.hidden ? 'eye' : 'eyeOff', 'Скрытая работа', () => {
          l.hidden = !l.hidden;
          renderChosen();
        }, l.hidden ? 'done-on' : ''));
        acts.appendChild(mkMini('trash', 'Убрать', () => {
          chosen.splice(i, 1);
          renderChosen();
        }, 'danger'));
        const m = document.createElement('div');
        m.className = 'ln-meta';
        m.textContent = `${trimNum(l.qty)} ${l.unit || ''} × ${trimNum(l.price)}`;
        row.append(n, s, m, acts);
        picked.appendChild(row);
      });
    };

    const renderResults = (q) => {
      const query = (q || '').toLowerCase();
      results.replaceChildren();
      const items = priceCache.filter((p) => !query || p.name.includes(query)).slice(0, 20);
      if (!items.length) {
        const d = document.createElement('div');
        d.className = 'muted';
        d.style.cssText = 'padding:10px;font-size:12.5px;';
        d.textContent = priceCache.length ? 'Ничего не нашлось' : 'Прайс пуст — заполните вкладку «Прайс»';
        results.appendChild(d);
        return;
      }
      for (const p of items) {
        const row = document.createElement('button');
        row.type = 'button';
        row.style.cssText = 'display:flex;justify-content:space-between;width:100%;background:none;border:0;border-bottom:1px solid var(--border);padding:10px 12px;font:inherit;color:inherit;cursor:pointer;text-align:left;';
        const nm = document.createElement('span');
        nm.style.cssText = 'font-size:13.5px;';
        nm.textContent = p.name;
        const pr = document.createElement('b');
        pr.className = 'num';
        pr.style.cssText = 'font-size:13px;color:var(--accent);';
        pr.textContent = `${trimNum(p.price)}${p.unit ? '/' + p.unit : ''}`;
        row.append(nm, pr);
        row.addEventListener('click', () => {
          if (chosen.some((c) => c.name === p.name)) {
            toast('Уже в шаблоне', { icon: '☑️' });
            return;
          }
          chosen.push({
            name: p.name, qty: 1, unit: p.unit, price: p.price,
            hidden: /грунт|шпакл|шлиф|армир|штроб|укрыв|заделк|сетк|уголок/.test(p.name),
          });
          haptics.tap();
          renderChosen();
        });
        results.appendChild(row);
      }
    };
    search.input.addEventListener('input', () => renderResults(search.input.value));
    renderResults('');
    renderChosen();

    const save = document.createElement('button');
    save.className = 'btn btn-primary btn-block';
    save.textContent = 'Сохранить шаблон';
    save.addEventListener('click', async () => {
      const name = nameF.input.value.trim();
      if (name.length < 2) {
        toast('Дайте шаблону название', { tone: 'danger', icon: '✍️' });
        return;
      }
      if (!chosen.length) {
        toast('Добавь хоть одну позицию из прайса', { tone: 'danger', icon: '📁' });
        return;
      }
      save.disabled = true;
      try {
        await api.upsertTemplate(name, chosen);
        haptics.success();
        toast('Шаблон сохранён — используй его в любой смете', { icon: '🧰' });
        sheet.close();
        reload();
      } catch (e) {
        save.disabled = false;
        toast(e.message, { tone: 'danger' });
      }
    });

    const sheet = openSheet({
      title: '🧰 Новый шаблон',
      body: [nameF.el, picked, search.el, results],
      footer: [save],
    });
  }

  async function reload() {
    content.replaceChildren();
    content.appendChild(skeletons(2, 'card-sk'));
    try {
      const res = await api.templates();
      content.replaceChildren();
      for (const tpl of res.items) content.appendChild(tplCard(tpl));
      if (!res.items.length) {
        content.appendChild(Empty({ iconName: 'template', title: 'Шаблонов нет', sub: 'Собери первый из прайса' }));
      }
    } catch { /* молча: список уже был */ }
  }

  return { el };
}

function trimNum(v) {
  return String(Math.round(v * 100) / 100);
}
