# X-UI DNS Log Management Tools

Программа собирает и логирует dns

# Установка и настройка
## archive_logs

```bash
# Клонируем репозиторий
git clone <repository-url>
cd 3xui_dns_log/archive_logs/
go run main.go
```
## merge_logs
```bash
cd 3xui_dns_log/merge_logs
go run main.go
```

## 📋 Что умеет

### 1. 🔄 X-UI Log Archiver

#### Основные возможности:
- ⏰ **Автоматическое архивирование** - читает новые строки из `/usr/local/x-ui/access.log`
- 🕐 **Часовое архивирование** - создает сжатые архивы в начале каждого часа
- 🔧 **Управление автозапуском** - установка/удаление через cron
- 📊 **Детальное логирование** - ведет лог работы в `/usr/local/x-ui/archives/archive.log`


### 2. 🔗 Merge Logs Tool

**Утилита для объединения и обработки архивных логов**

#### Функциональность:
- 📁 **Автосоздание директорий** - создает необходимые папки
- 📦 **Копирование архивов** - копирует `.gz` файлы из `/usr/local/x-ui/archives`
- 📂 **Распаковка** - автоматически распаковывает все архивы
- 🔄 **Объединение** - объединяет все логи с сортировкой и удалением дубликатов
- 🧹 **Очистка** - удаляет временные файлы после обработки
- 🧪 **Тестовые данные** - создает тестовые архивы если исходная папка пуста

#### Результат работы:
- Объединенный файл: `/usr/local/x-ui/mergelog/merged_access.log`
- Временные файлы: `/usr/local/x-ui/mergelog/logs/`

## ⚙️ Системные требования

### Минимальные требования:
- **Go**: версия 1.21 или выше
- **ОС**: Linux (тестировано на Ubuntu/Debian)
- **Права**: root для установки архиватора
- **Утилиты**: `gzip`, `find`

### Права доступа:
- Чтение: `/usr/local/x-ui/access.log`
- Запись: `/usr/local/x-ui/archives/`
- Запись: `/usr/local/x-ui/mergelog/`

## 🔧 Конфигурация

### Пути архиватора
Настраиваются в `archive_logs/archiver/archiver.go`:
```go
const (
    ACCESS_LOG_PATH = "/usr/local/x-ui/access.log"
    ARCHIVE_DIR     = "/usr/local/x-ui/archives"
    ARCHIVE_LOG     = "/usr/local/x-ui/archives/archive.log"
)
```

### Пути merge_logs
Настраиваются в `merge_logs/main.go`:
```go
const (
    ARCHIVE_SOURCE_DIR = "/usr/local/x-ui/archives"
    MERGE_DEST_DIR     = "/usr/local/x-ui/mergelog"
    MERGED_LOG_FILE    = "/usr/local/x-ui/mergelog/merged_access.log"
)
```

## 📊 Логирование и мониторинг

### Архиватор
- **Лог работы**: `/usr/local/x-ui/archives/archive.log`
- **Формат логов**: `YYYY-MM-DD HH:MM:SS [LEVEL] MESSAGE`
- **Уровни**: INFO, WARN, ERROR

### Merge Logs
- **Вывод**: консоль с подробной информацией о процессе
- **Статистика**: количество обработанных файлов и строк

## 🔄 Автоматизация

### Cron настройка
После установки архиватор автоматически выполняется:
- **Частота**: каждые 10 минут
- **Архивирование**: в начале каждого часа (минута 00)
- **Команда**: `*/10 * * * * /path/to/xui_log_archiver --cron`

### Рекомендуемый workflow

1. **Установка и настройка**:
   ```bash
   cd archive_logs
   go build -o xui_log_archiver main.go
   sudo ./xui_log_archiver
   # Выберите пункт 2 для установки автозапуска
   ```

2. **Периодическое объединение**:
   ```bash
   cd merge_logs
   go build -o merge_logs main.go
   ./merge_logs
   ```

3. **Мониторинг**:
   ```bash
   # Проверка логов архиватора
   tail -f /usr/local/x-ui/archives/archive.log
   
   # Проверка статуса автозапуска
   sudo ./xui_log_archiver
   # Выберите пункт 4
   ```

## 🛠️ Разработка

### Структура проекта
```
3xui_dns_log/
├── archive_logs/              # Система архивирования
│   ├── main.go               # Главная программа
│   ├── archiver/             # Модуль архивирования
│   ├── installer/            # Модуль установки
│   ├── sh/                   # Bash скрипты (legacy)
│   └── go.mod                # Go модуль
├── merge_logs/               # Система объединения
│   └── main.go               # Программа объединения
├── release.sh                # Скрипт релиза
└── README.md                 # Документация
```

### Сборка
```bash
# Архиватор
cd archive_logs
go build -o xui_log_archiver main.go

# Merge logs
cd merge_logs
go build -o merge_logs main.go
```

### Тестирование
```bash
# Тест архиватора
sudo ./xui_log_archiver --cron

# Тест объединения (создаст тестовые данные)
./merge_logs
```

## 🔍 Отладка

### Частые проблемы

1. **Ошибка прав доступа**:
   ```bash
   sudo chown -R root:root /usr/local/x-ui/
   sudo chmod 755 /usr/local/x-ui/archives/
   ```

2. **Архиватор не запускается**:
   ```bash
   # Проверьте права на файл
   ls -la /usr/local/x-ui/access.log
   
   # Проверьте cron
   sudo crontab -l
   ```

3. **Merge logs не находит архивы**:
   ```bash
   # Проверьте наличие архивов
   ls -la /usr/local/x-ui/archives/
   
   # Запустите с тестовыми данными
   ./merge_logs
   ```

## 📈 Производительность

### Оптимизации
- **Потоковая обработка** - файлы читаются построчно
- **Удаление дубликатов** - используется map для быстрого поиска
- **Сжатие** - архивы сохраняются в gzip формате
- **Очистка** - автоматическое удаление временных файлов

### Ограничения
- **Память**: ~1MB на 100,000 строк логов
- **Диск**: архивы сжимаются до ~10% от исходного размера
- **Время**: обработка 1GB логов занимает ~2-3 минуты

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для новой функции
3. Внесите изменения
4. Создайте Pull Request

## 📄 Лицензия

Проект распространяется под лицензией MIT.

## 🆘 Поддержка

При возникновении проблем:
1. Проверьте раздел "Отладка"
2. Изучите логи в `/usr/local/x-ui/archives/archive.log`
3. Создайте Issue с описанием проблемы

---

**Версия**: 1.0.0  
**Последнее обновление**: $(date +%Y-%m-%d)  
**Автор**: X-UI Log Management Team