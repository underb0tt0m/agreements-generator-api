# Agreements Generator API

Сервис для асинхронной генерации договоров (DOCX) на основе данных из Excel и DOCX-шаблонов.

---

## Запуск

```bash
docker-compose up
```

Сервер доступен: `http://localhost:8080`

---

## API

| Метод | Эндпоинт | Описание |
|-------|----------|----------|
| POST | `/bulk_generate` | Загрузить архив → получить `job_id` |
| GET | `/get_job_status?id={job_id}` | Статус задачи |
| GET | `/get_archive_info?id={job_id}` | Ошибки и количество документов |
| GET | `/get_archive?id={job_id}` | Скачать архив |

### Пример

```bash
# Создать задачу
JOB_ID=$(curl -s -X POST http://localhost:8080/bulk_generate \
  -F "archive=@test.zip" | jq -r '.job_id')

# Проверить статус
curl "http://localhost:8080/get_job_status?id=$JOB_ID"

# Скачать результат
curl "http://localhost:8080/get_archive?id=$JOB_ID" --output output.zip
```

---

## Конфиг

`config/local.yaml`

```yaml
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
```

---

## Структура

```
.
├── cmd/            # точка входа
├── internal/       # бизнес-логика (API, service, storage)
├── worker/         # Python-воркер (submodule)
├── config/         # конфиги
├── proto/          # контракт
└── docker-compose.yaml
```

---

## Зависимости

- Go 1.26+
- Python 3.13+
- Docker (опционально)
