package archiver

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	LOG_FILE        = "/usr/local/x-ui/access.log"
	ARCHIVE_DIR     = "/usr/local/x-ui/archives"
	STATE_FILE      = "/usr/local/x-ui/last_archived_line.txt"
	POSITION_FILE   = "/usr/local/x-ui/last_archived_position.txt"
	TEMP_HOURLY_LOG = "/usr/local/x-ui/temp_hourly_archive.log"
)

// Archiver представляет архиватор логов
type Archiver struct {
	logFile       string
	archiveDir    string
	stateFile     string
	positionFile  string
	tempHourlyLog string
	localLogFile  string
}

// New создает новый экземпляр архиватора
func New() *Archiver {
	// Получаем домашнюю директорию пользователя для создания локального лога
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "." // Fallback к текущей директории
	}

	return &Archiver{
		logFile:       LOG_FILE,
		archiveDir:    ARCHIVE_DIR,
		stateFile:     STATE_FILE,
		positionFile:  POSITION_FILE,
		tempHourlyLog: TEMP_HOURLY_LOG,
		localLogFile:  filepath.Join(homeDir, "archiver.log"),
	}
}

// RunArchiving выполняет процесс архивирования
func (a *Archiver) RunArchiving() error {
	startTime := time.Now()
	fmt.Println("Начинаем процесс архивирования...")

	// Создаем необходимые директории, если их нет
	if err := os.MkdirAll(a.archiveDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", a.archiveDir, err)
	}

	// Получаем текущий размер файла
	fileInfo, err := os.Stat(a.logFile)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле %s: %v", a.logFile, err)
	}
	currentSize := fileInfo.Size()

	// Получаем позицию последней обработанной строки
	lastPosition := a.getLastProcessedPosition()

	// Если файл был очищен (стал меньше), начинаем с начала
	if currentSize < lastPosition {
		a.logInfo(fmt.Sprintf("Файл был очищен (было %d байт, стало %d), начинаем с начала", lastPosition, currentSize))
		lastPosition = 0
	}

	// Вычисляем количество новых байт
	newBytes := currentSize - lastPosition

	// Извлекаем новые строки и добавляем во временный файл-накопитель
	if newBytes > 0 {
		extractStart := time.Now()
		linesProcessed, err := a.appendNewLinesFromPosition(lastPosition)
		if err != nil {
			return fmt.Errorf("ошибка добавления новых строк: %v", err)
		}
		extractDuration := time.Since(extractStart)
		a.logInfo(fmt.Sprintf("Добавлено %d новых строк (%d байт) во временный накопитель за %v", linesProcessed, newBytes, extractDuration))
		a.logPerformance("EXTRACT_LINES", extractDuration, fmt.Sprintf("Извлечено %d строк из %d байт", linesProcessed, newBytes))
	} else {
		a.logInfo("Новых записей для добавления в накопитель не найдено.")
	}

	// Обновляем позицию - записываем текущий размер файла
	if err := a.updateLastProcessedPosition(currentSize); err != nil {
		return fmt.Errorf("ошибка обновления позиции: %v", err)
	}

	// Проверяем текущую минуту. Если это 00-я минута, архивируем накопитель.
	currentMinute := time.Now().Minute()
	if currentMinute == 0 {
		archiveStart := time.Now()
		if err := a.archiveHourlyLog(); err != nil {
			return fmt.Errorf("ошибка архивирования: %v", err)
		}
		archiveDuration := time.Since(archiveStart)
		a.logPerformance("ARCHIVE_HOURLY", archiveDuration, "Создан часовой архив")
	} else {
		fmt.Printf("Архивирование произойдет в %d минут следующего часа (в 00 минут)\n", 60-currentMinute)
	}

	// Очистка старых архивов отключена - архивы сохраняются навсегда
	// if err := a.cleanupOldArchives(); err != nil {
	//	return fmt.Errorf("ошибка очистки старых архивов: %v", err)
	// }

	// Логируем статистику производительности
	duration := time.Since(startTime)
	performanceStats := fmt.Sprintf("Процесс архивирования завершен за %v. Обработано байт: %d, новых байт: %d",
		duration, currentSize, newBytes)
	a.logInfo(performanceStats)
	a.logPerformance("TOTAL_RUN", duration, fmt.Sprintf("Обработано %d байт, новых %d", currentSize, newBytes))
	fmt.Printf("Архивирование завершено успешно! Время выполнения: %v\n", duration)
	return nil
}

func (a *Archiver) getLastProcessedPosition() int64 {
	data, err := os.ReadFile(a.positionFile)
	if err != nil {
		return 0
	}

	posStr := strings.TrimSpace(string(data))
	lastPos, err := strconv.ParseInt(posStr, 10, 64)
	if err != nil {
		return 0
	}
	return lastPos
}

