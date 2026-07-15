# BMS API

Business Management System API — мультитенантный backend для оптимизации бизнеса:
проекты (направления), услуги, клиенты, задачи, календарь, а также универсальные
связи «всё со всем», теги и типизированные кастом-поля поверх любых сущностей.

## Сборка
```
docker build --platform linux/amd64 -t yuriydubinin100/bms-api:1.0.0 .
```

## Запуск
```
docker run -d \
  --name bms-api \
  --env-file .env \
  -p 28080:8080 \
  --network bms_default \
  yuriydubinin100/bms-api:1.0.0
```

> Тома `docker.sock` / `systemd` / `dbus` / `ssh` / `compose` и `--user root` из dijex убраны —
> они относились к его функционалу (управление докером/ssh/systemd), в bms-api их нет. Вместо них
> `--network bms_default` — чтобы контейнер видел `bms_postgres`. Порт 28080 (18080 занят dijex-api).
> Подключение к БД задаётся в `.env` (`POSTGRES_HOST=bms_postgres`).

## Деплой
```
docker push yuriydubinin100/bms-api:1.0.0
```

```
docker pull yuriydubinin100/bms-api:1.0.0
```

## Стек и архитектура

- **Go 1.25**, чистая архитектура: `transport/http` → `service` → `domain` → `repository/postgres`.
- **chi/v5** — роутер; **pgx/v5** (pgxpool) — доступ к PostgreSQL; **golang-migrate** — миграции.
- **go-playground/validator** — валидация DTO; **log/slog** — логи (JSON в production).
- Авторизация — opaque Bearer-токены (в БД хранится SHA-256 хэш), пароли — bcrypt.
- **Мультитенантность**: каждый запрос скоупится по `customer_id` из токена; доступ к чужому
  кастумеру неотличим от отсутствия записи (404). Оператор системы — `customer_users`.
- **Реестр сущностей**: `projects/services/clients/tasks/calendar_events/deals/customer_users`
  авто-регистрируются в таблице `entities`, что даёт единый механизм связей (`entity_links`),
  тегов и кастом-полей с реальными FK-ограничениями.

Схема БД в миграциях `migrations/` (источник истины). Приложение применяет их на старте.

## Запуск на сервере (рядом с bms_postgres)

БД уже поднята контейнером `bms_postgres` (compose-проект `bms`, сеть `bms_default`,
БД `bms_db`). API подключается к ней по внутренней docker-сети и слушает **28080**
(чтобы не конфликтовать с 3000/13080/18080/5432/15432).

```bash
# 1. Проверить имя сети БД (ожидается bms_default):
docker network ls | grep bms

# 2. Собрать и запустить API (использует .env и внешнюю сеть bms_default):
docker compose up -d --build

# 3. Проверить:
curl http://localhost:28080/api/ping
docker compose logs -f
```

`.env` по умолчанию настроен на `POSTGRES_HOST=bms_postgres` / `POSTGRES_PORT=5432`
(внутри сети). Пароль/пользователь/БД совпадают с `db/docker-compose.yml`.

## Локальная разработка (с хоста)

```bash
# В .env переключить на localhost:15432 (см. закомментированный ВАРИАНТ 2), затем:
make run          # go run ./cmd/api  (миграции применяются автоматически)
make test         # go test -race ./...
make build        # бинарь в bin/api
```

## Первый вход

Публичная регистрация создаёт кастумер и первого пользователя-владельца и сразу выдаёт токен:

```bash
curl -X POST http://localhost:28080/api/auth/register -H 'Content-Type: application/json' \
  -d '{"company_name":"Моя компания","full_name":"Иван Владелец","email":"owner@example.com","password":"supersecret1"}'
```

Ответ содержит `token` — передавай его в защищённые запросы как `Authorization: Bearer <token>`.

## Эндпоинты

Базовый URL: `http://localhost:28080`. Защищённые требуют заголовок `Authorization: Bearer <token>`.

### Публичные

| Метод | Путь | Описание |
|---|---|---|
| `GET`  | `/api/ping` | Health-check. |
| `POST` | `/api/auth/register` | Регистрация кастумера + владельца, выдаёт токен. |
| `POST` | `/api/auth/login` | Логин по email+паролю, выдаёт Bearer-токен. |

### Профиль, кастумер, операторы (защищённые)

