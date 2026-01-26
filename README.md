# Auth Service

Сервис авторизации на Go с использованием JWT токенов и PostgreSQL.

## Функциональность

- Регистрация пользователей
- Вход по email/паролю
- Выход (инвалидация токенов)
- Обновление access токенов через refresh токены
- Восстановление пароля
- Защищенные endpoints с JWT аутентификацией

## Технологии

- **Go 1.21+**
- **PostgreSQL** - основная база данных
- **Gin** - веб-фреймворк
- **JWT** - JSON Web Tokens для аутентификации
- **Docker** - контейнеризация
- **Viper** - управление конфигурацией

## Быстрый старт

### Предварительные требования

- Go 1.21 или выше
- Docker и Docker Compose
- PostgreSQL (если запускаете без Docker)

### Установка

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd auth-service
```

2. Установите зависимости:
```bash
go mod download
```

3. Настройте конфигурацию:
```bash
cp config.yaml.example config.yaml
# Отредактируйте config.yaml под свои нужды
```

### Запуск с Docker

1. Запустите сервисы:
```bash
make docker-up
```

2. Сервис будет доступен по адресу: http://localhost:8080

### Запуск без Docker

1. Запустите PostgreSQL:
```bash
# Убедитесь, что PostgreSQL запущен на localhost:5432
```

2. Выполните миграции:
```bash
# TODO: Добавить команду для миграций
```

3. Запустите сервис:
```bash
make run
```

## API Endpoints

### Аутентификация

- `POST /api/v1/auth/register` - Регистрация нового пользователя
- `POST /api/v1/auth/login` - Вход в систему
- `POST /api/v1/auth/logout` - Выход из системы
- `POST /api/v1/auth/refresh` - Обновление access токена
- `POST /api/v1/auth/password-reset-request` - Запрос сброса пароля
- `POST /api/v1/auth/password-reset-confirm` - Подтверждение сброса пароля

### Защищенные endpoints

- `GET /api/v1/protected/me` - Получение информации о текущем пользователе
- `GET /api/v1/protected/foods/search` - Поиск продуктов по описанию
- `GET /api/v1/protected/foods/:id` - Получение продукта по FDC ID

### Системные endpoints

- `GET /health` - Проверка здоровья сервиса

## Конфигурация

Конфигурация задается в файле `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "nutrition_db"
  schema: "auth"

jwt:
  access_token_secret: "your-secret-key"
  refresh_token_secret: "your-refresh-secret-key"
  access_token_expiry: "15m"
  refresh_token_expiry: "7d"
```

## Безопасность

- Пароли хешируются с использованием bcrypt
- JWT токены подписываются с использованием HMAC-SHA256
- Access токены имеют короткое время жизни (15 минут)
- Refresh токены хранятся в базе данных и могут быть отозваны
- Реализована защита от brute-force атак (rate limiting)
- Все endpoints требуют HTTPS в production

## API продуктов (Nutrition)

Сервис теперь включает функциональность для работы с данными о продуктах питания из базы данных USDA.

### Поиск продуктов

**Endpoint:** `GET /api/v1/protected/foods/search`

**Параметры:**
- `q` (обязательный) - строка поиска
- `limit` (опционально, по умолчанию 20) - количество результатов на странице (максимум 100)
- `offset` (опционально, по умолчанию 0) - смещение для пагинации

**Пример запроса:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/protected/foods/search?q=cheese&limit=5"
```

**Ответ:**
```json
{
  "data": [
    {
      "food": {
        "fdc_id": 747429,
        "description": "Cheese, American, restaurant",
        "data_type": "Foundation",
        "food_class": "FinalFood",
        "publication_date": "12/16/2019"
      },
      "nutrients": [
        {
          "id": 8523932,
          "nutrient_name": "Nitrogen",
          "unit_name": "g",
          "amount": 2.74,
          // ... другие поля
        }
        // ... другие питательные вещества
      ]
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 5,
    "total": 19,
    "total_pages": 4
  }
}
```

### Получение продукта по ID

**Endpoint:** `GET /api/v1/protected/foods/:id`

**Параметры:**
- `id` (в пути) - FDC ID продукта

**Пример запроса:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/protected/foods/747429"
```

## Импорт данных USDA

Сервис автоматически импортирует данные из USDA JSON файла при запуске. Для отключения импорта установите `importer.import_on_startup: false` в конфигурации.

## Структура проекта

```
auth-service/
├── cmd/
│   └── server/
│       └── main.go          # Точка входа
├── internal/
│   ├── config/              # Конфигурация
│   ├── handler/             # HTTP обработчики
│   │   └── food_handler.go  # Обработчики для продуктов
│   ├── middleware/          # Middleware (аутентификация, логирование)
│   ├── model/               # Модели данных
│   │   └── food.go          # Модели для продуктов
│   ├── repository/          # Работа с базой данных
│   │   └── food_repository.go # Репозиторий для продуктов
│   ├── service/             # Бизнес-логика
│   └── utils/               # Вспомогательные функции
├── migrations/              # Миграции базы данных
│   └── 002_create_nutrition_schema.up.sql # Схема для продуктов
├── usda-importer/           # Данные USDA
├── pkg/                     # Публичные пакеты
├── config.yaml              # Конфигурация
├── Dockerfile              # Docker образ
├── docker-compose.yml      # Docker Compose
├── go.mod                  # Зависимости Go
└── README.md               # Документация
```

## Разработка

### Запуск в режиме разработки

```bash
make dev
```

Или напрямую:

```bash
go build ./cmd/server && ./server
```

### Импорт данных

Для ручного импорта данных USDA:

```bash
# Убедитесь, что база данных запущена
# Запустите сервер, импорт произойдет автоматически
```

### Запуск тестов

```bash
make test
```

### Линтинг

```bash
make lint
```

### Форматирование кода

```bash
make fmt
```

## Миграции базы данных

Миграции находятся в директории `migrations/`. Для применения миграций:

```bash
# TODO: Добавить команду для миграций
```

## Деплой

### Сборка Docker образа

```bash
docker build -t auth-service:latest .
```

### Запуск в production

1. Установите секретные ключи в переменные окружения
2. Настройте SSL/TLS сертификаты
3. Запустите за reverse proxy (nginx, traefik)
4. Настройте мониторинг и логирование

## Лицензия

MIT
