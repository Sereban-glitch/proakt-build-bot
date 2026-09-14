# ПрорАКТ · Telegram Mini App (v0.5)

Мини-приложение прямо в Telegram: дашборд с деньгами, объекты,
**сметы со скрытыми работами** (v0.5), **шаблоны работ**, акты с оплатами,
прайс-лист и отчёт «упущенной выгоды» — с тап-удобным UI на телефоне.
Работает на **том же бинарнике**, что и бот, и на **той же базе**: данные,
созданные в чате, видны в приложении, и наоборот.

**Новое в v0.5 — модуль «Смета+»** (заточен под живой сценарий мастера:
смета в Excel → акты частями → заказчик спрашивает, «на что деньги»):

- 🧾 **Конструктор сметы** — позиции строкой (тот же парсер, что в чате:
  «шпаклёвка 45 140»), из прайса с поиском или **шаблоном работ** одним
  тапом; правка количества степпером, ↑↓, пометка «закрыто актом»;
- ▪ **Скрытые работы** — грунтовка/армировка/шпаклёвка помечаются как
  «скрытая подготовка»: в смете они подсвечены, на клиентской странице
  объяснены заказчику, доля считается автоматически (`hidden_share`);
- 🧮 **Коэффициент сложности** (×1.0…×2.0) — практика живого прайс-листа
  (потолки 280–300 см, лестницы, сжатые сроки): один тап — и вся смета
  пересчитана, суммы в интерфейсе, XLSX и на share-странице сходятся;
- 🔗 **Ссылка заказчику** `/s/<токен>` — самодостаточная страница без
  авторизации: чистовые vs скрытые работы, фото скрытых работ, итоги.
  Токен криптостойкий, перевыпуск одной кнопкой, noindex;
- 📊 **XLSX-экспорт** в формате таблицы мастера (`вид робіт | м²,шт | м.п |
  ціна | сума | примітки | сума з розділів`) с живыми формулами и
  подсветкой скрытых работ;
- 🧰 **Шаблоны работ** («админка») — техцикл из прайса («Стена под
  покраску»: грунт → шпаклёвка → шлифовка → покраска), вставляется в
  любую смету одним тапом;
- 🛡 **TMA 2026** — `disableVerticalSwipes` (длинные сметы не закрываются
  свайпом), `enableClosingConfirmation` при несохранённых правках,
  safe-area, хаптика на каждом действии, skeleton-загрузка.

> Патч v0.5: исправлена сериализация JSON (v0.4 в live-режиме отдавала
> поля в CamelCase — `total` вместо `Total`), у актов теперь заполнена
> дата в списках Mini App.

```
Telegram (кнопка 🧰 ПрорАКТ)
   │  HTTPS
   ▼
proakt :8443  ← WEBAPP_LISTEN (единственный новый порт, по желанию)
   ├── /            → веб-приложение (встроено в бинарник, go:embed)
   ├── /api/*       → JSON REST API (подпись initData, HMAC-SHA256)
   └── PostgreSQL   → те же таблицы, что использует бот
```

## Быстрый старт

1. **Telegram требует HTTPS.** Самый дешёвый путь — обратный прокси
   (Caddy автоматически получает сертификат):

   ```
   # Caddyfile
   tma.example.com {
       reverse_proxy 127.0.0.1:8443
   }
   ```

2. **Две переменные в .env:**

   ```bash
   WEBAPP_LISTEN=127.0.0.1:8443     # пусто = веб-часть выключена (0 портов)
   WEBAPP_URL=https://tma.example.com
   ```

3. **Перезапусти бота.** При старте он сам вызовет `setChatMenuButton` —
   в чате появится кнопка «🧰 ПрорАКТ» (слева от поля ввода). Открой —
   приложение готово.

> Пользователь должен хотя бы раз нажать /start в боте (так создаётся его
> чат-пространство). Первый пользователь становится админом — как и раньше.

## Что внутри (модульная структура)

Бэкенд — новый пакет `internal/httpapi`, зависимости направлены внутрь
(порты-и-адаптеры, как и весь проект):

```
internal/httpapi/
  auth.go       проверка подписи Telegram initData (HMAC-SHA256, TTL 24ч)
  server.go     Service (интерфейс данных) + HTTP-сервер + middleware
  handlers.go   REST: /api/dashboard, /api/objects, /api/acts,
                /api/payments, /api/price, /api/photos
  httpapi_test.go  тесты подписи, авторизации и маршрутов (без БД)

internal/webapp/
  embed.go      go:embed статики + фолбэк на index.html
  static/       само веб-приложение (zero-build ES-модули, без Node)

internal/store/  +1 метод: ActChatID (проверка владения актом для API)
internal/tg/     +1 метод: SetChatMenuButton (кнопка «🧰 ПрорАКТ»)
```

### Слои веб-приложения (static/)