| Метод | Путь | Описание |
|---|---|---|
| `POST`   | `/api/auth/logout` | Отозвать текущий токен. |
| `GET`    | `/api/me` | Текущая сессия: `user_id`, `customer_id`, `role`. |
| `GET`    | `/api/customer` | Данные текущего кастумера. |
| `PUT`    | `/api/customer/update` | Обновить кастумер (OWNER/ADMIN). |
| `POST`   | `/api/users/create` | Создать оператора (OWNER/ADMIN). |
| `GET`    | `/api/users/list` | Список операторов (пагинация/фильтры). |
| `PUT`    | `/api/users/update` | Обновить оператора (OWNER/ADMIN; последний OWNER защищён). |
| `DELETE` | `/api/users/delete` | Мягко удалить оператора (OWNER/ADMIN; последний OWNER защищён). |

### Бизнес-сущности (защищённые)

Единый CRUD-контракт для каждой: `POST …/create`, `GET …/list` (пагинация: `page`,
`page_size` (max 100), фильтры, `sort_by`, `order`), `PUT …/update`, `DELETE …/delete` (soft-delete).

| Сущность | Базовый путь | Особенности |
|---|---|---|
| Projects | `/api/projects/*` | направления бизнеса; `attributes` (JSONB), `slug` (уникален в кастумере). |
| Services | `/api/services/*` | `price`, `currency`, `duration_min`; необязательный `project_id`. |
| Clients  | `/api/clients/*`  | контактные поля, `address` (JSONB), `status`. |
| Tasks    | `/api/tasks/*`    | `assigned_to`, `priority`, `due_at`; статус `DONE` авто-проставляет `completed_at`. |
| Calendar | `/api/calendar-events/*` | интервал `starts_at`<`ends_at`; фильтр по окну `from`/`to`. |
| Deals    | `/api/deals/*`    | сделки (CRM): `amount`/`currency`, воронка `status`, `type` (доход/расход), `probability`, привязка `client_id`/`project_id`/`service_id`. |

Участники события:

| Метод | Путь | Описание |
|---|---|---|
| `POST`   | `/api/calendar-events/attendees/add` | Добавить участника (любая сущность кастумера). |
| `DELETE` | `/api/calendar-events/attendees/remove` | Убрать участника. |
| `GET`    | `/api/calendar-events/attendees/list?event_id=…` | Список участников. |

### Связи «всё со всем», теги, кастом-поля (защищённые)

| Метод | Путь | Описание |
|---|---|---|
| `POST`   | `/api/links/create` | Связать любые две сущности кастумера (`relation_type`). |
| `GET`    | `/api/links/list?entity_id=…` | Связи сущности (неориентированно) + тип связанной. |
| `DELETE` | `/api/links/delete` | Удалить связь. |
| `POST`   | `/api/tags/create` · `GET /list` · `PUT /update` · `DELETE /delete` | CRUD тегов. |
| `POST`   | `/api/tags/attach` · `DELETE /api/tags/detach` | Навесить/снять тег на сущность. |
| `POST`   | `/api/custom-fields/definitions/create` · `GET /list` · `PUT /update` · `DELETE /delete` | Конструктор полей per тип сущности. |
| `POST`   | `/api/custom-fields/values/set` | Upsert значения поля для сущности (типизированно). |
| `GET`    | `/api/custom-fields/values/list?entity_id=…` | Значения полей сущности. |
| `DELETE` | `/api/custom-fields/values/unset` | Удалить значение. |

## Модель данных: роли, статусы, поля

Общие правила для всех бизнес-сущностей:

- В ответе всегда есть `id` (UUID), `created_at`, `updated_at` (RFC3339).
- `customer_id` **не передаётся** в теле запроса — берётся из токена (кастумер).
- Тело `…/update` = тело `…/create` + обязательное поле `id`.
- Тело `…/delete` — только `{ "id": "<uuid>" }`. Удаление мягкое (`deleted_at`, наружу не отдаётся).
- Списки `…/list` принимают `page` (с 1), `page_size` (деф. 20, макс. 100), `sort_by`, `order` (`asc`/`desc`), и фильтры сущности.
- `attributes` у сущностей — произвольный JSON-объект (по умолчанию `{}`); хранится в JSONB.

### Роли операторов (`customer_user.role`)

| Роль | Права |
|---|---|
| `OWNER` | Полный доступ; управление кастумером и операторами. Последнего OWNER удалить/понизить нельзя. |
| `ADMIN` | Управление кастумером (`/customer/update`) и операторами (`/users/*`). |
| `MANAGER` | Работа со всеми бизнес-сущностями; **без** управления операторами/кастумером. |
| `STAFF` | То же, что MANAGER (базовый оператор). |

