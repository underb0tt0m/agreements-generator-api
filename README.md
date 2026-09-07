# Agreements Generator API

Сервис для генерации договоров в формате DOCX из Excel-файлов и DOCX-шаблонов.

Go-приложение предоставляет HTTP API, хранит состояние задач в PostgreSQL/Redis и поддерживает два способа взаимодействия с Python-воркером:

- **gRPC** — Go-сервис напрямую передаёт архив воркеру по gRPC и получает результат обратно.
- **RabbitMQ** — Go-сервис сохраняет входной архив, публикует задачу в очередь, а Python-воркер обрабатывает её асинхронно.

Режим выбирается через `execution_mode: grpc` или `execution_mode: queue`.

---

## Архитектура

### gRPC

```text
Client
  |
  v
Go HTTP API
  |
  +--> PostgreSQL / Redis
  |
  v
gRPC
  |
  v
Python Worker
  |
  v
document generation
  |
  v
gRPC response
  |
  v
Go -> PostgreSQL / Redis
```

### RabbitMQ

```text
Client
  |
  v
Go HTTP API
  |
  +--> PostgreSQL (job + input archive)
  |
  v
RabbitMQ
  |
  v
Python Consumer
  |
  +--> PostgreSQL (read input archive)
  |
  v
document generation
  |
  +--> PostgreSQL (result)
  +--> Redis (status)
```

---

## Технологии

### Go API

- Go
- chi
- pgx / pgxpool
- Redis
- RabbitMQ
- gRPC
- JWT
- Prometheus

### Python worker

- Python
- gRPC
- RabbitMQ / pika
- psycopg2 connection pool
- Redis
- Prometheus client

### Infrastructure

- PostgreSQL
- Redis
- RabbitMQ
- Docker Compose
- Prometheus
- Grafana
- k6

---

## Запуск

### Требования

- Go 1.26+
- Python 3.13+
- Docker & Docker Compose
- k6 — только для нагрузочного тестирования

### Клонирование

Python-воркер подключён как Git submodule, поэтому рекомендуется клонировать проект так:

```bash
git clone --recurse-submodules https://github.com/underb0tt0m/agreements-generator-api.git
cd agreements-generator-api
```

Если репозиторий уже склонирован:

```bash
git submodule update --init --recursive
```

### Быстрый старт

```bash
docker compose up --build
```

После запуска:

- API: `http://localhost:8080`
- API metrics: `http://localhost:8080/metrics`
- Worker metrics: `http://localhost:8001/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

Для локальной Grafana используются:

```text
login: admin
password: admin
```

Prometheus datasource внутри Docker Compose:

```text
http://prometheus:9090
```

### Локальная разработка

1. Запустите Python-воркер:

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

Основные конфиги:

- `config/server_local.yaml` — Go API.
- `config/worker_local.yaml` — Python worker.

### Go server

Пример:

```yaml
execution_mode: "queue" # grpc | queue

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
  job_max_duration: "2m"

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

### Worker

В gRPC-режиме количество worker threads настраивается через:

```yaml
execution_mode: "grpc"

max_workers: 1
```

Для PostgreSQL worker использует thread-safe connection pool с настраиваемым `max_conn`.

### Environment variables

Пароли и секреты загружаются из `.env`:

```env
DB_PASSWORD=password
REDIS_PASSWORD=password
RABBITMQ_PASSWORD=password
JWT_SECRET=password
```

---

## API

JWT требуется для рабочих endpoint'ов. `/health` и `/metrics` используются как служебные endpoint'ы.

| Метод | Endpoint | Описание |
|---|---|---|
| POST | `/auth/register` | Регистрация пользователя |
| POST | `/auth/login` | Авторизация и получение JWT |
| POST | `/bulk_generate` | Отправить ZIP-архив и получить `job_id` |
| GET | `/get_job_status?id={job_id}` | Получить статус задачи |
| GET | `/get_archive_info?id={job_id}` | Получить ошибки и количество созданных документов |
| GET | `/get_archive?id={job_id}` | Скачать результирующий ZIP |
| GET | `/health` | Healthcheck |
| GET | `/metrics` | Prometheus metrics |

---

## Monitoring

Prometheus собирает метрики отдельно с Go API и Python worker.

### Основные HTTP-метрики

```text
http_requests_total
http_request_duration_seconds
http_requests_in_flight
```

### Метрики Go-сервиса

```text
generator_jobs_submitted_total
generator_job_submission_duration_seconds
generator_jobs_in_progress
generator_job_processing_duration_seconds
```

### Метрики Python worker

```text
generator_worker_jobs_in_progress
generator_worker_job_processing_duration_seconds
generator_worker_jobs_processed_total
generator_worker_generation_duration_seconds
generator_queue_wait_duration_seconds
```