func (a *Archiver) updateLastProcessedPosition(position int64) error {
	return os.WriteFile(a.positionFile, []byte(fmt.Sprintf("%d", position)), 0644)
}

func (a *Archiver) appendNewLinesFromPosition(startPosition int64) (int, error) {
	// Открываем основной лог файл
	logFile, err := os.Open(a.logFile)
	if err != nil {
		return 0, err
	}
	defer logFile.Close()

	// Переходим к позиции последней обработанной строки
	if _, err := logFile.Seek(startPosition, 0); err != nil {
		return 0, err
	}

	// Открываем временный файл для добавления
	tempFile, err := os.OpenFile(a.tempHourlyLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer tempFile.Close()

	// Используем буферизованный writer для эффективной записи
	writer := bufio.NewWriter(tempFile)
	defer writer.Flush()

	scanner := bufio.NewScanner(logFile)
	linesWritten := 0

	// Читаем только новые строки (с позиции startPosition до конца файла)
	for scanner.Scan() {
		if _, err := writer.WriteString(scanner.Text() + "\n"); err != nil {
			return linesWritten, err
		}
		linesWritten++
	}

	return linesWritten, scanner.Err()
}

func (a *Archiver) archiveHourlyLog() error {
	// Проверяем, есть ли данные в временном файле
	fileInfo, err := os.Stat(a.tempHourlyLog)
	if err != nil {
		return err
	}

	if fileInfo.Size() == 0 {
		a.logInfo("Временный накопитель пуст, часовой архив не создан.")
		// Очищаем временный накопитель после проверки
		return os.Truncate(a.tempHourlyLog, 0)
	}

	// Создаем имя архива с временной меткой
	timestamp := time.Now().Format("20060102_150405")
	archiveFile := filepath.Join(a.archiveDir, fmt.Sprintf("access_%s.log", timestamp))

	// Перемещаем временный файл в архив
	moveStart := time.Now()
	if err := os.Rename(a.tempHourlyLog, archiveFile); err != nil {
		return err
	}
	moveDuration := time.Since(moveStart)
	a.logPerformance("MOVE_TEMP_FILE", moveDuration, fmt.Sprintf("Перемещен файл размером %d байт", fileInfo.Size()))

	// Сжимаем архив
	compressStart := time.Now()
	if err := a.compressFile(archiveFile); err != nil {
		a.logError(fmt.Sprintf("Ошибка сжатия архива %s: %v", archiveFile, err))
		// Продолжаем выполнение даже если сжатие не удалось
	} else {
		compressDuration := time.Since(compressStart)
		a.logInfo(fmt.Sprintf("Архивирован часовой лог в %s.gz", archiveFile))
		a.logPerformance("COMPRESS_ARCHIVE", compressDuration, fmt.Sprintf("Сжат файл размером %d байт", fileInfo.Size()))
	}

	// Очищаем временный накопитель после архивирования
	return os.Truncate(a.tempHourlyLog, 0)
}

func (a *Archiver) compressFile(filename string) error {
	// Выполняем команду gzip
	cmd := exec.Command("gzip", filename)
	return cmd.Run()
}

func (a *Archiver) cleanupOldArchives() error {
	// Удаляем архивы старше 30 дней
	cmd := exec.Command("find", a.archiveDir, "-name", "access_*.log.gz", "-mtime", "+30", "-delete")
	return cmd.Run()
}

func (a *Archiver) logInfo(message string) {
	a.logMessage("INFO", message)
}

func (a *Archiver) logError(message string) {
	a.logMessage("ERROR", message)
}

func (a *Archiver) logMessage(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("%s [%s]: %s\n", timestamp, level, message)

	// Записываем в основной лог (в директории архивов)
	logFile := filepath.Join(a.archiveDir, "archive.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Ошибка записи в основной лог: %v\n", err)
	} else {
		file.WriteString(logEntry)
		file.Close()
	}

	// Записываем в локальный лог (в директории запуска)
	localFile, err := os.OpenFile(a.localLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Ошибка записи в локальный лог: %v\n", err)
		return
	}
	defer localFile.Close()

	localFile.WriteString(logEntry)
}

// logPerformance записывает информацию о производительности в локальный лог
func (a *Archiver) logPerformance(operation string, duration time.Duration, details string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	perfEntry := fmt.Sprintf("%s [PERF] %s: %v - %s\n", timestamp, operation, duration, details)

	localFile, err := os.OpenFile(a.localLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Ошибка записи в локальный лог: %v\n", err)
		return
	}
	defer localFile.Close()

	localFile.WriteString(perfEntry)
}