Мутации `/customer/update` и `/users/*` доступны только `OWNER`/`ADMIN` (иначе `403 FORBIDDEN`).
Остальные защищённые роуты доступны любому аутентифицированному оператору кастумера.

### Статусы и перечисления (значения ENUM)

| Сущность / поле | Допустимые значения | По умолчанию |
|---|---|---|
| `project.status` | `DRAFT`, `ACTIVE`, `ON_HOLD`, `ARCHIVED` | `ACTIVE` |
| `service.status` | `ACTIVE`, `INACTIVE`, `ARCHIVED` | `ACTIVE` |
| `client.status` | `LEAD`, `ACTIVE`, `INACTIVE`, `ARCHIVED` | `LEAD` |
| `task.status` | `TODO`, `IN_PROGRESS`, `BLOCKED`, `DONE`, `CANCELLED` | `TODO` |
| `task.priority` | `LOW`, `MEDIUM`, `HIGH`, `URGENT` | `MEDIUM` |
| `calendar_event.status` | `SCHEDULED`, `COMPLETED`, `CANCELLED` | `SCHEDULED` |
| `deal.status` | `NEW`, `PENDING`, `WON`, `LOST` | `NEW` |
| `deal.type` | `INCOME`, `EXPENSE` | `INCOME` |
| `attendee.response_status` | `PENDING`, `ACCEPTED`, `DECLINED`, `TENTATIVE` | `PENDING` |
| `custom_field.field_type` | `TEXT`, `NUMBER`, `BOOLEAN`, `DATE`, `DATETIME`, `JSON`, `SELECT` | — |
| `entity_type` (links / custom fields) | `PROJECT`, `SERVICE`, `CLIENT`, `TASK`, `CALENDAR_EVENT`, `DEAL`, `CUSTOMER_USER` | — |
| `entity_link.relation_type` | произвольная строка (напр. `HAS_SERVICE`, `ORDERED_BY`) | `RELATED` |
| `customer_user.role` | `OWNER`, `ADMIN`, `MANAGER`, `STAFF` | `STAFF` |

### Project — `/api/projects/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `name` | string | ✔ | 2–255 символов. |
| `slug` | string | — | ≤100; уникален в рамках кастумера. |
| `direction` | string | — | Свободное описание направления («автопарк», «салон красоты»). ≤255. |
| `description` | string | — | ≤5000. |
| `status` | enum | — | см. таблицу статусов. |
| `attributes` | JSON | — | произвольные атрибуты. |
| `starts_at` | date | — | формат `YYYY-MM-DD`. |
| `ends_at` | date | — | формат `YYYY-MM-DD`. |

Фильтры списка: `status`, `search` (по name/direction). Сортировка: `created_at`, `updated_at`, `name`, `status`.

### Service — `/api/services/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `project_id` | uuid | — | «домашний» проект; должен принадлежать кастумеру. |
| `name` | string | ✔ | 2–255. |
| `category` | string | — | ≤255. |
| `description` | string | — | ≤5000. |
| `price` | number | — | ≥ 0; NUMERIC(12,2). |
| `currency` | string | — | ISO-код, 3 буквы (деф. `USD`). |
| `duration_min` | integer | — | длительность в минутах, ≥ 1. |
| `status` | enum | — | см. статусы. |
| `attributes` | JSON | — | |

Фильтры: `project_id`, `status`, `category`, `search` (по name). Сортировка: `created_at`, `updated_at`, `name`, `price`, `status`.

### Client — `/api/clients/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `project_id` | uuid | — | принадлежит кастумеру. |
| `first_name` | string | — | ≤255. |
| `last_name` | string | — | ≤255. |
| `company_name` | string | — | ≤255. |
| `email` | string | — | валидный email, ≤255. |
| `phone` | string | — | ≤50. |
| `address` | JSON | — | произвольный объект адреса. |
| `status` | enum | — | см. статусы. |
| `source` | string | — | откуда пришёл клиент, ≤255. |
| `attributes` | JSON | — | |

Правило: хотя бы одно из `first_name` / `last_name` / `company_name` должно быть непустым.
Фильтры: `project_id`, `status`, `search` (по имени/компании/email/телефону). Сортировка: `created_at`, `updated_at`, `first_name`, `last_name`, `company_name`.