```
js/services/     слой сервисов: изоляция платформы и данных
  tg.js            ЕДИНСТВЕННОЕ место, знающее о Telegram.WebApp:
                   init/expand, тема, хаптика, BackButton/MainButton,
                   попапы; вне Telegram — мягкие no-op (demo)
  api.js           REST-клиент: подписывает запросы initData,
                   нормализует ошибки; вне Telegram → mock
  mock.js          демо-данные (?demo=1) — превью без бэкенда
  format.js        деньги («11 700 грн»), даты, множественные числа

js/hooks/        переиспользуемая реактивность
  useTheme.js      themeParams → CSS-переменные, слушатель themeChanged
  useStore.js      крошечный стор + защита от гонок запросов

js/components/   UI-кит (каждый компонент — (props) → HTMLElement)
  ui/primitives.js   Button (loading/disabled), Card, Cell, Stat,
                     Badge/actBadge, Empty, Skeleton, SearchBar
  ui/feedback.js     toast, bottom-sheet, confirm-sheet, Field
  ui/icon.js         инлайн-SVG-иконки (стиль Lucide, stroke=currentColor)
  layout/chrome.js   Header (sticky, glass) + TabBar (плавающий)

js/views/        экраны: dashboard, objects, object-detail, acts,
                 act-detail, price, report
js/app.js        hash-роутер: монтирование, BackButton по глубине маршрута
js/main.js       загрузка: SDK → тема → роутер → таб-бар
```

## Как переиспользовать компоненты

Любой экран — это функция, собирающая DOM из кита:

```js
import { Button, Card, Cell } from '../components/ui/primitives.js';
import { toast, confirmSheet } from '../components/ui/feedback.js';
import { api } from '../services/api.js';
import { haptics } from '../services/tg.js';

const btn = Button({
  label: 'Удалить', iconName: 'trash', variant: 'danger', block: true,
  onClick: async () => {
    if (await confirmSheet({ title: 'Удалить?', text: '…', confirmLabel: 'Удалить' })) {
      await api.deleteAct(id);
      haptics.success();
      toast('Удалено', { icon: '🗑' });
    }
  },
});
card.appendChild(btn.el);       // ← элемент доступен как .el
btn.setLoading(true);           // ← состояние загрузки встроено
```

Новые экраны подключаются двумя строками: файл в `js/views/` + `register(...)`
в `js/main.js`. API нового метода: эндпоинт в `handlers.go` + строка в
`routes()` — компоненты переиспользуют `api.js` без изменений.

### REST API смет (v0.5)

```
GET    /api/estimates?object_id=      список смет (с деньгами и прогрессом)
POST   /api/estimates                 {object_id, title, coeff, note}
GET    /api/estimates/{id}            смета + позиции
PATCH  /api/estimates/{id}            {title?|status?|coeff?|note?}
DELETE /api/estimates/{id}
POST   /api/estimates/{id}/lines      одна позиция ИЛИ {lines:[…]} (шаблон)
PATCH  /api/estimates/{id}/lines/{lid}  {name?|qty?|price?|unit?|hidden?|done?|note?}
DELETE /api/estimates/{id}/lines/{lid}
POST   /api/estimates/{id}/lines/{lid}/move  {dir:"up"|"down"}
POST   /api/estimates/{id}/share      → {token, url} (перевыпуск = новый токен)
GET    /api/estimates/{id}/xlsx       файл сметы (формат таблицы мастера)
POST   /api/parse-line                {text:"шлифовка 45 м 50"} → позиции (парсер бота)
GET    /api/templates                 шаблоны работ
POST   /api/templates                 {name, lines:[{name,qty,unit,price,hidden}]}
DELETE /api/templates/{id}
GET    /s/{token}                     публичная страница заказчика (без auth)
GET    /s/{token}/photo/{id}          фото скрытых работ (только этой сметы)
```

Деньги считаются **на сервере**: строка хранит базу `qty × price`, итоги
умножаются на коэффициент сметы в SQL-агрегатах — смена коэффициента
пересчитывает всё мгновенно и одинаково в UI, XLSX и на share-странице.

## Безопасность

- **Подпись**: каждый запрос несёт Telegram `initData`; сервер проверяет
  HMAC-SHA256 с секретом `HMAC(bot_token, "WebAppData")` и отвергает
  подписи старше 24 ч (анти-replay).
- **Скоупинг**: chat_id берётся ТОЛЬКО из проверенной подписи. Перед
  удалением акта/оплаты сервер проверяет владение (`store.ActChatID`) —
  чужой id из интернета даёт 404, а не чужие данные.
- **Файлы**: отдача фото идёт только владельцу, путь проверяется на
  выход за `FILES_DIR` (anti-path-traversal).
- **Без SDK на сервере**: только стандартная библиотека Go.

## Demo-режим

Если открыть приложение не из Telegram (или добавить `?demo=1`), фронтенд
честно показывает чип «DEMO» и работает на мок-данных (`services/mock.js`) —
удобно для разработки UI без сервера и БД.

## Что осталось в чате (намеренно)

Создание актов — в чате: там голосовой ввод (главная фишка) и FSM-черновик.
Мини-приложение — «приборная панель»: смотреть деньги, закрывать долги,
вести прайс, показывать фотоотчёт. Всё, что удаляется, — удаляется
в обе стороны (акты, оплаты, фото, позиции прайса, объекты).
