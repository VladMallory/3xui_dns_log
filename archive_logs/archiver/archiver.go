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
}

// New создает новый экземпляр архиватора
func New() *Archiver {
	return &Archiver{
		logFile:       LOG_FILE,
		archiveDir:    ARCHIVE_DIR,
		stateFile:     STATE_FILE,
		tempHourlyLog: TEMP_HOURLY_LOG,
	}
}

// RunArchiving выполняет процесс архивирования
func (a *Archiver) RunArchiving() error {
	fmt.Println("Начинаем процесс архивирования...")

	// Создаем необходимые директории, если их нет
	if err := os.MkdirAll(a.archiveDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", a.archiveDir, err)
	}

	// Получаем общее количество строк в файле логов
	totalLines, err := a.countLines(a.logFile)
	if err != nil {
		return fmt.Errorf("ошибка подсчета строк в %s: %v", a.logFile, err)
	}

	// Получаем номер последней обработанной строки
	lastLine := a.getLastProcessedLine()

	// Если файл логов был сброшен (очищен), начинаем с начала
	if totalLines < lastLine {
		lastLine = 0
	}

	// Вычисляем количество новых строк
	newLines := totalLines - lastLine

	// Извлекаем новые строки и добавляем во временный файл-накопитель
	if newLines > 0 {
		if err := a.appendNewLinesToTempFile(newLines, lastLine); err != nil {
			return fmt.Errorf("ошибка добавления новых строк: %v", err)
		}
		a.logInfo(fmt.Sprintf("Добавлено %d новых строк во временный накопитель.", newLines))
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
		if err := a.archiveHourlyLog(); err != nil {
			return fmt.Errorf("ошибка архивирования: %v", err)
		}
	} else {
		fmt.Printf("Архивирование произойдет в %d минут следующего часа (в 00 минут)\n", 60-currentMinute)
	}

	// Очистка старых архивов отключена - архивы сохраняются навсегда
	// if err := a.cleanupOldArchives(); err != nil {
	//	return fmt.Errorf("ошибка очистки старых архивов: %v", err)
	// }

	a.logInfo("Процесс архивирования завершен")
	fmt.Println("Архивирование завершено успешно!")
	return nil
}

func (a *Archiver) countLines(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, scanner.Err()
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

	scanner := bufio.NewScanner(logFile)
	currentLine := 0
	linesToSkip := lastLine
	linesToTake := newLines

	for scanner.Scan() {
		currentLine++
		if currentLine <= linesToSkip {
			continue
		}
		if currentLine > linesToSkip+linesToTake {
			break
		}
		if _, err := tempFile.WriteString(scanner.Text() + "\n"); err != nil {
			return err
		}
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
	if err := os.Rename(a.tempHourlyLog, archiveFile); err != nil {
		return err
	}

	// Сжимаем архив
	if err := a.compressFile(archiveFile); err != nil {
		a.logError(fmt.Sprintf("Ошибка сжатия архива %s: %v", archiveFile, err))
		// Продолжаем выполнение даже если сжатие не удалось
	} else {
		a.logInfo(fmt.Sprintf("Архивирован часовой лог в %s.gz", archiveFile))
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
	logEntry := fmt.Sprintf("%s: %s\n", timestamp, message)

	logFile := filepath.Join(a.archiveDir, "archive.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Ошибка записи в лог: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(logEntry)
}
