/**
 * views/objects.js — список объектов с деньгами + создание (нижний лист).
 */

import { api } from '../services/api.js';
import { money, dateShort } from '../services/format.js';
import { Card, Cell, Empty, skeletons, SearchBar, Button } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { Field, openSheet, toast } from '../components/ui/feedback.js';
import { haptics } from '../services/tg.js';

export function view({ root, navigate }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Объекты', subtitle: 'квартиры и дома' }));

  const content = document.createElement('main');
  content.className = 'view';
  el.appendChild(content);
  root.appendChild(el);

  let items = [];
  let query = '';

  const search = SearchBar({ placeholder: 'Поиск по названию или заказчику…', onInput: (q) => { query = q.toLowerCase(); paint(); } });
  content.appendChild(search.el);

  const addBtn = Button({
    label: 'Новый объект', iconName: 'plus', variant: 'primary', block: true,
    onClick: () => openCreateSheet(),
  });

  const listBox = document.createElement('div');
  listBox.style.marginTop = 'var(--gap-m)';
  content.appendChild(listBox);
  content.appendChild(addBtn.el);
  addBtn.el.style.marginTop = 'var(--gap-l)';

  listBox.replaceChildren(skeletons(4));

  function paint() {
    listBox.replaceChildren();
    const visible = items.filter((o) =>
      !query || o.name.toLowerCase().includes(query) || (o.customer || '').toLowerCase().includes(query));
    if (!visible.length) {
      listBox.appendChild(Empty({
        iconName: query ? 'search' : 'objects',
        title: query ? 'Не нашлось' : 'Объектов пока нет',
        sub: query ? 'Попробуй другой запрос' : 'Создай первый — название и заказчик, всё остальное бот возьмёт на себя',
      }));
      return;
    }
    const card = Card();
    const list = document.createElement('div');
    list.className = 'list';
    for (const o of visible) {
      const debt = o.total - o.paid;
      list.appendChild(Cell({
        icon: 'objects',
        title: o.name,
        sub: `${o.customer || 'без заказчика'} · ${dateShort(o.created_at)}`,
        value: money(o.total),
        note: debt > 0.009 ? `долг ${money(debt)}` : `${o.acts} акт(а) · оплачено`,
        onClick: () => navigate('#/objects/' + o.id),
      }));
    }
    card.appendChild(list);
    listBox.appendChild(card);
  }

  function openCreateSheet() {
    const name = Field({ label: 'Название объекта', placeholder: 'ЖК Сонячний, кв. 45' });
    const customer = Field({ label: 'Заказчик (необязательно)', placeholder: 'Иван Петренко' });
    const save = Button({
      label: 'Создать объект', block: true,
      onClick: async () => {
        const n = name.input.value.trim();
        if (n.length < 2) {
          haptics.error();
          name.input.focus();
          toast('Название — от 2 символов', { tone: 'danger' });
          return;
        }
        save.setLoading(true);
        try {
          const obj = await api.createObject(n, customer.input.value.trim());
          sheet.close();
          haptics.success();
          toast(`Объект «${obj.name}» создан`, { icon: '🏠' });
          items = await api.objects();
          paint();
        } catch (e) {
          haptics.error();
          toast(e.message, { tone: 'danger' });
        } finally {
          save.setLoading(false);
        }
      },
    });
    const sheet = openSheet({
      title: 'Новый объект',
      body: [name.el, customer.el],
      footer: [save.el],
    });
    setTimeout(() => name.input.focus(), 350);
  }

  // удаление объекта доступно из детального экрана; тут — только список
  api.objects().then((objs) => { items = objs ?? []; paint(); })
    .catch((e) => { listBox.replaceChildren(Empty({ iconName: 'alert', title: 'Не загрузилось', sub: e.message })); });
  return { el };
}
