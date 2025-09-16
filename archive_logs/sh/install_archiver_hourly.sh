#!/bin/bash

ARCHIVE_SCRIPT_NAME="archive_logs.sh"
ARCHIVE_SCRIPT_PATH="/usr/local/bin/archive_3xui_logs.sh"
LOG_FILE_PATH="/usr/local/x-ui/access.log"
ARCHIVE_DIR="/usr/local/x-ui/archives"
STATE_FILE="/usr/local/x-ui/last_archived_line.txt"
TEMP_HOURLY_LOG="/usr/local/x-ui/temp_hourly_archive.log"

# Копируем скрипт архивирования
cp "./$ARCHIVE_SCRIPT_NAME" "$ARCHIVE_SCRIPT_PATH"
chmod +x "$ARCHIVE_SCRIPT_PATH"

# Создаем директорию для архивов если её нет
mkdir -p "$ARCHIVE_DIR"

# Создаем файл состояния, если его нет
if [ ! -f "$STATE_FILE" ]; then
    echo "0" > "$STATE_FILE"
fi

# Создаем временный файл-накопитель, если его нет
if [ ! -f "$TEMP_HOURLY_LOG" ]; then
    touch "$TEMP_HOURLY_LOG"
fi

# Добавляем задачу в cron для выполнения каждые 10 минут для root, если её нет
(sudo crontab -l 2>/dev/null | grep -v -F "$ARCHIVE_SCRIPT_PATH"; echo "*/10 * * * * $ARCHIVE_SCRIPT_PATH") | sudo crontab -

echo "Установка завершена. Логи будут проверяться каждые 10 минут."
echo "Часовые архивы будут создаваться в 59 минут каждого часа и сохраняться в $ARCHIVE_DIR"
echo "Лог работы архиватора: $ARCHIVE_DIR/archive.log"

