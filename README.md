
# Инструкция по использованию

## 1. Подготовить Telegram

1. Создать бота через `@BotFather` и сохранить token в `TELEGRAM_BOT_TOKEN`.
2. Добавить бота в Telegram-чат или канал, куда приходят build-уведомления.
3. Получить chat id и сохранить его в `TELEGRAM_CHAT_ID`.

### Самый надежный способ для группы

1. Добавить бота в нужную группу.
2. Написать в группе любое сообщение, например `/start` или `test`.
3. Выполнить запрос:

```bash
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"
```

1. Найти в ответе объект `message.chat.id`:

```json
{
  "message": {
    "chat": {
      "id": -123456789,
      "title": "SOTBI releases",
      "type": "group"
    }
  }
}
```

1. Сохранить значение `id` целиком:

```bash
export TELEGRAM_CHAT_ID=-123456789
```

Для супергрупп и каналов `chat_id` обычно начинается с `-100`, например `-1001234567890`. Минус является частью id, его нужно сохранять. Если бот добавлен в канал, ему нужны права на отправку сообщений; для private channel проще получить id через `getUpdates`, переслав или написав сообщение так, чтобы бот получил update.

Если `getUpdates` возвращает пустой `result`, проверить:

- бот действительно добавлен в чат;
- после добавления бота в чат было отправлено новое сообщение;
- у бота не включен webhook. Если включен, временно удалить его:

```bash
curl --request POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/deleteWebhook"
```

1. Получить numeric Telegram user id пользователей, которым разрешен релиз, и перечислить их через запятую в `TELEGRAM_ALLOWED_USER_IDS`, например `123456789,987654321`.

User id - это id конкретного Telegram-аккаунта, не username и не `@login`. Его можно получить несколькими способами.

### Способ A: через `getUpdates` у release bot

1. Пользователь, которому нужно разрешить релиз, пишет боту в личку любое сообщение, например `/start`.
2. Выполнить:

```bash
curl "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"
```

1. Найти `message.from.id`:

```json
{
  "message": {
    "from": {
      "id": 123456789,
      "is_bot": false,
      "first_name": "Ivan",
      "username": "ivan"
    }
  }
}
```

1. Добавить этот id в `TELEGRAM_ALLOWED_USER_IDS`:

```bash
export TELEGRAM_ALLOWED_USER_IDS=123456789,987654321
```

### Способ B: через callback от кнопки

Если бот уже отправил release-сообщение, пользователь может нажать кнопку. В Telegram update id будет лежать в `callback_query.from.id`. Это тот же numeric user id, который нужно добавить в whitelist.

### Способ C: через ботов-помощников

Пользователь может открыть, например, `@userinfobot` или аналогичный бот и получить свой numeric id. В whitelist нужно добавлять именно число, а не username.

Важно различать:

- `TELEGRAM_CHAT_ID` - куда бот отправляет release-сообщения; это группа, канал или личный чат.
- `TELEGRAM_ALLOWED_USER_IDS` - кто имеет право нажимать release-кнопки; это id пользователей.
- `TELEGRAM_CHAT_ID` может быть отрицательным, например `-1001234567890`.
- `TELEGRAM_ALLOWED_USER_IDS` обычно положительные числа, например `123456789`.

Важно: whitelist проверяется на каждом callback. Если пользователь видит сообщение, но его id не указан в `TELEGRAM_ALLOWED_USER_IDS`, кнопки релиза для него не сработают.

Важно: whitelist проверяется на каждом callback. Если пользователь видит сообщение, но его id не указан в `TELEGRAM_ALLOWED_USER_IDS`, кнопки релиза для него не сработают.

## 2. Подготовить GitHub token

1. Создать `GITHUB_TOKEN` для бота: fine-grained PAT или GitHub App token.
2. Дать token доступ к сервисным репозиториям, в которых нужно запускать `.github/workflows/restart.yaml`.
3. Выдать права на запуск GitHub Actions workflow dispatch. Для fine-grained PAT нужны permissions `Actions: Read and write` и доступ к нужным repositories.
4. Проверить, что в каждом сервисном репозитории есть `.github/workflows/restart.yaml` с `workflow_dispatch` inputs `tag` и `environment`.

