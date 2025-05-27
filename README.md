# News Aggregator

Сервис агрегации новостей из различных RSS-источников с возможностью комментирования и цензуры комментариев.

## Требования

### Системные требования
- Docker 20.10+
- Docker Compose 2.0+
- Git
- Минимум 2GB RAM
- 10GB свободного места на диске

### Опциональные инструменты
- Postman (для тестирования API)
- pgAdmin или другой PostgreSQL клиент (для работы с базой данных)

## Установка и запуск

1. Клонируйте репозиторий:
```bash
git clone https://github.com/your-username/news_aggregator.git
cd news_aggregator
```

2. Запустите сервисы с помощью Docker Compose:
```bash
docker-compose up -d
```

Сервисы будут доступны по следующим адресам:
- API Gateway: http://localhost:8080
- News Service: http://localhost:8080 (внутренний порт)
- Comments Service: http://localhost:8081
- Censorship Service: http://localhost:8083
- PostgreSQL: localhost:5432

## Конфигурация

### Переменные окружения
Основные переменные окружения для каждого сервиса:

#### API Gateway
- `NEWS_SERVICE_URL` - URL сервиса новостей
- `COMMENTS_SERVICE_URL` - URL сервиса комментариев
- `CENSORSHIP_SERVICE_URL` - URL сервиса цензуры
- `LOG_LEVEL` - Уровень логирования (info/debug/error)

#### News Service
- `DATABASE_URL` - URL подключения к базе данных
- `LOG_LEVEL` - Уровень логирования

#### Comments Service
- `DB_HOST` - Хост базы данных
- `DB_PORT` - Порт базы данных
- `DB_USER` - Пользователь базы данных
- `DB_PASSWORD` - Пароль базы данных
- `DB_NAME` - Имя базы данных
- `LOG_LEVEL` - Уровень логирования

### Конфигурация RSS-источников
Файл `config.json` содержит настройки RSS-источников:
```json
{
    "rss_feeds": [
        "https://tass.ru/rss/v2.xml",
        "https://www.kommersant.ru/RSS/news.xml",
        "https://lenta.ru/rss",
        "https://news.un.org/feed/subscribe/ru/news/all/rss.xml",
        "https://www.ria.ru/export/rss2/archive/index.xml",
        "https://www.5-tv.ru/news/rss/"
    ],
    "poll_interval": 5
}
```

## Мониторинг и логи

### Логи
Логи сервисов доступны в следующих директориях:
- API Gateway: `./logs/api_gateway/`
- News Service: `./logs/news_service/`
- Comments Service: `./logs/comments_service/`
- Censorship Service: `./logs/censorship_service/`
- PostgreSQL: `./logs/postgres/`

Просмотр логов:
```bash
# Просмотр логов конкретного сервиса
docker-compose logs -f [service_name]

# Просмотр всех логов
docker-compose logs -f
```

### Метрики
Сервисы предоставляют метрики Prometheus:
- HTTP-метрики (latency, requests, status codes)
- Метрики базы данных
- Метрики RSS-агрегации

## API Endpoints

### API Gateway (http://localhost:8080)

- `GET /` - Главная страница с документацией
- `GET /api/news` - Список новостей
  - Query параметры:
    - `page` - номер страницы (по умолчанию 1)
    - `s` - поиск по заголовку
- `GET /api/news/{id}` - Детали новости
- `POST /api/comments` - Добавление комментария
  - Body: `{"news_id": 1, "text": "Текст комментария"}`

### News Service (http://localhost:8080)

- `GET /api/news` - Список новостей
- `GET /api/news/{id}` - Детали новости
- `GET /health` - Проверка здоровья сервиса

### Comments Service (http://localhost:8081)

- `GET /api/comments?news_id={id}` - Получение комментариев
- `POST /api/comments` - Добавление комментария
- `GET /health` - Проверка здоровья сервиса

### Censorship Service (http://localhost:8083)

- `POST /api/censor` - Проверка комментария
  - Body: `{"text": "Текст для проверки"}`
- `GET /health` - Проверка здоровья сервиса

## Тестирование

### Postman
В репозитории есть готовая коллекция с примерами запросов:
1. Откройте Postman
2. Импортируйте коллекцию:
   - Нажмите "Import"
   - Выберите файл `news_aggregator_api_gateway.postman_collection.json`

### Unit-тесты
Запуск тестов:
```bash
# Запуск всех тестов
go test ./...

# Запуск тестов конкретного сервиса
cd news_service && go test ./...
```

## Разработка

### Структура проекта
- `api_gateway/` - API Gateway сервис
- `news_service/` - Сервис новостей
- `comments_service/` - Сервис комментариев
- `censorship_service/` - Сервис цензуры
- `init-multiple-dbs.sh` - Скрипт инициализации баз данных
- `init.sql` - SQL скрипт создания таблиц
- `docker-compose.yml` - Конфигурация Docker Compose
- `config.json` - Конфигурация RSS-источников

### Запуск в режиме разработки
```bash
# Запуск с пересборкой при изменении кода
docker-compose up --build

# Запуск конкретного сервиса
docker-compose up --build [service_name]
```

## Остановка сервисов

```bash
# Остановка сервисов
docker-compose down

# Остановка с удалением данных
docker-compose down -v

# Остановка с удалением образов
docker-compose down --rmi all
```

## Лицензия

MIT License. См. файл LICENSE для подробностей. 