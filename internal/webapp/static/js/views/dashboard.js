/**
 * views/dashboard.js — рабочий стол мастера.
 * Один главный сценарий, понятные деньги и только те сигналы,
 * которые требуют действия сегодня.
 */

import { api } from '../services/api.js';
import { money, dateShort, plural } from '../services/format.js';
import { Card, Cell, Empty, skeletons, actBadge } from '../components/ui/primitives.js';
import { BrandHeader } from '../components/layout/chrome.js';
import { ProgressBar } from '../components/ui/adv.js';
import { icon } from '../components/ui/icon.js';
import { toast } from '../components/ui/feedback.js';

const STATUS_LABEL = {
  draft: 'Черновик',
  sent: 'У заказчика',
  approved: 'Согласована',
  done: 'Закрыта',
};

function sectionHead(title, action, onClick) {
  const row = document.createElement('div');
  row.className = 'dashboard-section-head';
  const h = document.createElement('h2');
  h.textContent = title;
  row.appendChild(h);
  if (action) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'section-link';
    btn.textContent = action;
    btn.appendChild(icon('chevron', 15));
    btn.addEventListener('click', onClick);
    row.appendChild(btn);
  }
  return row;
}

function quickAction({ title, note, iconName, primary = false, onClick }) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = `quick-action${primary ? ' primary' : ''}`;
  const ic = document.createElement('span');
  ic.className = 'quick-action-ic';
  ic.appendChild(icon(iconName, primary ? 24 : 20));
  const copy = document.createElement('span');
  copy.className = 'quick-action-copy';
  const t = document.createElement('b');
  t.textContent = title;
  const n = document.createElement('small');
  n.textContent = note;
  copy.append(t, n);
  btn.append(ic, copy, icon('arrow', primary ? 21 : 17));
  btn.addEventListener('click', onClick);
  return btn;
}

function miniStat(value, label, onClick) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'work-stat';
  const v = document.createElement('strong');
  v.className = 'num';
  v.textContent = String(value);
  const l = document.createElement('span');
  l.textContent = label;
  btn.append(v, l);
  btn.addEventListener('click', onClick);
  return btn;
}

function setupCard(d, estimates) {
  const steps = [
    { done: d.price_count > 0, label: 'Добавить цены', hash: '#/price' },
    { done: d.stats.objects > 0, label: 'Создать объект', hash: '#/objects?new=1' },
    { done: estimates.length > 0, label: 'Собрать смету', hash: '#/estimates?new=1' },
  ];
  const done = steps.filter((s) => s.done).length;
  if (done === steps.length) return null;

  const card = Card({ className: 'setup-card section' });
  const top = document.createElement('div');
  top.className = 'setup-top';
  const copy = document.createElement('div');
  const eyebrow = document.createElement('div');
  eyebrow.className = 'setup-eyebrow';
  eyebrow.textContent = 'ПЕРВЫЙ ЗАПУСК';
  const title = document.createElement('div');
  title.className = 'setup-title';
  title.textContent = done === 0 ? 'Подготовим всё к первой смете' : 'Осталось совсем немного';
  const sub = document.createElement('div');
  sub.className = 'setup-sub';
  sub.textContent = `${done} из ${steps.length} шагов готово`;
  copy.append(eyebrow, title, sub);
  const count = document.createElement('div');
  count.className = 'setup-count num';
  count.textContent = `${done}/${steps.length}`;
  top.append(copy, count);
  card.appendChild(top);

  const track = document.createElement('div');
  track.className = 'setup-track';
  const fill = document.createElement('span');
  fill.style.width = `${(done / steps.length) * 100}%`;
  track.appendChild(fill);
  card.appendChild(track);

  const list = document.createElement('div');
  list.className = 'setup-list';
  for (const [index, step] of steps.entries()) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = `setup-step${step.done ? ' done' : ''}`;
    const mark = document.createElement('span');
    mark.className = 'setup-step-mark';
    mark.textContent = step.done ? '✓' : String(index + 1);
    const label = document.createElement('span');
    label.textContent = step.label;
    btn.append(mark, label);
    if (!step.done) btn.appendChild(icon('chevron', 16));
    btn.disabled = step.done;
    btn.addEventListener('click', () => { location.hash = step.hash; });
    list.appendChild(btn);
  }
  card.appendChild(list);
  return card;
}

