# News Aggregator

Сервис агрегации новостей из различных RSS-источников с возможностью комментирования.

## Требования

- Docker
- Docker Compose
- Git

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
- News Service: http://localhost:8082
- Comments Service: http://localhost:8081
- Censorship Service: http://localhost:8083

## Структура проекта

- `api_gateway/` - API Gateway сервис
- `news_service/` - Сервис новостей
- `comments_service/` - Сервис комментариев
- `censorship_service/` - Сервис цензуры
- `init-multiple-dbs.sh` - Скрипт инициализации баз данных
- `init.sql` - SQL скрипт создания таблиц
- `docker-compose.yml` - Конфигурация Docker Compose

## API Endpoints

### API Gateway (http://localhost:8080)

- `GET /` - Главная страница с документацией
- `GET /api/news` - Список новостей
- `GET /api/news/{id}` - Детали новости
- `POST /api/comments` - Добавление комментария

### News Service (http://localhost:8082)

- `GET /api/news` - Список новостей
- `GET /api/news/{id}` - Детали новости

### Comments Service (http://localhost:8081)

- `GET /api/comments?news_id={id}` - Получение комментариев
- `POST /api/comments` - Добавление комментария

### Censorship Service (http://localhost:8083)

- `POST /api/censor` - Проверка комментария
- `GET /health` - Проверка здоровья сервиса

## Остановка сервисов

```bash
docker-compose down
```

Для полной очистки данных (включая базу данных):
```bash
docker-compose down -v
``` 