### Task — `/api/tasks/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `project_id` | uuid | — | принадлежит кастумеру. |
| `client_id` | uuid | — | клиент, к которому относится задача; принадлежит кастумеру. |
| `deal_id` | uuid | — | сделка, к которой относится задача; принадлежит кастумеру. |
| `title` | string | ✔ | 2–500. |
| `description` | string | — | ≤10000. |
| `status` | enum | — | при `DONE` сервер сам проставляет `completed_at`. |
| `priority` | enum | — | см. статусы. |
| `assigned_to` | uuid | — | оператор (`customer_user`) того же кастумера. |
| `due_at` | datetime | — | RFC3339. |
| `attributes` | JSON | — | |

Только в ответе: `created_by` (оператор из токена), `completed_at` (авто при `DONE`).
Фильтры: `project_id`, `client_id`, `deal_id`, `status`, `priority`, `assigned_to`, `search` (по title), `due_before`, `due_after`. Сортировка: `created_at`, `updated_at`, `due_at`, `priority`, `status`, `title`.

### Calendar event — `/api/calendar-events/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `project_id` | uuid | — | принадлежит кастумеру. |
| `title` | string | ✔ | 2–255. |
| `description` | string | — | ≤10000. |
| `location` | string | — | ≤500. |
| `starts_at` | datetime | ✔ | RFC3339. |
| `ends_at` | datetime | ✔ | RFC3339; строго больше `starts_at`. |
| `all_day` | boolean | — | деф. `false`. |
| `status` | enum | — | см. статусы. |
| `recurrence_rule` | string | — | iCal RRULE, ≤1000. |
| `timezone` | string | — | деф. `UTC`. |
| `attributes` | JSON | — | |

Только в ответе: `created_by`. Фильтры: `project_id`, `status`, `search`, `from`, `to` (окно пересечения по времени). Сортировка: `starts_at`, `created_at`, `updated_at`, `title`, `status`.

Участники (`/api/calendar-events/attendees/*`):

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `event_id` | uuid | ✔ | событие кастумера. |
| `attendee_entity_id` | uuid | ✔ | любая сущность кастумера (клиент, оператор и т.д.). |
| `response_status` | enum | — | см. статусы (для `add`). |

### Deal (сделка) — `/api/deals/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `project_id` | uuid | — | направление; принадлежит кастумеру. |
| `client_id` | uuid | — | клиент сделки; принадлежит кастумеру. |
| `service_id` | uuid | — | услуга, продаваемая/покупаемая в сделке; принадлежит кастумеру. |
| `title` | string | ✔ | 2–255. |
| `description` | string | — | ≤10000. |
| `status` | enum | — | этап воронки (`NEW`/`PENDING`/`WON`/`LOST`); `WON`/`LOST` авто-проставляют `closed_at`. |
| `type` | enum | — | `INCOME` (доход) или `EXPENSE` (расход); деф. `INCOME`. |
| `amount` | number | — | сумма сделки, ≥ 0; NUMERIC(14,2). |
| `currency` | string | — | ISO-код, 3 буквы (деф. `USD`). |
| `probability` | integer | — | вероятность закрытия, 0–100. |
| `expected_close_at` | date | — | ожидаемая дата закрытия, `YYYY-MM-DD`. |
| `assigned_to` | uuid | — | ответственный оператор кастумера. |
| `attributes` | JSON | — | |

Только в ответе: `created_by` (оператор из токена), `closed_at` (авто при `WON`/`LOST`). Фильтры: `project_id`, `client_id`, `service_id`, `status`, `type`, `assigned_to`, `search` (по title). Сортировка: `created_at`, `updated_at`, `title`, `status`, `amount`, `expected_close_at`.

### Overview (агрегированные представления)

Одним запросом возвращают сущность вместе со всем связанным (в разрезе кастумера):

| Метод | Путь | Что возвращает |
|---|---|---|
| `GET` | `/api/deals/overview?id=<uuid>` | `deal` + `project` + `client` + `service` (по `service_id`) + `tasks` (по `deal_id`) + `links` (прочие связи) |
| `GET` | `/api/clients/overview?id=<uuid>` | `client` + `project` + `deals` (по `client_id`) + `tasks` (по `client_id`) + `links` |

Каждый элемент в `links` содержит `linked_type` и `linked_label` — так фронт сразу показывает связанные услуги/сущности без дозапросов. Ошибки: `404 DEAL_NOT_FOUND` / `404 CLIENT_NOT_FOUND`, `422` (нет `id`).

