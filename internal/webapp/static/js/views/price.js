/**
 * views/price.js — прайс-лист: поиск, добавление/обновление цены
 * («было → стало», v0.3.5), удаление позиции.
 */

import { api } from '../services/api.js';
import { money } from '../services/format.js';
import { Card, Cell, Empty, Button, SearchBar, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { Field, openSheet, toast, confirmSheet } from '../components/ui/feedback.js';
import { haptics } from '../services/tg.js';
import { icon } from '../components/ui/icon.js';

const UNITS = ['', 'м²', 'м.п.', 'шт', 'компл', 'год'];

export function view({ root }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Прайс-лист', subtitle: 'цены подставятся сами' }));

  const content = document.createElement('main');
  content.className = 'view';
  el.appendChild(content);
  root.appendChild(el);

  let items = [];
  let query = '';

  const search = SearchBar({ placeholder: 'Найти позицию…', onInput: (q) => { query = q.toLowerCase(); paint(); } });
  content.appendChild(search.el);

  const addBtn = Button({
    label: 'Добавить / обновить цену', iconName: 'plus', block: true,
    onClick: () => openUpsertSheet(),
  });
  addBtn.el.style.marginTop = 'var(--gap-m)';
  content.appendChild(addBtn.el);

  const listBox = document.createElement('div');
  listBox.style.marginTop = 'var(--gap-m)';
  listBox.replaceChildren(skeletons(6));
  content.appendChild(listBox);

  function unitLabel(u) {
    return u || 'ед.';
  }

  function paint() {
    listBox.replaceChildren();
    const visible = items.filter((c) => !query || c.name.toLowerCase().includes(query));
    if (!visible.length) {
      listBox.appendChild(Empty({
        iconName: query ? 'search' : 'price',
        title: query ? 'Не нашлось' : 'Прайс пуст',
        sub: query ? 'Попробуй другое слово' : 'Добавь цену — дальше в актах она подставится сама («шлифовка 45 м»)',
      }));
      return;
    }
    const card = Card();
    const head = document.createElement('div');
    head.className = 'card-title row-between';
    head.innerHTML = `<span>Позиции</span><span>${visible.length}</span>`;
    card.appendChild(head);
    const list = document.createElement('div');
    list.className = 'list';
    for (const c of visible) {
      const delSlot = document.createElement('button');
      delSlot.className = 'icon-btn';
      delSlot.style.cssText = 'width:34px;height:34px;';
      delSlot.setAttribute('aria-label', 'Удалить позицию');
      delSlot.appendChild(icon('trash', 15));
      delSlot.addEventListener('click', async (e) => {
        e.stopPropagation();
        haptics.warn();
        const ok = await confirmSheet({
          title: 'Убрать позицию?',
          text: `«${c.name}» исчезнет из прайса. Уже созданные акты не пересчитаются.`,
          confirmLabel: 'Убрать позицию',
          icon: '🗑',
        });
        if (!ok) return;
        try {
          await api.deletePrice(c.name);
          items = items.filter((x) => x.name !== c.name);
          haptics.success();
          toast(`Убрал: ${c.name}`, { icon: '🗑' });
          paint();
        } catch (err) {
          toast(err.message, { tone: 'danger' });
        }
      });
      list.appendChild(Cell({
        icon: 'price',
        title: c.name,
        sub: 'за ' + unitLabel(c.unit),
        value: money(c.price),
        rightSlot: delSlot,
      }));
    }
    card.appendChild(list);
    listBox.appendChild(card);
  }

  function openUpsertSheet(prefill) {
    const name = Field({ label: 'Наименование', placeholder: 'штукатурка', value: prefill?.name || '' });
    const price = Field({ label: 'Цена, грн', inputmode: 'decimal', placeholder: '260', value: prefill ? String(prefill.price) : '' });
    const unit = Field({ label: 'Единица (м², м.п., шт — необязательно)', placeholder: 'м²', value: prefill?.unit || '' });
    const note = document.createElement('p');
    note.className = 'hint-text';
    note.textContent = 'Та же позиция ещё раз — это обновление: покажу «было → стало». Уже созданные акты не пересчитываются.';
    note.style.marginBottom = '14px';

    const save = Button({
      label: 'Сохранить в прайс', block: true,
      onClick: async () => {
        const n = name.input.value.trim();
        const p = parseFloat(price.input.value.replace(',', '.').replace(/\s/g, ''));
        if (n.length < 2) { haptics.error(); toast('Название — от 2 символов', { tone: 'danger' }); return; }
        if (!(p > 0)) { haptics.error(); toast('Цена — положительным числом', { tone: 'danger' }); return; }
        save.setLoading(true);
        try {
          const res = await api.upsertPrice(n, unit.input.value.trim(), p);
          sheet.close();
          haptics.success();
          toast(res.existed ? `Обновил: ${money(res.prev)} → ${money(res.item.price)}` : `Добавил: ${res.item.name}`, { icon: '💵' });
          const fresh = await api.price();
          items = fresh.items;
          paint();
        } catch (err) {
          haptics.error();
          toast(err.message, { tone: 'danger' });
        } finally {
          save.setLoading(false);
        }
      },
    });
    const sheet = openSheet({ title: 'Цена позиции', body: [note, name.el, price.el, unit.el], footer: [save.el] });
    setTimeout(() => name.input.focus(), 350);
  }

  api.price().then((res) => { items = res.items ?? []; paint(); })
    .catch((e) => listBox.replaceChildren(Empty({ iconName: 'alert', title: 'Не загрузилось', sub: e.message })));

  return { el };
}
