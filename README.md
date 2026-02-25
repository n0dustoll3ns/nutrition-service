# Nutrition Service

Сервис для отслеживания питания с полноценной системой авторизации и дневником питания.

## Основная функциональность

### ✅ Система авторизации (полностью реализована)

- **Регистрация пользователей** - валидация email, проверка сложности пароля
- **Вход по email/паролю** - JWT токены, обновление через refresh токены
- **Выход из системы** - инвалидация токенов
- **Восстановление пароля** - токены сброса пароля
- **JWT аутентификация** - middleware с реальной валидацией токенов
- **Audit logging** - запись действий пользователей
- **Rate limiting** - защита от брутфорс атак

### ✅ Дневник питания

- Добавление записей о потребленных продуктах
- Просмотр истории по дням
- Суммарная информация по нутриентам
- Копирование записей между днями

### ✅ База данных продуктов USDA

- 365 продуктов с полной нутриентной информацией
- Поиск по описанию продуктов
- Автоматический импорт при запуске

### ✅ Технологии

- **Go 1.23+** - основной язык
- **PostgreSQL** - основная база данных  
- **Gin** - веб-фреймворк
- **JWT** - аутентификация с access/refresh токенами
- **Docker** - контейнеризация
- **Viper** - управление конфигурацией

## Быстрый старт

### Запуск с Docker (рекомендуется)

1. Клонируйте репозиторий:

```bash
git clone <repository-url>
cd nutrition-service
```

1. Запустите сервисы:

```bash

```

1. Сервис будет доступен по адресу:

<http://localhost:8080>

1. Проверьте статус:

```bash
docker-compose ps
```

### Тестирование API

1. Проверка здоровья сервиса:

```bash
curl http://localhost:8080/health
```

1. Регистрация пользователя:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test123!@#","first_name":"Test","last_name":"User"}'
```

1. Вход в систему:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test123!@#"}'
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
  host: "host.docker.internal"  # для Docker Compose
  port: 5433                    # порт для доступа из контейнера
  user: "postgres"
  password: "postgres"
  dbname: "nutrition_db"
  schema: "auth"

jwt:
  access_token_secret: "your-secret-key-change-in-production"
  refresh_token_secret: "your-refresh-secret-key-change-in-production"
  access_token_expiry: "15m"
  refresh_token_expiry: "7d"

security:
  bcrypt_cost: 12
  password_min_length: 8
  require_special_char: true
  require_number: true
  require_uppercase: true
```

## Безопасность

- Пароли хешируются с использованием bcrypt
- JWT токены подписываются с использованием HMAC-SHA256
- Access токены имеют короткое время жизни (15 минут)
- Refresh токены хранятся в базе данных и могут быть отозваны
- Реализована защита от brute-force атак (rate limiting)
- Все endpoints требуют HTTPS в production

## Дневник питания (Food Diary)

Реализована функциональность для ведения дневника питания пользователей. Пользователи могут добавлять записи о потребленных продуктах, просматривать историю, получать суммарную информацию по дням и копировать записи между днями.

**Примечание:** Для доступа к endpoints дневника требуется JWT аутентификация.

### Endpoints дневника

Все endpoints требуют аутентификации (заголовок `Authorization: Bearer <token>`).

#### Получение записей дневника

**Endpoint:** `GET /api/v1/protected/diary/entries`

**Параметры:**

- `date` (обязательный) - дата в формате YYYY-MM-DD
- `daysCount` (опционально, по умолчанию 1) - количество дней для выборки (1-7)

