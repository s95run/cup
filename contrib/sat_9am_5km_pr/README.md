# PR в vol1ura/Sat_9am_5km

Реализация открыта: https://github.com/vol1ura/Sat_9am_5km/pull/482

Ветка `feature/cup-s95-api` от `master`. Файлы в этой папке повторяют то, что уже в PR: их не нужно копировать второй раз.

## Что добавляется

- `GET /cup_sso` — одноразовый HMAC-редирект на `https://cup.s95.<tld>/api/v1/auth/s95/callback` (`ru`, `rs`, `by`, `kz`); без `return_to` и без `is_admin` / `athlete_code` в токене
- `GET /api/cup/athletes/search` — поиск атлетов для набора команды (имя или личный код, число забегов и волонтёрств для допуска)

Протокол этапа Кубок берёт с публичного JSON (`/events/:code_name.json` → `/activities/:id.json`), отдельного эндпоинта не требуется.

Миграций и записей в БД нет. Ключ и секрет берутся из credentials:

```yaml
cup:
  api_key: <случайный ключ>
  sso_secret: <не короче 32 байт>
  # url: http://127.0.0.1:8080  # опционально для локального стенда
  # urls:
  #   ru: http://127.0.0.1:8080
```