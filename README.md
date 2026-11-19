# Кубок С95 — cup.s95.ru

Сервис командного чемпионата С95: сетка этапов, заявки капитанов, протоколы и зачёт сезона.

## Стек

- **Backend:** Go, Viper, Chi, pgx, goose
- **Frontend:** Vue 3, Vite, Tailwind CSS v4
- **БД:** PostgreSQL, изолированная схема `cup` на `vm-data`
- **Деплой:** Docker Compose (`vm-app`), Ansible

## Быстрый старт (локально)

```bash
cp configs/config.example.yaml configs/config.yaml
make up
make migrate
```

API: http://localhost:8080/api/v1/healthz  
Web: http://localhost:3000

## Деплой

На vm-app стек слушает только `127.0.0.1`. HAProxy на том же хосте направляет `cup.s95.ru` на `127.0.0.1:3000`. База — схема `cup` на vm-data, роль `cup_app`.

Секреты лежат в Ansible Vault (`inventories/<env>/group_vars/vault.yml`, образец — `vault.example.yml`), на сервере — `/etc/cup/cup.env`.

```bash
ansible-galaxy collection install -r deploy/ansible/collections/requirements.yml
ansible-playbook deploy/ansible/playbooks/site.yml -i deploy/ansible/inventories/dev --ask-vault-pass
```

Образы `ghcr.io/s95run/cup-api` и `ghcr.io/s95run/cup-web`. Тег задаёт `CUP_IMAGE_TAG` (по умолчанию `latest`).

## Интеграция с s95.ru

Требуется PR в [Sat_9am_5km](https://github.com/vol1ura/Sat_9am_5km): `/cup_sso` и `/api/cup/*` (Bearer token).

См. [AGENTS.md](AGENTS.md) для агентов и CI.