function estimateCard(est, navigate) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'dash-estimate';

  const top = document.createElement('div');
  top.className = 'dash-estimate-top';
  const titleBox = document.createElement('div');
  titleBox.className = 'grow';
  const title = document.createElement('div');
  title.className = 'dash-estimate-title ellipsis';
  title.textContent = est.title;
  const object = document.createElement('div');
  object.className = 'dash-estimate-object ellipsis';
  object.textContent = est.object_name;
  titleBox.append(title, object);
  const status = document.createElement('span');
  status.className = `est-status ${est.status}`;
  status.textContent = STATUS_LABEL[est.status] || est.status;
  top.append(titleBox, status);
  btn.appendChild(top);

  const moneyRow = document.createElement('div');
  moneyRow.className = 'dash-estimate-money';
  const total = document.createElement('strong');
  total.className = 'num';
  total.textContent = money(est.total);
  const progress = document.createElement('span');
  const pct = est.total > 0 ? Math.round((est.done_sum / est.total) * 100) : 0;
  progress.textContent = est.total > 0 ? `${pct}% закрыто актами` : 'Добавьте работы';
  moneyRow.append(total, progress);
  btn.appendChild(moneyRow);
  btn.appendChild(ProgressBar({ value: est.total > 0 ? est.done_sum / est.total : 0 }).el);
  btn.addEventListener('click', () => navigate('#/estimates/' + est.id));
  return btn;
}

