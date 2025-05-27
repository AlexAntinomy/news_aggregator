# News Aggregator

Сервис агрегации новостей из различных RSS-источников с возможностью комментирования и цензуры комментариев.

## Требования

- Docker 20.10+
- Docker Compose 2.0+
- Git
- 2GB RAM
- 10GB свободного места

## Установка

```bash
git clone https://github.com/AlexAntinomy/news_aggregator.git
cd news_aggregator
docker-compose up -d
```

## Доступ к сервисам

- API Gateway: http://localhost:8080
- News Service: http://localhost:8080 (внутренний порт)
- Comments Service: http://localhost:8081
- Censorship Service: http://localhost:8083
- PostgreSQL: localhost:5432

## API Endpoints

### API Gateway

- `GET /` - Главная страница
- `GET /api/news` - Список новостей
  - `?page=1` - пагинация
  - `?s=query` - поиск
- `GET /api/news/{id}` - Детали новости
- `POST /api/comments` - Добавление комментария
  ```json
  {
    "news_id": 1,
    "text": "Текст комментария"
  }
  ```

### Comments Service

- `GET /api/comments?news_id={id}` - Комментарии к новости
- `POST /api/comments` - Добавление комментария

### Censorship Service

- `POST /api/censor` - Проверка комментария
  ```json
  {
    "text": "Текст для проверки"
  }
  ```

## Логи

```bash
# Просмотр логов сервиса
docker-compose logs -f [service_name]

# Просмотр всех логов
docker-compose logs -f
```

## Управление

```bash
# Запуск
docker-compose up -d

# Остановка
docker-compose down

# Остановка с удалением данных
docker-compose down -v
```

## Тестирование

В репозитории есть коллекция Postman для тестирования API:
`news_aggregator_api_gateway.postman_collection.json`

## Лицензия

MIT License 