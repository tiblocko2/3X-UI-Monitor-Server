#!/usr/bin/env bash
# install.sh — интерактивный установщик vpn-monitor
# Использование: bash install.sh [--uninstall]
set -euo pipefail

INSTALL_DIR="/opt/vpn-monitor"
DATA_DIR="/var/lib/vpn-monitor"
CONFIG_DIR="/etc/vpn-monitor"
SERVICE_FILE="/etc/systemd/system/vpn-monitor.service"
ENV_FILE="$CONFIG_DIR/vpn-monitor.env"
BINARY="$INSTALL_DIR/vpn-monitor"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
die()     { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ──────────────────────────────────────────────────────────────────────────────
# Удаление
# ──────────────────────────────────────────────────────────────────────────────
uninstall() {
    info "Удаляем vpn-monitor..."
    systemctl stop vpn-monitor 2>/dev/null || true
    systemctl disable vpn-monitor 2>/dev/null || true
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    rm -rf "$CONFIG_DIR"
    warn "Данные в $DATA_DIR НЕ удалены. Удалите вручную: rm -rf $DATA_DIR"
    success "vpn-monitor удалён."
    exit 0
}

[[ "${1:-}" == "--uninstall" ]] && uninstall

# ──────────────────────────────────────────────────────────────────────────────
# Проверка окружения
# ──────────────────────────────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && die "Запустите скрипт от root: sudo bash install.sh"
command -v systemctl &>/dev/null || die "systemctl не найден (только systemd-системы)"
command -v curl &>/dev/null || die "curl не найден: apt install curl"

# ──────────────────────────────────────────────────────────────────────────────
# Определяем архитектуру
# ──────────────────────────────────────────────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH_TAG="linux-amd64" ;;
    aarch64) ARCH_TAG="linux-arm64" ;;
    *) die "Неподдерживаемая архитектура: $ARCH" ;;
esac

# ──────────────────────────────────────────────────────────────────────────────
# Вспомогательные функции ввода
# ──────────────────────────────────────────────────────────────────────────────
prompt() {
    # prompt VAR "Вопрос" "default"
    local var="$1" question="$2" default="${3:-}"
    local hint=""
    [[ -n "$default" ]] && hint=" [$default]"
    read -rp "  ${question}${hint}: " value
    [[ -z "$value" ]] && value="$default"
    [[ -z "$value" ]] && die "Значение обязательно: $question"
    printf -v "$var" '%s' "$value"
}

prompt_secret() {
    local var="$1" question="$2"
    local value confirm
    while true; do
        read -rsp "  ${question}: " value; echo
        [[ -z "$value" ]] && { warn "Пароль не может быть пустым."; continue; }
        read -rsp "  Повторите пароль: " confirm; echo
        [[ "$value" == "$confirm" ]] && break
        warn "Пароли не совпадают, попробуйте снова."
    done
    printf -v "$var" '%s' "$value"
}

yn() {
    # yn VAR "Вопрос" "y|n"
    local var="$1" question="$2" default="${3:-n}"
    local hint="y/n"
    [[ "$default" == "y" ]] && hint="Y/n" || hint="y/N"
    read -rp "  ${question} [${hint}]: " value
    [[ -z "$value" ]] && value="$default"
    value="${value,,}"
    [[ "$value" == "y" || "$value" == "yes" ]] && printf -v "$var" 'true' || printf -v "$var" 'false'
}

# ──────────────────────────────────────────────────────────────────────────────
# Интерактивная конфигурация
# ──────────────────────────────────────────────────────────────────────────────
echo
echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║       vpn-monitor — установка            ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
echo

info "Шаг 1: GitHub репозиторий"
prompt GITHUB_REPO "GitHub репозиторий (owner/repo)" "tiblocko2/3X-UI-Monitor-Server"

echo
info "Шаг 2: Сетевые настройки"
prompt APP_PORT "Порт веб-интерфейса" "8080"
yn USE_HTTPS "Включить HTTPS?" "n"

TLS_CERT=""
TLS_KEY=""
if [[ "$USE_HTTPS" == "true" ]]; then
    prompt TLS_CERT "Путь к сертификату (fullchain.pem)" "/etc/letsencrypt/live/your-domain/fullchain.pem"
    prompt TLS_KEY  "Путь к приватному ключу (privkey.pem)" "/etc/letsencrypt/live/your-domain/privkey.pem"
    [[ -f "$TLS_CERT" ]] || warn "Файл $TLS_CERT не найден — убедитесь что он существует перед запуском"
    [[ -f "$TLS_KEY"  ]] || warn "Файл $TLS_KEY не найден — убедитесь что он существует перед запуском"
fi