export function view({ root, navigate }) {
  const el = document.createElement('div');
  el.appendChild(BrandHeader());

  const content = document.createElement('main');
  content.className = 'view dashboard-view';
  content.appendChild(skeletons(4, 'card-sk'));
  el.appendChild(content);
  root.appendChild(el);

  Promise.all([
    api.dashboard(),
    api.estimates().catch(() => []),
  ]).then(([d, estimates]) => {
    content.replaceChildren();
    const activeEstimates = estimates.filter((e) => e.status !== 'done');
    const debt = Math.max(0, d.stats.total - d.stats.paid);
    const paidRatio = d.stats.total > 0 ? d.stats.paid / d.stats.total : 0;

    // Главная задача всегда первая: пользователь не ищет кнопку по экрану.
    const intro = document.createElement('section');
    intro.className = 'dashboard-intro';
    const hello = document.createElement('h1');
    hello.textContent = 'Что делаем сегодня?';
    const hint = document.createElement('p');
    hint.textContent = 'Начни с действия — остальное приложение подскажет по ходу.';
    intro.append(hello, hint);
    content.appendChild(intro);

    const actions = document.createElement('div');
    actions.className = 'quick-actions';
    const hasObjects = d.stats.objects > 0;
    actions.appendChild(quickAction({
      title: hasObjects ? 'Новая смета' : 'Добавить первый объект',
      note: hasObjects ? 'Расчёт работ для заказчика' : 'Объект нужен для сметы и актов',
      iconName: hasObjects ? 'estimate' : 'objects',
      primary: true,
      onClick: () => { location.hash = hasObjects ? '#/estimates?new=1' : '#/objects?new=1'; },
    }));
    const secondary = document.createElement('div');
    secondary.className = 'quick-actions-row';
    secondary.append(
      quickAction({
        title: 'Объекты', note: `${d.stats.objects} в работе`, iconName: 'objects',
        onClick: () => navigate('#/objects'),
      }),
      quickAction({
        title: 'Прайс', note: `${d.price_count} позиций`, iconName: 'price',
        onClick: () => navigate('#/price'),
      }),
    );
    actions.appendChild(secondary);
    content.appendChild(actions);

    const setup = setupCard(d, estimates);
    if (setup) content.appendChild(setup);

    // Деньги: главным числом показываем то, что ещё нужно получить.
    const finance = Card({
      className: `finance-card section${debt < 0.01 ? ' clear' : ''}`,
      clickable: true,
      onClick: () => navigate('#/report'),
    });
    const ftop = document.createElement('div');
    ftop.className = 'finance-top';
    const fcopy = document.createElement('div');
    const fk = document.createElement('div');
    fk.className = 'finance-kicker';
    fk.textContent = debt > 0 ? 'К ПОЛУЧЕНИЮ' : 'ОПЛАТЫ';
    const famount = document.createElement('div');
    famount.className = 'finance-amount num';
    famount.textContent = debt > 0 ? money(debt) : 'Всё оплачено';
    const fnote = document.createElement('div');
    fnote.className = 'finance-note';
    fnote.textContent = debt > 0
      ? `${d.stats.unpaid_acts} ${plural(d.stats.unpaid_acts, 'акт ждёт', 'акта ждут', 'актов ждут')} оплаты`
      : 'По закрытым актам задолженности нет';
    fcopy.append(fk, famount, fnote);
    const ficon = document.createElement('span');
    ficon.className = 'finance-icon';
    ficon.appendChild(icon(debt > 0 ? 'wallet' : 'check', 24));
    ftop.append(fcopy, ficon);
    finance.appendChild(ftop);
    finance.appendChild(ProgressBar({ value: paidRatio, tone: 'var(--ok)' }).el);
    const progressCopy = document.createElement('div');
    progressCopy.className = 'finance-progress-copy';
    const received = document.createElement('span');
    received.textContent = `Получено ${money(d.stats.paid)}`;
    const completed = document.createElement('span');
    completed.textContent = `Выполнено ${money(d.stats.total)}`;
    progressCopy.append(received, completed);
    finance.appendChild(progressCopy);
    content.appendChild(finance);

    const stats = document.createElement('div');
    stats.className = 'work-stats';
    stats.append(
      miniStat(d.stats.objects, plural(d.stats.objects, 'объект', 'объекта', 'объектов'), () => navigate('#/objects')),
      miniStat(activeEstimates.length, plural(activeEstimates.length, 'смета', 'сметы', 'смет'), () => navigate('#/estimates')),
      miniStat(d.stats.acts, plural(d.stats.acts, 'акт', 'акта', 'актов'), () => navigate('#/acts')),
    );
    content.appendChild(stats);

    // Только реальные проблемы — без абстрактной «упущенной выгоды».
    if (debt > 0.009 || d.stats.zero_acts > 0) {
      content.appendChild(sectionHead('Требует внимания'));
      const attention = Card({ className: 'attention-card', clickable: true, onClick: () => navigate('#/report') });
      const aic = document.createElement('span');
      aic.className = 'attention-ic';
      aic.appendChild(icon('alert', 21));
      const acopy = document.createElement('div');
      acopy.className = 'grow';
      const at = document.createElement('div');
      at.className = 'attention-title';
      at.textContent = debt > 0 ? `Напомнить об оплате ${money(debt)}` : 'Есть акт без итоговой суммы';
      const as = document.createElement('div');
      as.className = 'attention-sub';
      if (debt > 0 && d.stats.zero_acts > 0) {
        as.textContent = `И ещё ${d.stats.zero_acts} ${plural(d.stats.zero_acts, 'акт', 'акта', 'актов')} без суммы`;
      } else if (debt > 0) {
        as.textContent = 'Открой отчёт и проверь, кто ещё не рассчитался';
      } else {
        as.textContent = 'Работы есть, но заказчику ещё нечего выставить';
      }
      acopy.append(at, as);
      attention.append(aic, acopy, icon('chevron', 18));
      content.appendChild(attention);
    }

    if (activeEstimates.length) {
      content.appendChild(sectionHead('Сметы в работе', 'Все сметы', () => navigate('#/estimates')));
      const list = document.createElement('div');
      list.className = 'dashboard-estimates';
      for (const est of activeEstimates.slice(0, 2)) list.appendChild(estimateCard(est, navigate));
      content.appendChild(list);
    }

    content.appendChild(sectionHead('Последние акты', 'Все акты', () => navigate('#/acts')));
    const recent = Card({ className: 'dashboard-recent' });
    if (!d.recent_acts?.length) {
      recent.appendChild(Empty({
        iconName: 'acts',
        title: 'Актов ещё нет',
        sub: 'После выполнения работ создай акт в чате — можно продиктовать голосом.',
      }));
    } else {
      const list = document.createElement('div');
      list.className = 'list';
      for (const a of d.recent_acts.slice(0, 3)) {
        list.appendChild(Cell({
          icon: 'file',
          title: `Акт №${a.act_no} · ${a.object_name}`,
          sub: dateShort(a.date),
          badge: actBadge(a.total, a.paid),
          value: money(a.total),
          onClick: () => navigate('#/acts/' + a.id),
        }));
      }
      recent.appendChild(list);
    }
    content.appendChild(recent);
  }).catch((e) => {
    content.replaceChildren();
    content.appendChild(Empty({ iconName: 'alert', title: 'Данные не загрузились', sub: e.message }));
    if (e.status === 401) toast(e.message, { tone: 'danger' });
  });

  return { el };
}
