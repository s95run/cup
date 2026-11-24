# PR в vol1ura/Sat_9am_5km

Реализация открыта: https://github.com/vol1ura/Sat_9am_5km/pull/482

Ветка `feature/cup-s95-api` от `master`. Файлы в этой папке повторяют то, что уже в PR: их не нужно копировать второй раз.

## Что добавляется

- `GET /cup/authorize` — одноразовый HMAC-редирект на `https://cup.s95.<tld>/api/v1/auth/s95/callback` (`ru`, `rs`, `by`, `kz`); без `return_to`, без переопределения адреса в credentials и без `is_admin` / `athlete_code` в токене
- `GET /api/cup/athletes/search` — поиск атлетов (имя или личный код). Отдаёт `runs_count` и `volunteering_by_role`; правило допуска считает Кубок

Протокол этапа Кубок берёт с публичного JSON (`/events/:code_name.json` → `/activities/:id.json`), отдельного эндпоинта не требуется.

Миграций и записей в БД нет. Ключ и секрет берутся из credentials:

```yaml
cup:
  api_key: <случайный ключ>
  sso_secret: <не короче 32 байт>
```
