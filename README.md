# VPN Monitor — 3X-UI Dashboard

Веб-монитор для VPN-сервера на базе 3X-UI. Показывает в реальном времени:
- Нагрузку ЦП
- Использование ОЗУ
- Скорость входящего/исходящего трафика
- Количество активных клиентов (из 3X-UI API)

Доступ защищён логином/паролем. Графики масштабируемы (зум, панорамирование). Данные хранятся в SQLite и не теряются при перезапуске (по умолчанию 30 дней).

---

## Быстрая установка на сервер

```bash
curl -fsSL https://raw.githubusercontent.com/tiblocko2/3X-UI-Monitor-Server/main/install.sh | sudo bash
```

Или скачать скрипт и запустить интерактивно:

```bash
wget https://raw.githubusercontent.com/tiblocko2/3X-UI-Monitor-Server/main/install.sh
sudo bash install.sh
```

Скрипт в интерактивном режиме запросит:
- Порт (по умолчанию 8080)
- Использовать ли HTTPS (и пути к сертификатам)
- URL, логин и пароль от панели 3X-UI
- Логин и пароль для дашборда

После установки сервис зарегистрируется в systemd и запустится автоматически.

### Удаление

```bash
sudo bash install.sh --uninstall
```

---

## Ручной запуск (без install.sh)

### 1. Без Docker

```bash
# Установить Go 1.21+
sudo apt install golang-go

cd vpn-monitor

# Собрать
go build -o vpn-monitor .

# Запустить (все переменные обязательны)
DASH_USER=admin DASH_PASS=secret \
XUI_URL=http://localhost:2053 XUI_USER=admin XUI_PASS=admin \
./vpn-monitor
```

### 2. Docker Compose

```bash
cp .env.example .env
# Отредактируйте .env — заполните все переменные
docker-compose up -d --build
```

---

## Переменные окружения

| Переменная        | Обязательная | По умолчанию             | Описание                          |
|-------------------|:---:|--------------------------|-----------------------------------|
| `DASH_USER`       | ✅  | —                        | Логин для входа в дашборд         |
| `DASH_PASS`       | ✅  | —                        | Пароль для входа в дашборд        |
| `XUI_URL`         | ✅  | —                        | URL панели 3X-UI (с портом)       |
| `XUI_USER`        | ✅  | —                        | Логин 3X-UI                       |
| `XUI_PASS`        | ✅  | —                        | Пароль 3X-UI                      |
| `PORT`            | —  | `8080`                   | Порт веб-сервера                  |
| `HTTPS_ENABLED`   | —  | `false`                  | Включить TLS                      |
| `TLS_CERT`        | —  | —                        | Путь к fullchain.pem              |
| `TLS_KEY`         | —  | —                        | Путь к privkey.pem                |
| `SESSION_SECRET`  | —  | *(авто)*                 | Секрет сессионных cookie          |
| `DATA_DIR`        | —  | `/var/lib/vpn-monitor`   | Директория SQLite базы данных     |
| `RETENTION_DAYS`  | —  | `30`                     | Сколько дней хранить метрики      |

---

## Как работает

1. Бэкенд на Go каждые **10 секунд** собирает:
   - CPU% через `gopsutil`
   - RAM% через `gopsutil`
   - Скорость сети (байты/сек → Mbps) через `gopsutil`
   - Кол-во клиентов: логинится в 3X-UI, запрашивает онлайн-список

2. Данные хранятся в **SQLite** (`DATA_DIR/metrics.db`) и сохраняются между перезапусками

3. Старые данные автоматически удаляются раз в час согласно `RETENTION_DAYS`

4. Фронтенд каждые 10с запрашивает `/api/data` и обновляет графики

5. Графики: Chart.js + chartjs-plugin-zoom
   - `Ctrl + колёсико` — зум по оси X
   - Перетаскивание — панорамирование
   - Кнопка «СБРОС ЗУМА» — возврат в исходный вид
   - Кнопки периода (15 мин / 1ч / 6ч / 12ч / 24ч / Всё) — фильтрация

---

## Управление сервисом

```bash
systemctl status vpn-monitor       # статус
journalctl -u vpn-monitor -f       # логи в реальном времени
systemctl restart vpn-monitor      # перезапуск
systemctl stop vpn-monitor         # остановка
```

---

## Nginx (опционально, проброс на 80/443)

```nginx
server {
    listen 443 ssl;
    server_name monitor.yourdomain.com;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
