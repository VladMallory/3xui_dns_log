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
	TEMP_HOURLY_LOG = "/usr/local/x-ui/temp_hourly_archive.log"
)

// Archiver представляет архиватор логов
type Archiver struct {
	logFile       string
	archiveDir    string
	stateFile     string
	tempHourlyLog string
	localLogFile  string
}

// New создает новый экземпляр архиватора
func New() *Archiver {
	// Получаем текущую директорию для создания локального лога
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "." // Fallback к текущей директории
	}

	return &Archiver{
		logFile:       LOG_FILE,
		archiveDir:    ARCHIVE_DIR,
		stateFile:     STATE_FILE,
		tempHourlyLog: TEMP_HOURLY_LOG,
		localLogFile:  filepath.Join(workDir, "archiver.log"),
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

	// Получаем общее количество строк в файле логов
	countStart := time.Now()
	totalLines, err := a.countLines(a.logFile)
	if err != nil {
		return fmt.Errorf("ошибка подсчета строк в %s: %v", a.logFile, err)
	}
	countDuration := time.Since(countStart)
	a.logInfo(fmt.Sprintf("Подсчет строк выполнен за %v", countDuration))
	a.logPerformance("COUNT_LINES", countDuration, fmt.Sprintf("Обработано %d строк", totalLines))

	// Получаем номер последней обработанной строки
	lastLine := a.getLastProcessedLine()

	// Если файл логов был сброшен (очищен), начинаем с начала
	if totalLines < lastLine {
		a.logInfo(fmt.Sprintf("Файл был очищен (было %d строк, стало %d), начинаем с начала", lastLine, totalLines))
		lastLine = 0
	}

	// Вычисляем количество новых строк
	newLines := totalLines - lastLine

	// Извлекаем новые строки и добавляем во временный файл-накопитель
	if newLines > 0 {
		extractStart := time.Now()
		if err := a.appendNewLinesToTempFile(newLines, lastLine); err != nil {
			return fmt.Errorf("ошибка добавления новых строк: %v", err)
		}
		extractDuration := time.Since(extractStart)
		a.logInfo(fmt.Sprintf("Добавлено %d новых строк во временный накопитель за %v", newLines, extractDuration))
		a.logPerformance("EXTRACT_LINES", extractDuration, fmt.Sprintf("Извлечено %d новых строк", newLines))
	} else {
		a.logInfo("Новых записей для добавления в накопитель не найдено.")
	}

	// Обновляем состояние - записываем номер последней обработанной строки
	if err := a.updateLastProcessedLine(totalLines); err != nil {
		return fmt.Errorf("ошибка обновления состояния: %v", err)
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
	performanceStats := fmt.Sprintf("Процесс архивирования завершен за %v. Обработано строк: %d, новых строк: %d",
		duration, totalLines, newLines)
	a.logInfo(performanceStats)
	a.logPerformance("TOTAL_RUN", duration, fmt.Sprintf("Обработано %d строк, новых %d", totalLines, newLines))
	fmt.Printf("Архивирование завершено успешно! Время выполнения: %v\n", duration)
	return nil
}

func (a *Archiver) countLines(filename string) (int, error) {
	// Используем размер файла для быстрого подсчета приблизительного количества строк
	// Это намного быстрее чем чтение всего файла построчно
	stat, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}

	// Приблизительная оценка: средняя строка лога ~100-150 байт
	// Для точности читаем первые 1000 строк и вычисляем средний размер
	avgLineSize, err := a.calculateAverageLineSize(filename)
	if err != nil {
		// Если не удалось вычислить, используем консервативную оценку
		avgLineSize = 120
	}

	estimatedLines := int(stat.Size() / avgLineSize)
	a.logInfo(fmt.Sprintf("Размер файла: %d байт, примерное количество строк: %d", stat.Size(), estimatedLines))
	return estimatedLines, nil
}

func (a *Archiver) calculateAverageLineSize(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	totalSize := int64(0)
	lineCount := 0
	maxLines := 1000 // Читаем только первые 1000 строк для расчета

	for scanner.Scan() && lineCount < maxLines {
		totalSize += int64(len(scanner.Bytes()) + 1) // +1 для символа новой строки
		lineCount++
	}

	if lineCount == 0 {
		return 120, nil // Значение по умолчанию
	}

	return totalSize / int64(lineCount), scanner.Err()
}

func (a *Archiver) getLastProcessedLine() int {
	data, err := os.ReadFile(a.stateFile)
	if err != nil {
		return 0
	}

	lineStr := strings.TrimSpace(string(data))
	lastLine, err := strconv.Atoi(lineStr)
	if err != nil {
		return 0
	}
	return lastLine
}

func (a *Archiver) appendNewLinesToTempFile(newLines, lastLine int) error {
	// Открываем основной лог файл
	logFile, err := os.Open(a.logFile)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// Открываем временный файл для добавления
	tempFile, err := os.OpenFile(a.tempHourlyLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	// Используем буферизованный writer для эффективной записи
	writer := bufio.NewWriter(tempFile)
	defer writer.Flush()

	scanner := bufio.NewScanner(logFile)
	currentLine := 0
	linesToSkip := lastLine
	linesToTake := newLines
	linesWritten := 0

	// Пропускаем уже обработанные строки
	for scanner.Scan() && currentLine < linesToSkip {
		currentLine++
	}

	// Обрабатываем новые строки
	for scanner.Scan() && linesWritten < linesToTake {
		if _, err := writer.WriteString(scanner.Text() + "\n"); err != nil {
			return err
		}
		linesWritten++
	}

	// Проверяем, не осталось ли строк (файл мог увеличиться)
	if linesWritten < linesToTake {
		a.logInfo(fmt.Sprintf("Обработано %d строк из ожидаемых %d (файл мог увеличиться)", linesWritten, linesToTake))
	}

	return scanner.Err()
}

func (a *Archiver) updateLastProcessedLine(totalLines int) error {
	return os.WriteFile(a.stateFile, []byte(fmt.Sprintf("%d", totalLines)), 0644)
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
