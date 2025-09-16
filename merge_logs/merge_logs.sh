#!/bin/bash

ARCHIVE_SOURCE_DIR="/usr/local/x-ui/archives"
MERGE_DEST_DIR="/usr/local/x-ui/mergelog"
LOGS_SUBDIR="$MERGE_DEST_DIR/logs"
MERGED_LOG_FILE="$MERGE_DEST_DIR/merged_access.log"

# Создаем необходимые директории, если их нет
mkdir -p "$LOGS_SUBDIR"

# Копируем все архивы из исходной директории в директорию для логов
cp "$ARCHIVE_SOURCE_DIR"/*.gz "$LOGS_SUBDIR" 2>/dev/null

# Переходим в директорию с логами для удобства распаковки
cd "$LOGS_SUBDIR"

# Распаковываем все архивы. Опция -f перезаписывает существующие файлы.
gunzip -f *.gz 2>/dev/null

# Склеиваем все распакованные файлы в один, сортируем и удаляем дубликаты
# Используем sort -u для удаления дубликатов строк
cat *.log | sort -u > "$MERGED_LOG_FILE"

# Очищаем директорию с распакованными логами, оставляя только объединенный файл
rm -f *.log

echo "$(date): Логи успешно скопированы, объединены и сохранены в $MERGED_LOG_FILE"