### Entity link (связь «всё со всем») — `/api/links/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `src_entity_id` | uuid | ✔ | сущность-источник (id из реестра `entities`). |
| `dst_entity_id` | uuid | ✔ | сущность-приёмник; ≠ `src_entity_id`. |
| `relation_type` | string | — | тип связи, ≤100 (деф. `RELATED`). |
| `attributes` | JSON | — | |

В ответе `list`: `link_id`, `entity_id`, `linked_entity_id`, `linked_type` (тип связанной сущности), `linked_label` (её имя/заголовок — чтобы показать связь без дозапроса), `relation_type`, `attributes`, `created_at`.
Фильтр `?entity_id=…` возвращает связи, где сущность участвует любой из сторон (неориентированно).

### Tag — `/api/tags/*`

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `name` | string | ✔ | 1–100; уникален в кастумере. |
| `color` | string | — | ≤32 (напр. `#ff0000`). |

Навесить/снять (`attach`/`detach`): `{ "tag_id": "<uuid>", "entity_id": "<uuid>" }` (обе стороны — кастумера).

### Custom fields — `/api/custom-fields/*`

Определение поля (`definitions/*`):

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `entity_type` | enum | ✔ | для какого типа сущности поле (`PROJECT`…`CUSTOMER_USER`). |
| `field_key` | string | ✔ | машинный ключ, ≤100; уникален в паре (кастумер, entity_type). |
| `label` | string | ✔ | отображаемое имя, ≤255. |
| `field_type` | enum | ✔ | `TEXT`/`NUMBER`/`BOOLEAN`/`DATE`/`DATETIME`/`JSON`/`SELECT`. |
| `options` | JSON | — | напр. список вариантов для `SELECT`. |
| `is_required` | boolean | — | деф. `false`. |
| `sort_order` | integer | — | порядок отображения. |

Значение поля (`values/set`):

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `entity_id` | uuid | ✔ | сущность кастумера; её тип должен совпадать с `entity_type` определения. |
| `field_definition_id` | uuid | ✔ | id определения. |
| `value` | по типу поля | ✔ | строка/число/булево/`"YYYY-MM-DD"`/RFC3339/JSON — согласно `field_type`. |

### Operator (`customer_user`) — `/api/users/*` (OWNER/ADMIN)

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `email` | string | ✔ | валидный email; глобально уникален. |
| `full_name` | string | ✔ | 2–255. |
| `role` | enum | — | `OWNER`/`ADMIN`/`MANAGER`/`STAFF` (деф. `STAFF`). |
| `password` | string | ✔/— | при `create` обязателен (≥8); при `update` — только для смены. |
| `is_active` | boolean | — | деф. `true`; `false` отзывает активные токены пользователя. |

### Customer (кастумер) — `/api/customer`, `/api/customer/update` (OWNER/ADMIN)

| Поле | Тип | Обяз. | Описание |
|---|---|:---:|---|
| `name` | string | ✔ | 1–255. |
| `legal_name` | string | — | юр. название. |
| `industry` | string | — | отрасль. |
| `email` | string | ✔ | валидный email. |
| `phone` | string | — | ≤50. |
| `timezone` | string | — | деф. `UTC`. |
| `locale` | string | — | деф. `ru`. |
| `settings` | JSON | — | произвольные настройки кастумера. |

Только для чтения (в ответе): `subscription_plan` (деф. `TRIAL`), `subscription_status` (деф. `ACTIVE`).

### Регистрация и вход — `/api/auth/*` (публичные)

`register` создаёт кастумер + владельца: `company_name` (2–255), `full_name` (2–255), `email`, `password` (≥8) — все обязательны.
`login`: `email`, `password` — обязательны. Оба возвращают `token`, `expires_at`, `customer_id` и краткие данные пользователя.

## Формат ответов

Успех — JSON тело. Ошибка — единый конверт:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "…", "details": [{ "field": "name", "message": "is required" }] } }
```

Коды: `INVALID_JSON` (400), `VALIDATION_ERROR` (422), `UNAUTHORIZED`/`TOKEN_EXPIRED`/`TOKEN_REVOKED`/`USER_DISABLED` (401),
`FORBIDDEN` (403), `<ENTITY>_NOT_FOUND` (404), `<ENTITY>_EXISTS`/`EMAIL_EXISTS` (409), `INTERNAL_ERROR` (500).

## Сборка образа

```bash
docker build --platform linux/amd64 -t bms-api:1.0.0 .
```