Бот запускает workflow в том репозитории, который пришел в build notification как `repository`. Поэтому token должен иметь права именно на этот сервисный репозиторий, а не только на [`/Users/bazys/Projects/sotbi/gh-actions`](file:///Users/bazys/Projects/sotbi/gh-actions).

## 3. Настроить секрет для build workflow

1. Сгенерировать случайный shared secret, например:

```bash
openssl rand -hex 32
```

1. Передать это значение боту в `RELEASEBOT_SHARED_SECRET`.
2. Добавить такое же значение в GitHub Secrets сервисных репозиториев как `RELEASEBOT_SHARED_SECRET`.
3. Добавить публичный URL бота в GitHub Secrets сервисных репозиториев как `RELEASEBOT_URL`, например `https://releasebot.example.com`.

Workflow [`/Users/bazys/Projects/sotbi/gh-actions/.github/workflows/build-image.yaml`](file:///Users/bazys/Projects/sotbi/gh-actions/.github/workflows/build-image.yaml) отправляет успешный build в `${RELEASEBOT_URL}/build-notifications` с заголовком `X-Releasebot-Secret`. Если `RELEASEBOT_URL` или `RELEASEBOT_SHARED_SECRET` не заданы, workflow использует старое plain Telegram-уведомление.

## 4. Запустить бота локально для проверки

В репозитории [`/Users/bazys/Projects/sotbi/gh-actions`](file:///Users/bazys/Projects/sotbi/gh-actions):

```bash
export RELEASEBOT_HTTP_ADDR=:8080
export TELEGRAM_BOT_TOKEN=...
export TELEGRAM_CHAT_ID=...
export TELEGRAM_ALLOWED_USER_IDS=123456789
export GITHUB_TOKEN=...
export RELEASEBOT_SHARED_SECRET=...
export RELEASEBOT_LONG_POLLING=true

go test ./...
go run ./cmd/releasebot
```

Проверить healthcheck:

```bash
curl -i http://localhost:8080/healthz
```

Отправить тестовое build notification:

```bash
curl --fail-with-body \
  --request POST http://localhost:8080/build-notifications \
  --header 'Content-Type: application/json' \
  --header "X-Releasebot-Secret: ${RELEASEBOT_SHARED_SECRET}" \
  --data '{
    "repository": "SOTBI-LLC/service",
    "ref": "main",
    "branch": "main",
    "sha": "abc123",
    "tag": "v1.2.3",
    "actor": "octocat",
    "commit_message": "test release",
    "run_url": "https://github.com/SOTBI-LLC/service/actions/runs/1"
  }'
```

После успешного запроса бот должен отправить Telegram-сообщение с inline-кнопкой `release`.

## 5. Выбрать режим получения Telegram callbacks

Вариант A: long polling.

Использовать, если бот работает в приватной сети или нет публичного HTTPS endpoint для Telegram:

```bash
export RELEASEBOT_LONG_POLLING=true
```

В этом режиме Telegram webhook настраивать не нужно. Бот сам читает updates через Telegram API.

Вариант B: webhook.

Использовать, если бот доступен Telegram по публичному HTTPS URL. Переменные:

```bash
export TELEGRAM_WEBHOOK_SECRET_TOKEN=...
unset RELEASEBOT_LONG_POLLING
```

Зарегистрировать webhook в Telegram:

```bash
curl --request POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  --form "url=https://releasebot.example.com/telegram/webhook" \
  --form "secret_token=${TELEGRAM_WEBHOOK_SECRET_TOKEN}"
```

Telegram будет отправлять callbacks на `/telegram/webhook`, а бот проверит заголовок `X-Telegram-Bot-Api-Secret-Token`.

## 6. Подключить сервисные репозитории

Для каждого сервиса, который использует reusable [`/Users/bazys/Projects/sotbi/gh-actions/.github/workflows/build-image.yaml`](file:///Users/bazys/Projects/sotbi/gh-actions/.github/workflows/build-image.yaml):

1. Добавить secrets `RELEASEBOT_URL` и `RELEASEBOT_SHARED_SECRET`.
2. Оставить существующие `TELEGRAM_CHANNEL` и `TELEGRAM_TOKEN`, потому что они используются для failure-уведомлений и fallback при недоступном release bot.
3. Убедиться, что build workflow передает корректный `inputs.tag`.
4. Убедиться, что `.github/workflows/restart.yaml` в сервисном репозитории принимает `workflow_dispatch` с `tag` и `environment`.

## 7. Ежедневный сценарий релиза

1. Разработчик запускает build image workflow или workflow запускается по существующим правилам.
2. После успешной сборки reusable workflow отправляет payload в release bot.
3. Бот публикует Telegram-сообщение с контекстом сборки: repository, branch/ref, tag, actor, commit message и ссылка на GitHub Actions run.
4. Разрешенный пользователь нажимает `release`.
5. Бот заменяет клавиатуру на выбор `dev` / `prod`.
6. Пользователь выбирает окружение.
7. Бот запускает `workflow_dispatch` для `.github/workflows/restart.yaml` в сервисном репозитории с inputs `{ tag, environment }`.
8. В Telegram callback появляется ответ `Deploy started for dev` или `Deploy started for prod`.
9. Дальше результат деплоя смотреть в GitHub Actions сервисного репозитория.

## 8. Проверка и troubleshooting

- Если build прошел успешно, но кнопки нет, проверить secrets `RELEASEBOT_URL` и `RELEASEBOT_SHARED_SECRET` в сервисном репозитории. При их отсутствии workflow уйдет в fallback и отправит обычное Telegram-сообщение.
- Если `/build-notifications` возвращает `401`, значения `RELEASEBOT_SHARED_SECRET` у workflow и бота не совпадают.
- Если пользователь нажимает кнопку и получает отказ, проверить его numeric id в `TELEGRAM_ALLOWED_USER_IDS`.
- Если dispatch не запускается, проверить права `GITHUB_TOKEN`, наличие `.github/workflows/restart.yaml` в сервисном репозитории и корректность `repository` в payload.
- Если webhook callbacks не доходят, проверить публичный HTTPS URL, регистрацию `setWebhook` и совпадение `TELEGRAM_WEBHOOK_SECRET_TOKEN`.
- Если бот запущен через long polling, убедиться, что для этого token не висит старый webhook. При необходимости удалить его:

```bash
curl --request POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/deleteWebhook"
```
