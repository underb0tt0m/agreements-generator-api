# Agreements Generator API

Сервис для генерации договоров (DOCX) из Excel и DOCX-шаблонов. Поддерживает два режима работы:

- **gRPC** — Go-сервер передаёт задачу Python-воркеру через gRPC (в фоновой горутине).
- **Очередь (RabbitMQ)** — Go-сервер публикует задачу в очередь, Python-воркер забирает и обрабатывает асинхронно.

Переключение через конфиг (`execution_mode: grpc` или `queue`).

---

## Запуск

### Требования

- Go 1.26+
- Docker & Docker Compose
- Python 3.13+ (воркер)

### Быстрый старт (всё в Docker)

```bash
docker-compose up --build
```

Сервер будет доступен по адресу: `http://localhost:8080`.

### Локальная разработка

1. Запустите Python-воркер (в режиме gRPC или consumer):
   ```bash
   cd worker
   python -m cmd.main
   ```

2. Запустите Go-сервер:
   ```bash
   go run cmd/generator/main.go -config ./config/server_local.yaml
   ```

---

## Конфигурация

Конфиги разделены для сервера и воркера:

- `config/server_local.yaml` — для Go-сервера.
- `config/worker_local.yaml` — для Python-воркера.

Пример `server_local.yaml`:

```yaml
execution_mode: "queue" #grpc | queue

env: "local"

logger:
  type: "zap"
  level: "development"

server:
  port: 8080
  shutdown_duration: "10s"


grpc_client:
  host: "worker"
  port: 50051
  job_max_duration: "30s"

storage:
  job_ttl: "5m"
  type: "postgres"
  driver: "postgres"
  host: "db"
  port: "5432"
  database: "generator"

redis:
  host: "redis"
  port: 6379
  db: 0
  job_status_ttl: "10m"

rabbit_mq:
  host: "rabbitmq"
  port: 5672
  username: "guest"
  vhost: "/"
  queue: "jobs"

jwt:
  jwt_signing_method: "HS256"
  token_ttl: "1h"
  prefix: "Bearer"

security:
  hash_cost: 10

```

Все пароли и секреты загружаются из `.env`:

```env
DB_PASSWORD=postgres
REDIS_PASSWORD=password
RABBITMQ_PASSWORD=password
JWT_SECRET=password
```

---

## API

Все эндпоинты требуют JWT-токен (кроме `/health`).

| Метод | Эндпоинт | Описание |
|-------|----------|----------|
| POST | `/auth/register` | Регистрация пользователя |
| POST | `/auth/login` | Логин, возвращает JWT |
| POST | `/bulk_generate` | Загрузить архив → получить `job_id` |
| GET | `/get_job_status?id={job_id}` | Статус задачи |
| GET | `/get_archive_info?id={job_id}` | Ошибки и количество документов |
| GET | `/get_archive?id={job_id}` | Скачать архив (ZIP) |
| GET | `/health` | Healthcheck |

---

## Структура проекта

```
.
├── cmd/                    # точки входа (Go)
│   └── generator/          # main.go
├── internal/               # бизнес-логика (API, service, storage)
├── proto/                  # gRPC-контракт (generator.proto)
├── worker/                 # Python-воркер (submodule)
│   ├── cmd/                # точки входа Python
│   │   ├── main.py         # единая точка (выбор режима по конфигу)
│   │   ├── grpc_server/    # gRPC-сервер
│   │   └── consumer/       # consumer RabbitMQ
│   ├── internal/           # внутренняя логика Python
│   └── config/             # конфиг воркера
├── config/                 # конфиги для Go-сервера
├── docker-compose.yaml
├── Makefile
└── README.md
```

---

## Docker Compose

Поднимает все сервисы:

- **server** — Go-API
- **worker** — Python-воркер
- **db** — PostgreSQL
- **redis** — кэш статусов
- **rabbitmq** — очередь задач
- **migrations** — применяет миграции при старте

---

## Режимы работы воркера

Воркер может запускаться в двух режимах (управляется через `execution_mode` в `config/worker_local.yaml`):

- `grpc` — запускает gRPC-сервер (порт 50051).
- `queue` — запускает consumer RabbitMQ.

При переключении режима достаточно перезапустить контейнер воркера.

---

## Зависимости

- Go 1.26+
- Python 3.13+
- Docker & Docker Compose
- PostgreSQL 18
- Redis 7
- RabbitMQ 3