echo
info "Шаг 3: Подключение к 3X-UI"
while true; do
    prompt XUI_URL  "URL панели 3X-UI (с портом, например http://1.2.3.4:2053)" "http://localhost:2053"
    prompt XUI_USER "Логин 3X-UI" ""
    prompt_secret XUI_PASS "Пароль 3X-UI"

    info "Проверяем подключение к 3X-UI..."
    XUI_LOGIN_BODY=$(curl -s \
        --max-time 10 \
        -X POST "${XUI_URL}/login" \
        -d "username=${XUI_USER}&password=${XUI_PASS}" \
        2>/dev/null || true)

    # 3X-UI возвращает {"success":true} при верных данных
    if echo "$XUI_LOGIN_BODY" | grep -q '"success":true'; then
        success "Подключение к 3X-UI успешно"
        break
    elif [[ -z "$XUI_LOGIN_BODY" ]]; then
        echo
        warn "Сервер не отвечает по адресу ${XUI_URL}"
        warn "Проверьте URL (адрес и порт) и попробуйте снова."
        echo
    else
        echo
        warn "Авторизация в 3X-UI не прошла (неверный логин или пароль)"
        warn "Ответ сервера: ${XUI_LOGIN_BODY:0:120}"
        echo
    fi
done

echo
info "Шаг 4: Учётные данные дашборда"
prompt DASH_USER "Логин для входа в дашборд" ""
prompt_secret DASH_PASS "Пароль для входа в дашборд"

echo
info "Шаг 5: Хранение данных"
prompt DATA_DIR_INPUT   "Директория для базы данных" "$DATA_DIR"
prompt RETENTION_DAYS  "Хранить данные (дней)" "30"

DATA_DIR="$DATA_DIR_INPUT"

# ──────────────────────────────────────────────────────────────────────────────
# Скачивание бинарника
# ──────────────────────────────────────────────────────────────────────────────
echo
info "Получаем последнюю версию из GitHub releases..."
LATEST_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
DOWNLOAD_URL=$(curl -fsSL "$LATEST_URL" \
    | grep -o "\"browser_download_url\": *\"[^\"]*${ARCH_TAG}[^\"]*\"" \
    | head -1 \
    | sed 's/.*": *"\(.*\)"/\1/')

[[ -z "$DOWNLOAD_URL" ]] && die "Не удалось найти бинарник для ${ARCH_TAG} в releases репозитория ${GITHUB_REPO}.\nУбедитесь что в репозитории есть GitHub Release с бинарником vpn-monitor-${ARCH_TAG}."

info "Скачиваем: $DOWNLOAD_URL"
mkdir -p "$INSTALL_DIR"
curl -fsSL -o "$BINARY" "$DOWNLOAD_URL"
chmod +x "$BINARY"
success "Бинарник установлен в $BINARY"

# ──────────────────────────────────────────────────────────────────────────────
# Создаём директории и конфигурацию
# ──────────────────────────────────────────────────────────────────────────────
mkdir -p "$DATA_DIR" "$CONFIG_DIR"

SESSION_SECRET=$(openssl rand -hex 32 2>/dev/null || cat /dev/urandom | tr -dc 'a-f0-9' | head -c 64)

cat > "$ENV_FILE" <<EOF
PORT=${APP_PORT}
DASH_USER=${DASH_USER}
DASH_PASS=${DASH_PASS}
SESSION_SECRET=${SESSION_SECRET}
HTTPS_ENABLED=${USE_HTTPS}
TLS_CERT=${TLS_CERT}
TLS_KEY=${TLS_KEY}
XUI_URL=${XUI_URL}
XUI_USER=${XUI_USER}
XUI_PASS=${XUI_PASS}
DATA_DIR=${DATA_DIR}
RETENTION_DAYS=${RETENTION_DAYS}
EOF

chmod 600 "$ENV_FILE"
chown root:root "$ENV_FILE"
success "Конфигурация сохранена в $ENV_FILE"

# ──────────────────────────────────────────────────────────────────────────────
# Systemd сервис
# ──────────────────────────────────────────────────────────────────────────────
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=VPN Monitor Dashboard
After=network.target

[Service]
Type=simple
User=root
EnvironmentFile=${ENV_FILE}
ExecStart=${BINARY}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now vpn-monitor
success "Служба vpn-monitor запущена и добавлена в автозапуск"

# ──────────────────────────────────────────────────────────────────────────────
# Итог
# ──────────────────────────────────────────────────────────────────────────────
SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[[ -z "$SERVER_IP" ]] && SERVER_IP=$(curl -fsSL --max-time 3 https://api.ipify.org 2>/dev/null || echo "<ваш-ip>")

PROTO="http"
[[ "$USE_HTTPS" == "true" ]] && PROTO="https"

echo
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║           Установка завершена успешно!               ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║${NC}  Адрес:    ${CYAN}${PROTO}://${SERVER_IP}:${APP_PORT}${NC}"
echo -e "${GREEN}║${NC}  Логин:    ${DASH_USER}"
echo -e "${GREEN}║${NC}  Данные:   ${DATA_DIR}"
echo -e "${GREEN}║${NC}  Логи:     journalctl -u vpn-monitor -f"
echo -e "${GREEN}║${NC}  Статус:   systemctl status vpn-monitor"
echo -e "${GREEN}║${NC}  Удалить:  bash install.sh --uninstall"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
