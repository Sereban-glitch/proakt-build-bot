/**
 * views/acts.js — список актов со статусами оплаты и поиском.
 */

import { api } from '../services/api.js';
import { money, dateShort } from '../services/format.js';
import { Card, Cell, Empty, skeletons, SearchBar, actBadge } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';

export function view({ root, navigate }) {
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Акты', subtitle: 'выполненные работы' }));

  const content = document.createElement('main');
  content.className = 'view';
  el.appendChild(content);
  root.appendChild(el);

  let items = [];
  let query = '';

  const search = SearchBar({
    placeholder: 'Номер, объект, заказчик…',
    onInput: (q) => { query = q.toLowerCase(); paint(); },
  });
  content.appendChild(search.el);

  const listBox = document.createElement('div');
  listBox.style.marginTop = 'var(--gap-m)';
  listBox.replaceChildren(skeletons(5));
  content.appendChild(listBox);

  function paint() {
    listBox.replaceChildren();
    const visible = items.filter((a) =>
      !query ||
      String(a.act_no).includes(query) ||
      a.object_name.toLowerCase().includes(query) ||
      (a.customer || '').toLowerCase().includes(query));
    if (!visible.length) {
      listBox.appendChild(Empty({
        iconName: query ? 'search' : 'acts',
        title: query ? 'Ничего не нашлось' : 'Актов ещё нет',
        sub: query ? 'Попробуй номер акта или объект' : 'Создай первый в чате: «📋 Новый акт» — голосом или строкой',
      }));
      return;
    }
    const card = Card();
    const list = document.createElement('div');
    list.className = 'list';
    for (const a of visible) {
      list.appendChild(Cell({
        icon: 'file',
        title: `Акт №${a.act_no} · ${a.object_name}`,
        sub: `${a.customer || 'без заказчика'} · ${dateShort(a.date)}`,
        badge: actBadge(a.total, a.paid),
        value: money(a.total),
        note: a.total - a.paid > 0.009 ? `долг ${money(a.total - a.paid)}` : (a.total > 0 ? 'оплачен' : ''),
        onClick: () => navigate('#/acts/' + a.id),
      }));
    }
    card.appendChild(list);
    listBox.appendChild(card);
  }

  api.acts().then((acts) => { items = acts ?? []; paint(); })
    .catch((e) => listBox.replaceChildren(Empty({ iconName: 'alert', title: 'Не загрузилось', sub: e.message })));

  return { el };
}