`generator_queue_wait_duration_seconds` используется только для RabbitMQ-режима и показывает время между публикацией сообщения и началом его обработки consumer'ом.

---

## Нагрузочное тестирование

Для тестов используется k6.

В репозитории есть три сценария:

```text
loadtest/
├── smoke.js
├── load.js
├── constant.js
└── fixtures/
    └── test.zip
```

### Smoke test

Проверяет полный сценарий:

```text
POST /bulk_generate
        ↓
polling /get_job_status
        ↓
completed / failed
```

Запуск:

```bash
make loadtest_smoke
```

### Stress test

Постепенно увеличивает интенсивность примерно от `2` до `20 jobs/s`:

```bash
make loadtest_load
```

### Constant-rate test

Запуск с фиксированной интенсивностью:

```bash
make loadtest_constant RATE=4 DURATION=60s
```

Для каждого теста измеряется **Time To Result** — время от отправки `/bulk_generate` до получения финального статуса задачи.

---

## Результаты нагрузочного тестирования

Для корректного сравнения обе реализации тестировались при одинаковой concurrency:

```text
gRPC:      1 worker thread
RabbitMQ:  1 consumer
```

Условия:

- одинаковый тестовый ZIP;
- одинаковый Go API;
- одинаковые PostgreSQL и Redis;
- один и тот же компьютер;
- constant arrival rate;
- длительность каждого теста — 60 секунд;
- отсутствие HTTP- и job-ошибок во всех приведённых ниже прогонах.

### Constant-rate benchmark

| Target rate | gRPC TTR avg | gRPC TTR p95 | RabbitMQ TTR avg | RabbitMQ TTR p95 |
|---:|---:|---:|---:|---:|
| 2 jobs/s | 216 ms | 239 ms | 219 ms | 224 ms |
| 4 jobs/s | 208 ms | 209 ms | 210 ms | 214 ms |
| 6 jobs/s | 638 ms | 3.04 s | 208 ms | 212 ms |
| 8 jobs/s | 14.37 s | 24.01 s | 8.67 s | 21.67 s |

### Наблюдения

При небольшой нагрузке (`2–4 jobs/s`) обе реализации показывают близкую end-to-end latency около `0.2 s`.

При `6 jobs/s`:

- RabbitMQ остаётся стабильным: `p95 ≈ 212 ms`;
- у gRPC начинает расти tail latency: `p95 ≈ 3.04 s`.

При `8 jobs/s` обе реализации уже работают выше устойчивой пропускной способности:

- gRPC: `p95 ≈ 24.01 s`;
- RabbitMQ: `p95 ≈ 21.67 s`.

При перегрузке RabbitMQ сохраняет стабильное время непосредственной обработки одной job, но растёт `queue wait`: лишние задачи накапливаются в broker.

В gRPC-режиме при одном worker сама генерация также остаётся быстрой, а задержка возникает из-за ожидания свободного gRPC worker.

> Эти результаты относятся к конкретной реализации и тестовой среде проекта. Они не являются общим benchmark gRPC против RabbitMQ.

---

## Структура проекта

```text
.
├── cmd/
│   └── generator/                # entrypoint Go API
├── config/                       # конфигурация Go-сервера
├── internal/
│   ├── api/                      # HTTP handlers и middleware
│   ├── metrics/                  # Prometheus metrics
│   ├── service/
│   │   ├── grpc_generator/       # gRPC implementation
│   │   └── queue_generator/      # RabbitMQ implementation
│   ├── storage/                  # PostgreSQL / in-memory storage
│   └── ...
├── loadtest/
│   ├── fixtures/
│   ├── smoke.js
│   ├── load.js
│   └── constant.js
├── monitoring/
│   └── prometheus.yml
├── proto/                        # gRPC contracts
├── worker/                       # Python worker (Git submodule)
├── docker-compose.yaml
├── Makefile
└── README.md
```

---

## Docker Compose

`docker-compose.yaml` поднимает:

- **server** — Go HTTP API;
- **worker** — Python worker;
- **db** — PostgreSQL;
- **redis** — status cache;
- **rabbitmq** — message broker;
- **migrations** — database migrations;
- **prometheus** — сбор метрик;
- **grafana** — визуализация метрик.

---

## Режимы работы worker

Worker запускается в одном из двух режимов:

### gRPC

```yaml
execution_mode: "grpc"
max_workers: 1
```

Запускается gRPC server на порту `50051`.

### RabbitMQ

```yaml
execution_mode: "queue"
```

Запускается RabbitMQ consumer.

После изменения режима необходимо пересоздать/restart worker container.

---

## Зависимости

- Go 1.26+
- Python 3.13+
- Docker & Docker Compose
- PostgreSQL
- Redis
- RabbitMQ
- Prometheus
- Grafana
- k6
