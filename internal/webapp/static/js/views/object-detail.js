/**
 * views/object-detail.js — карточка объекта: деньги, акты объекта, фото
 * скрытых работ (сетка), архивация объекта.
 */

import { api } from '../services/api.js';
import { money, dateShort, dateTime, pluralN } from '../services/format.js';
import { Card, Cell, Stat, Empty, Button, actBadge, skeletons } from '../components/ui/primitives.js';
import { Header } from '../components/layout/chrome.js';
import { toast, confirmSheet } from '../components/ui/feedback.js';
import { haptics, isDemo } from '../services/tg.js';
import { icon } from '../components/ui/icon.js';

export function view({ root, params, navigate }) {
  const id = Number(params.id);
  const el = document.createElement('div');
  el.appendChild(Header({ title: 'Объект', subtitle: 'загружаю…' }));
  const content = document.createElement('main');
  content.className = 'view';
  content.appendChild(skeletons(4, 'card-sk'));
  el.appendChild(content);
  root.appendChild(el);

  let headerEl = null;

  api.objects().then(async (objs) => {
    const o = (objs ?? []).find((x) => x.id === id);
    if (!o) {
      content.replaceChildren();
      content.appendChild(Empty({
        iconName: 'search', title: 'Объект не найден', sub: 'Возможно, скрыт из меню — смотри акты в общем списке',
      }));
      return;
    }

    const debt = o.total - o.paid;
    content.replaceChildren();
    headerEl = Header({ title: o.name, subtitle: o.customer || 'без заказчика', back: () => history.back() });
    el.querySelector('.app-header')?.replaceWith(headerEl);

    // деньги объекта
    const grid = document.createElement('div');
    grid.className = 'stat-grid';
    grid.appendChild(Stat({ label: 'Выполнено', value: money(o.total), iconName: 'wallet' }));
    grid.appendChild(Stat({
      label: debt > 0.009 ? 'Долг' : 'Оплачено',
      value: money(debt > 0.009 ? debt : o.paid),
      sub: `${pluralN(o.acts, 'акт', 'акта', 'актов')}`,
      iconName: debt > 0.009 ? 'alert' : 'check',
      hero: false,
    }));
    content.appendChild(grid);

    // Кабинет заказчика: демо — маска, живьём — форма привязки Telegram ID.
    if (!isDemo || o.id === 1) {
      const accessCard = Card({ className: 'section client-access-card' });
      const accessHead = document.createElement('div');
      accessHead.className = 'client-access-head';
      const accessCopy = document.createElement('div');
      const accessTitle = document.createElement('div');
      accessTitle.className = 'card-title';
      accessTitle.textContent = 'Кабинет заказчика';
      const accessSub = document.createElement('p');
      accessSub.textContent = 'Заказчик видит только этот объект, фото, акты и согласованные суммы.';
      accessCopy.append(accessTitle, accessSub);
      const connected = document.createElement('span');
      connected.className = 'client-access-status';
      connected.textContent = isDemo ? 'Подключён' : 'Доступ по ID';
      accessHead.append(accessCopy, connected);
      accessCard.appendChild(accessHead);

      if (isDemo) {
        const idRow = document.createElement('div');
        idRow.className = 'client-access-id';
        const idLabel = document.createElement('span');
        idLabel.textContent = 'Telegram ID заказчика';
        const idValue = document.createElement('strong');
        idValue.className = 'num';
        idValue.textContent = '583•••741';
        idRow.append(idLabel, idValue);
        accessCard.appendChild(idRow);
      } else {
        const form = document.createElement('div');
        form.className = 'client-access-form';
        const input = document.createElement('input');
        input.type = 'text';
        input.inputMode = 'numeric';
        input.placeholder = 'Telegram ID заказчика';
        input.setAttribute('aria-label', 'Telegram ID заказчика');
        const bindBtn = Button({
          label: 'Привязать', iconName: 'check', variant: 'primary', block: true,
          onClick: async () => {
            const v = Number(String(input.value).replace(/\D/g, ''));
            if (!v) { toast('Введи числовой Telegram ID', { icon: '⚠️' }); return; }
            try {
              await api.clientGrant(o.id, v);
              haptics.success();
              toast('Заказчик привязан', { icon: '✅' });
            } catch (e) { toast(e?.message || 'Не привязалось', { icon: '⚠️' }); }
          },
        });
        const unbindBtn = Button({
          label: 'Отключить', iconName: 'close', variant: 'secondary', block: true,
          onClick: async () => {
            const v = Number(String(input.value).replace(/\D/g, ''));
            if (!v) { toast('Введи ID для отключения', { icon: '⚠️' }); return; }
            if (!await confirmSheet({ title: 'Отключить доступ?', text: 'Заказчик перестанет видеть объект.', confirmLabel: 'Отключить' })) return;
            try {
              await api.clientRevoke(o.id, v);
              haptics.success();
              toast('Доступ отключён', { icon: '🔒' });
            } catch (e) { toast(e?.message || 'Не отключилось', { icon: '⚠️' }); }
          },
        });
        form.append(input, bindBtn.el, unbindBtn.el);
        accessCard.appendChild(form);
      }

      const actions = document.createElement('div');
      actions.className = 'client-access-actions';
      const openClient = Button({
        label: 'Открыть как заказчик', iconName: 'eye', variant: 'primary', block: true,
        onClick: () => { location.href = `client.html?object_id=${o.id}#overview`; },
      });
      const copyClient = Button({
        label: 'Скопировать ссылку', iconName: 'share', variant: 'secondary', block: true,
        onClick: async () => {
          const url = new URL(`client.html?object_id=${o.id}#overview`, location.href).href;
          try { await navigator.clipboard.writeText(url); } catch { /* старый WebView */ }
          haptics.success();
          toast('Ссылка на кабинет скопирована', { icon: '🔗' });
        },
      });
      actions.append(openClient.el, copyClient.el);
      accessCard.appendChild(actions);
      content.appendChild(accessCard);
    }

    // акты объекта
    const actsCard = Card({ className: 'section' });
    const at = document.createElement('div');
    at.className = 'card-title';
    at.textContent = 'Акты объекта';
    actsCard.appendChild(at);
    const allActs = await api.acts().catch(() => []);
    const objActs = (allActs ?? []).filter((a) => a.object_id === o.id);
    if (!objActs.length) {
      actsCard.appendChild(Empty({ iconName: 'acts', title: 'Актов ещё нет', sub: 'В чате: «📋 Новый акт» → выбери этот объект' }));
    } else {
      const list = document.createElement('div');
      list.className = 'list';
      for (const a of objActs) {
        list.appendChild(Cell({
          icon: 'file',
          title: `Акт №${a.act_no}`,
          sub: dateShort(a.date),
          badge: actBadge(a.total, a.paid),
          value: money(a.total),
          onClick: () => navigate('#/acts/' + a.id),
        }));
      }
      actsCard.appendChild(list);
    }
    content.appendChild(actsCard);

    // фото скрытых работ
    const phCard = Card({ className: 'section' });
    const pt = document.createElement('div');
    pt.className = 'card-title';
    pt.textContent = 'Фото скрытых работ';
    phCard.appendChild(pt);
    const photos = await api.photos(o.id).catch(() => []);
    if (!photos?.length) {
      phCard.appendChild(Empty({
        iconName: 'camera',
        title: 'Фото пока нет',
        sub: 'В чате пришли снимок с подписью — он появится здесь и в фотоотчёте заказчику',
      }));
    } else {
      const gridPh = document.createElement('div');
      gridPh.className = 'photo-grid';
      for (const p of photos) {
        const cell = document.createElement('div');
        cell.className = 'ph';
        cell.title = (p.caption || '') + (p.act_no ? ` · акт №${p.act_no}` : '');
        const img = document.createElement('img');
        img.loading = 'lazy';
        img.alt = p.caption || 'фото скрытых работ';
        img.src = api.photoURL(p);
        cell.appendChild(img);
        // долгий тап / кнопка удаления — под снимком (v0.3.8: «в обе стороны»)
        const del = document.createElement('button');
        del.className = 'icon-btn';
        del.style.cssText = 'position:absolute;right:4px;bottom:4px;width:28px;height:28px;background:rgba(0,0,0,.45);color:#fff;border:0;';
        del.setAttribute('aria-label', 'Удалить фото');
        del.appendChild(icon('trash', 14));
        del.addEventListener('click', async (e) => {
          e.stopPropagation();
          haptics.warn();
          const when = dateTime(p.created_at);
          const act = p.act_no ? ` · акт №${p.act_no}` : '';
          const ok = await confirmSheet({
            title: 'Убрать снимок?',
            text: `Уйдёт: фото и подпись (${when}${act}). Позиции акта не трогаем.`,
            confirmLabel: 'Удалить фото',
            icon: '🗑',
          });
          if (!ok) return;
          try {
            await api.deletePhoto(o.id, p.id);
            cell.remove();
            haptics.success();
            toast('Снимок удалён', { icon: '🗑' });
          } catch (err) {
            toast(err.message, { tone: 'danger' });
          }
        });
        cell.appendChild(del);
        const cap = document.createElement('div');
        cap.style.cssText = 'position:absolute;left:0;right:0;bottom:0;padding:14px 6px 5px;font-size:10px;color:#fff;background:linear-gradient(transparent, rgba(0,0,0,.55));white-space:nowrap;overflow:hidden;text-overflow:ellipsis;';
        cap.textContent = p.caption || (p.act_no ? `акт №${p.act_no}` : '');
        cell.appendChild(cap);
        gridPh.appendChild(cell);
      }
      phCard.appendChild(gridPh);
    }
    content.appendChild(phCard);

    // архивация («в обе стороны», v0.3.8)
    const archiveBtn = Button({
      label: 'Убрать объект из меню', iconName: 'archive', variant: 'secondary', block: true,
      onClick: async () => {
        const ok = await confirmSheet({
          title: 'Убрать объект?',
          text: `«${o.name}» исчезнет из меню. Акты, оплаты и фото останутся в /acts и в отчёте — ничего не теряется.`,
          confirmLabel: 'Убрать объект',
          icon: '📦',
        });
        if (!ok) return;
        try {
          await api.archiveObject(o.id);
          haptics.success();
          toast('Объект скрыт, деньги остались в отчётах', { icon: '📦' });
          navigate('#/objects');
        } catch (err) {
          toast(err.message, { tone: 'danger' });
        }
      },
    });
    const wrapBtn = document.createElement('div');
    wrapBtn.className = 'section';
    wrapBtn.appendChild(archiveBtn.el);
    content.appendChild(wrapBtn);
  }).catch((e) => {
    content.replaceChildren();
    content.appendChild(Empty({ iconName: 'alert', title: 'Не загрузилось', sub: e.message }));
  });

  return {
    el,
    cleanup() {
      /* фото-URL с авторизационными заголовками не кэшируем */
    },
  };
}
