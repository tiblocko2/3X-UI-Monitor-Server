# VPN Monitor — 3X-UI Dashboard

Веб-монитор для VPN-сервера на базе 3X-UI. Показывает в реальном времени:
- Нагрузку ЦП
- Использование ОЗУ
- Скорость входящего/исходящего трафика
- Количество активных клиентов (из 3X-UI API)

Доступ защищён логином/паролем. Графики масштабируемы (зум, панорамирование).

---

## Быстрый старт

### 1. Без Docker (прямо на сервере)

```bash
# Установить Go 1.21+
sudo apt install golang-go  # или через официальный сайт

# Клонировать / распаковать проект
cd vpn-monitor

# Установить зависимости и собрать
make deps
make build

# Запустить
XUI_URL=http://localhost:2053 XUI_USER=admin XUI_PASS=admin ./vpn-monitor
```

Сервис запустится на `http://0.0.0.0:8080`.

---

### 2. С Docker

```bash
make docker

docker run -d \
  --name vpn-monitor \
  --restart unless-stopped \
  --network host \
  -e XUI_URL=http://localhost:2053 \
  -e XUI_USER=admin \
  -e XUI_PASS=admin \
  vpn-monitor
```

---

### 3. Docker Compose

```bash
# Отредактируйте docker-compose.yml — укажите правильные XUI_USER и XUI_PASS
docker-compose up -d --build
```

---

## Переменные окружения

| Переменная | По умолчанию             | Описание                              |
|------------|--------------------------|---------------------------------------|
| `PORT`     | `8080`                   | Порт веб-сервера мониторинга          |
| `XUI_URL`  | `http://localhost:2053`  | URL панели 3X-UI                      |
| `XUI_USER` | `admin`                  | Логин 3X-UI                           |
| `XUI_PASS` | `admin`                  | Пароль 3X-UI                          |

---

## Авторизация в мониторе

- **Логин:** `Akvil0n`
- **Пароль:** `Perfect10nizm`

---

## Как работает

1. Бэкенд на Go каждые **10 секунд** собирает:
   - CPU% через `gopsutil`
   - RAM% через `gopsutil`
   - Скорость сети (байты/сек → MB/s) через `gopsutil`
   - Кол-во клиентов: логинится в 3X-UI, запрашивает `/xui/API/inbounds`

2. Данные хранятся в памяти (кольцевой буфер, последние 1440 точек ≈ 24ч)

3. Фронтенд каждые 10с запрашивает `/api/data` и обновляет графики

4. Графики: Chart.js + chartjs-plugin-zoom
   - `Ctrl + колёсико` — зум по оси X
   - Перетаскивание — панорамирование
   - Кнопка «СБРОС ЗУМА» — возврат в исходный вид
   - Кнопки периода (15 мин / 1ч / 6ч / 12ч / 24ч / Всё) — фильтрация данных

5. Под каждым графиком — сводка за выбранный период: мин / ср / макс

---

## Настройка systemd (автозапуск без Docker)

```ini
# /etc/systemd/system/vpn-monitor.service
[Unit]
Description=VPN Monitor
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vpn-monitor
ExecStart=/opt/vpn-monitor/vpn-monitor
Restart=always
Environment=PORT=8080
Environment=XUI_URL=http://localhost:2053
Environment=XUI_USER=admin
Environment=XUI_PASS=admin

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable vpn-monitor
sudo systemctl start vpn-monitor
```

---

## Nginx (опционально, если нужен 80/443 порт)

```nginx
server {
    listen 80;
    server_name monitor.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