**Пример запроса:**

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/protected/diary/entries?date=2024-01-27&daysCount=1"
```

#### Создание записи о продукте

**Endpoint:** `POST /api/v1/protected/diary/entries`

**Тело запроса (JSON):**

```json
{
  "fdc_id": 747429,                    // ID продукта из базы USDA (опционально, если используется custom_food_name)
  "custom_food_name": "Apple",         // Название пользовательского продукта (опционально, если используется fdc_id)
  "quantity": 1,                       // Количество
  "unit": "piece",                     // Единица измерения (g, ml, piece, cup, tbsp, tsp)
  "amount_grams": 150,                 // Вес в граммах (обязательно)
  "meal_type": "breakfast",            // Тип приема пищи: breakfast, lunch, dinner, snack
  "date": "2024-01-27",                // Дата в формате YYYY-MM-DD
  "notes": "Fresh apple"               // Дополнительные заметки (опционально)
}
```

**Примечание:** Должен быть указан либо `fdc_id` (для продуктов из базы USDA), либо `custom_food_name` (для пользовательских продуктов).

#### Обновление записи

**Endpoint:** `PUT /api/v1/protected/diary/entries/:id`

**Параметры:**

- `id` (в пути) - UUID записи дневника

**Тело запроса:** аналогично созданию записи

#### Удаление записи

**Endpoint:** `DELETE /api/v1/protected/diary/entries/:id`

**Параметры:**

- `id` (в пути) - UUID записи дневника

#### Получение суммарной информации

**Endpoint:** `GET /api/v1/protected/diary/summary`

**Параметры:**

- `date` (обязательный) - дата в формате YYYY-MM-DD
- `daysCount` (опционально, по умолчанию 1) - количество дней для агрегации (1-30)

Возвращает суммарные значения калорий, белков, жиров, углеводов и других нутриентов за указанный период.

#### Копирование записей между днями

**Endpoint:** `POST /api/v1/protected/diary/copy`

**Тело запроса (JSON):**

```json
{
  "source_date": "2024-01-27",         // Дата-источник
  "target_date": "2024-01-28",         // Дата-назначение
  "meal_type": "breakfast"             // Тип приема пищи для фильтрации (опционально)
}
```

### Структура базы данных

Создана схема `diary` с таблицей `food_entries`:

- `id` - UUID, первичный ключ
- `user_id` - UUID пользователя (внешний ключ на auth.users)
- `fdc_id` - ID продукта из базы USDA (опционально)
- `custom_food_name` - название пользовательского продукта (опционально)
- `quantity` - количество
- `unit` - единица измерения
- `amount_grams` - вес в граммах
- `meal_type` - тип приема пищи
- `date` - дата потребления
- `notes` - дополнительные заметки
- `created_at`, `updated_at` - временные метки

## API продуктов (Nutrition)

Сервис включает функциональность для работы с данными о продуктах питания из базы данных USDA.

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

``` bash
nutrition-service/
├── cmd/
│   └── server/
│       └── main.go          # Точка входа
├── internal/
│   ├── config/              # Конфигурация
│   ├── handler/             # HTTP обработчики (auth, food, diary)
│   ├── middleware/          # Middleware (аутентификация, логирование, rate limiting)
│   ├── model/               # Модели данных (user, token, food, diary)
│   ├── repository/          # Работа с базой данных
│   └── utils/               # Вспомогательные функции (JWT, password)
├── migrations/              # Миграции базы данных
├── usda_importer/           # Данные USDA
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

## ВАЖНО: Костыль для отладки

**ВНИМАНИЕ:** В текущей версии кода добавлен костыль для отладки - автоматическое создание супер-пользователя с учетными данными:
- Email: `test@test.com`
- Пароль: `test`

**ЭТОТ КОСТЫЛЬ ДОЛЖЕН БЫТЬ УДАЛЕН ПЕРЕД ДЕПЛОЕМ В PRODUCTION!**

Функция `createSuperUserForDebug` в файле `cmd/server/main.go` создает этого пользователя при каждом запуске сервера, если он не существует. Это представляет серьезную угрозу безопасности в production среде.

### Как удалить костыль перед деплоем:

1. Удалите вызов функции `createSuperUserForDebug` из функции `main()` в `cmd/server/main.go`
2. Удалите саму функцию `createSuperUserForDebug` из файла `cmd/server/main.go`
3. Удалите вспомогательную функцию `stringPtr` если она больше не используется
4. Убедитесь, что пользователь `test@test.com` не существует в production базе данных

## Деплой

### Сборка Docker образа

```bash
docker build -t nutrition-service:latest .
```

### Запуск в production

1. Установите секретные ключи в переменные окружения
2. Настройте SSL/TLS сертификаты
3. Запустите за reverse proxy (nginx, traefik)
4. Настройте мониторинг и логирование
5. **ОБЯЗАТЕЛЬНО УДАЛИТЕ КОСТЫЛЬ С СУПЕР-ПОЛЬЗОВАТЕЛЕМ!**

## Лицензия

MIT
