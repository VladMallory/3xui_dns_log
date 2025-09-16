// Package main предоставляет утилиту для объединения и обработки логов
package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ARCHIVE_SOURCE_DIR = "/usr/local/x-ui/archives"
	MERGE_DEST_DIR     = "/usr/local/x-ui/mergelog"
	LOGS_SUBDIR        = "/usr/local/x-ui/mergelog/logs"
	MERGED_LOG_FILE    = "/usr/local/x-ui/mergelog/merged_access.log"
)

func main() {
	// Создаем необходимые директории, если их нет
	if err := os.MkdirAll(ARCHIVE_SOURCE_DIR, 0755); err != nil {
		log.Fatalf("Ошибка создания директории %s: %v", ARCHIVE_SOURCE_DIR, err)
	}

	if err := os.MkdirAll(LOGS_SUBDIR, 0755); err != nil {
		log.Fatalf("Ошибка создания директории %s: %v", LOGS_SUBDIR, err)
	}

	// Создаем тестовые архивы, если исходная директория пуста
	if err := createTestArchives(); err != nil {
		log.Printf("Предупреждение: не удалось создать тестовые архивы: %v", err)
	}

	// Копируем и распаковываем архивы
	if err := copyAndExtractArchives(); err != nil {
		log.Fatalf("Ошибка копирования и распаковки архивов: %v", err)
	}

	// Объединяем логи
	if err := mergeLogs(); err != nil {
		log.Fatalf("Ошибка объединения логов: %v", err)
	}

	// Очищаем временные файлы
	if err := cleanupTempFiles(); err != nil {
		log.Printf("Предупреждение: ошибка очистки временных файлов: %v", err)
	}

	fmt.Printf("%s: Логи успешно скопированы, объединены и сохранены в %s\n",
		time.Now().Format("2006-01-02 15:04:05"), MERGED_LOG_FILE)
}

// createTestArchives создает тестовые архивы, если исходная директория пуста
func createTestArchives() error {
	// Проверяем, есть ли уже файлы в исходной директории
	entries, err := os.ReadDir(ARCHIVE_SOURCE_DIR)
	if err != nil {
		return err
	}

	// Если есть .gz файлы, не создаем тестовые
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gz") {
			return nil // Уже есть архивы
		}
	}

	// Создаем тестовые .log файлы
	testLogs := []string{
		"2024-01-01 10:00:00 INFO: Test log entry 1",
		"2024-01-01 10:01:00 ERROR: Test error message",
		"2024-01-01 10:02:00 INFO: Another test log",
		"2024-01-01 10:03:00 WARN: Test warning message",
		"2024-01-01 10:04:00 INFO: Final test log",
	}

	// Создаем несколько тестовых .log файлов
	for i, content := range testLogs {
		logFile := filepath.Join(ARCHIVE_SOURCE_DIR, fmt.Sprintf("test_log_%d.log", i+1))
		if err := os.WriteFile(logFile, []byte(content+"\n"), 0644); err != nil {
			return fmt.Errorf("ошибка создания тестового файла %s: %v", logFile, err)
		}

		// Создаем .gz архив из .log файла
		gzFile := logFile + ".gz"
		if err := createGzipFile(logFile, gzFile); err != nil {
			os.Remove(logFile) // Удаляем .log файл при ошибке
			return fmt.Errorf("ошибка создания архива %s: %v", gzFile, err)
		}

		// Удаляем исходный .log файл
		os.Remove(logFile)
	}

	fmt.Println("Созданы тестовые архивы в", ARCHIVE_SOURCE_DIR)
	return nil
}

// createGzipFile создает .gz архив из исходного файла
func createGzipFile(sourceFile, gzFile string) error {
	// Открываем исходный файл
	src, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer src.Close()

	// Создаем .gz файл
	dst, err := os.Create(gzFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Создаем gzip writer
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()

	// Копируем данные
	_, err = io.Copy(gzWriter, src)
	return err
}

// copyAndExtractArchives копирует .gz файлы из исходной директории и распаковывает их
func copyAndExtractArchives() error {
	// Читаем все .gz файлы из исходной директории
	entries, err := os.ReadDir(ARCHIVE_SOURCE_DIR)
	if err != nil {
		return fmt.Errorf("ошибка чтения директории %s: %v", ARCHIVE_SOURCE_DIR, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gz") {
			sourcePath := filepath.Join(ARCHIVE_SOURCE_DIR, entry.Name())
			destPath := filepath.Join(LOGS_SUBDIR, entry.Name())

			// Копируем файл
			if err := copyFile(sourcePath, destPath); err != nil {
				log.Printf("Предупреждение: не удалось скопировать %s: %v", sourcePath, err)
				continue
			}

			// Распаковываем файл
			if err := extractGzipFile(destPath); err != nil {
				log.Printf("Предупреждение: не удалось распаковать %s: %v", destPath, err)
				continue
			}
		}
	}

	return nil
}

// copyFile копирует файл из source в destination
func copyFile(source, dest string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// extractGzipFile распаковывает .gz файл
func extractGzipFile(gzPath string) error {
	// Открываем сжатый файл
	gzFile, err := os.Open(gzPath)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	// Создаем gzip reader
	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	// Создаем файл для распакованных данных
	extractedPath := strings.TrimSuffix(gzPath, ".gz")
	extractedFile, err := os.Create(extractedPath)
	if err != nil {
		return err
	}
	defer extractedFile.Close()

	// Копируем распакованные данные
	_, err = io.Copy(extractedFile, gzReader)
	if err != nil {
		os.Remove(extractedPath) // Удаляем частично созданный файл при ошибке
		return err
	}

	// Удаляем исходный .gz файл
	return os.Remove(gzPath)
}

// mergeLogs объединяет все .log файлы, сортирует и удаляет дубликаты
func mergeLogs() error {
	// Читаем все .log файлы из директории логов
	entries, err := os.ReadDir(LOGS_SUBDIR)
	if err != nil {
		return fmt.Errorf("ошибка чтения директории %s: %v", LOGS_SUBDIR, err)
	}

	var allLines []string
	lineSet := make(map[string]bool) // Для удаления дубликатов

	// Читаем все строки из всех .log файлов
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			filePath := filepath.Join(LOGS_SUBDIR, entry.Name())

			file, err := os.Open(filePath)
			if err != nil {
				log.Printf("Предупреждение: не удалось открыть %s: %v", filePath, err)
				continue
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !lineSet[line] {
					lineSet[line] = true
					allLines = append(allLines, line)
				}
			}

			file.Close()
		}
	}

	// Сортируем строки
	sort.Strings(allLines)

	// Записываем объединенный файл
	mergedFile, err := os.Create(MERGED_LOG_FILE)
	if err != nil {
		return fmt.Errorf("ошибка создания файла %s: %v", MERGED_LOG_FILE, err)
	}
	defer mergedFile.Close()

	writer := bufio.NewWriter(mergedFile)
	for _, line := range allLines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("ошибка записи в файл: %v", err)
		}
	}

	return writer.Flush()
}

// cleanupTempFiles удаляет временные .log файлы из директории логов
func cleanupTempFiles() error {
	entries, err := os.ReadDir(LOGS_SUBDIR)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			filePath := filepath.Join(LOGS_SUBDIR, entry.Name())
			if err := os.Remove(filePath); err != nil {
				log.Printf("Предупреждение: не удалось удалить %s: %v", filePath, err)
			}
		}
	}

	return nil
}
