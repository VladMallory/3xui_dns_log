#!/bin/bash

LOG_FILE="/usr/local/x-ui/access.log"
ARCHIVE_DIR="/usr/local/x-ui/archives"
STATE_FILE="/usr/local/x-ui/last_archived_line.txt"
TEMP_HOURLY_LOG="/usr/local/x-ui/temp_hourly_archive.log"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
ARCHIVE_FILE="$ARCHIVE_DIR/access_$TIMESTAMP.log"

# Создаем необходимые директории, если их нет
mkdir -p "$ARCHIVE_DIR"

# Получаем общее количество строк в файле логов
TOTAL_LINES=$(wc -l < "$LOG_FILE")

# Получаем номер последней обработанной строки
if [ -f "$STATE_FILE" ]; then
    LAST_LINE=$(cat "$STATE_FILE")
    if ! [[ "$LAST_LINE" =~ ^[0-9]+$ ]]; then
        LAST_LINE=0
    fi
else
    LAST_LINE=0
fi

# Если файл логов был сброшен (очищен), начинаем с начала
if [ $TOTAL_LINES -lt $LAST_LINE ]; then
    LAST_LINE=0
fi

# Вычисляем количество новых строк
NEW_LINES=$((TOTAL_LINES - LAST_LINE))

# Извлекаем новые строки и добавляем во временный файл-накопитель
if [ $NEW_LINES -gt 0 ]; then
    tail -n +$((LAST_LINE + 1)) "$LOG_FILE" | head -n $NEW_LINES >> "$TEMP_HOURLY_LOG"
    echo "$(date): Добавлено $NEW_LINES новых строк во временный накопитель." >> "$ARCHIVE_DIR/archive.log"
else
    echo "$(date): Новых записей для добавления в накопитель не найдено." >> "$ARCHIVE_DIR/archive.log"
fi

# Обновляем состояние - записываем номер последней обработанной строки
echo "$TOTAL_LINES" > "$STATE_FILE"

# Проверяем текущую минуту. Если это 00-я минута, архивируем накопитель.
CURRENT_MINUTE=$(date +"%M")
if [ "$CURRENT_MINUTE" -eq 00 ]; then
    if [ -s "$TEMP_HOURLY_LOG" ]; then
        # Архивируем содержимое временного накопителя
        mv "$TEMP_HOURLY_LOG" "$ARCHIVE_FILE"
        gzip "$ARCHIVE_FILE"
        echo "$(date): Архивирован часовой лог в ${ARCHIVE_FILE}.gz" >> "$ARCHIVE_DIR/archive.log"
    else
        echo "$(date): Временный накопитель пуст, часовой архив не создан." >> "$ARCHIVE_DIR/archive.log"
    fi
    # Очищаем временный накопитель после архивирования
    > "$TEMP_HOURLY_LOG"
fi

# Очистка старых архивов (старше 30 дней)
find "$ARCHIVE_DIR" -name "access_*.log.gz" -mtime +30 -delete

echo "$(date): Процесс архивирования завершен" >> "$ARCHIVE_DIR/archive.log"


