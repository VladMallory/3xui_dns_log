package installer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	SCRIPT_PATH     = "/usr/local/bin/xui_log_archiver"
	ARCHIVE_DIR     = "/usr/local/x-ui/archives"
	STATE_FILE      = "/usr/local/x-ui/last_archived_line.txt"
	TEMP_HOURLY_LOG = "/usr/local/x-ui/temp_hourly_archive.log"
)

// Installer управляет установкой и удалением автозапуска
type Installer struct {
	scriptPath string
}

// New создает новый экземпляр установщика
func New() *Installer {
	return &Installer{
		scriptPath: SCRIPT_PATH,
	}
}

// InstallAutostart устанавливает автозапуск
func (i *Installer) InstallAutostart() error {
	fmt.Println("Установка автозапуска...")

	// Копируем текущую программу в /usr/local/bin/
	if err := i.copySelfToBin(); err != nil {
		return fmt.Errorf("ошибка копирования программы: %v", err)
	}

	// Создаем необходимые директории и файлы
	if err := i.createDirectoriesAndFiles(); err != nil {
		return fmt.Errorf("ошибка создания директорий и файлов: %v", err)
	}

	// Добавляем задачу в cron
	if err := i.addCronJob(); err != nil {
		return fmt.Errorf("ошибка добавления задачи в cron: %v", err)
	}

	fmt.Println("✅ Автозапуск установлен успешно!")
	fmt.Println("📅 Программа будет выполняться каждые 10 минут")
	fmt.Println("📦 Часовые архивы будут создаваться в 00 минут каждого часа")
	fmt.Printf("📁 Архивы сохраняются в: %s\n", ARCHIVE_DIR)
	fmt.Printf("📋 Лог работы: %s/archive.log\n", ARCHIVE_DIR)
	return nil
}

// RemoveAutostart удаляет автозапуск
func (i *Installer) RemoveAutostart() error {
	fmt.Println("Удаление автозапуска...")

	// Получаем текущий crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil && !strings.Contains(err.Error(), "no crontab") {
		return fmt.Errorf("ошибка получения crontab: %v", err)
	}

	currentCrontab := string(output)

	// Проверяем, есть ли наша задача
	if !strings.Contains(currentCrontab, i.scriptPath) {
		fmt.Println("❌ Автозапуск не найден в crontab")
		return nil
	}

	// Удаляем нашу задачу
	lines := strings.Split(currentCrontab, "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, i.scriptPath) {
			newLines = append(newLines, line)
		}
	}

	newCrontab := strings.Join(newLines, "\n")

	// Создаем временный файл с новым crontab
	tempFile, err := os.CreateTemp("", "crontab_*")
	if err != nil {
		return fmt.Errorf("ошибка создания временного файла: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(newCrontab); err != nil {
		tempFile.Close()
		return fmt.Errorf("ошибка записи в временный файл: %v", err)
	}
	tempFile.Close()

	// Устанавливаем новый crontab
	cmd = exec.Command("crontab", tempFile.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка установки нового crontab: %v", err)
	}

	fmt.Println("✅ Автозапуск удален успешно!")
	return nil
}

// ShowAutostartStatus показывает статус автозапуска
func (i *Installer) ShowAutostartStatus() {
	// Получаем текущий crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil && !strings.Contains(err.Error(), "no crontab") {
		fmt.Printf("Ошибка получения crontab: %v\n", err)
		return
	}

	currentCrontab := string(output)

	if strings.Contains(currentCrontab, i.scriptPath) {
		fmt.Println("✅ Автозапуск активен")
		// Показываем строку из crontab
		lines := strings.Split(currentCrontab, "\n")
		for _, line := range lines {
			if strings.Contains(line, i.scriptPath) {
				fmt.Printf("📅 Расписание: %s\n", line)
				break
			}
		}
	} else {
		fmt.Println("❌ Автозапуск не настроен")
	}

	// Показываем информацию о файлах
	fmt.Printf("📁 Директория архивов: %s\n", ARCHIVE_DIR)
	if _, err := os.Stat(ARCHIVE_DIR); err == nil {
		fmt.Println("✅ Директория архивов существует")
	} else {
		fmt.Println("❌ Директория архивов не существует")
	}

	fmt.Printf("📄 Файл состояния: %s\n", STATE_FILE)
	if _, err := os.Stat(STATE_FILE); err == nil {
		fmt.Println("✅ Файл состояния существует")
	} else {
		fmt.Println("❌ Файл состояния не существует")
	}
}

func (i *Installer) copySelfToBin() error {
	// Получаем путь к текущему исполняемому файлу
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("ошибка получения пути к программе: %v", err)
	}

	// Копируем файл
	if err := i.copyFile(execPath, i.scriptPath); err != nil {
		return fmt.Errorf("ошибка копирования файла: %v", err)
	}

	// Устанавливаем права на выполнение
	if err := os.Chmod(i.scriptPath, 0755); err != nil {
		return fmt.Errorf("ошибка установки прав на выполнение: %v", err)
	}

	fmt.Printf("✅ Программа скопирована в %s\n", i.scriptPath)
	return nil
}

func (i *Installer) createDirectoriesAndFiles() error {
	// Создаем директорию для архивов
	if err := os.MkdirAll(ARCHIVE_DIR, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", ARCHIVE_DIR, err)
	}

	// Создаем файл состояния, если его нет
	if _, err := os.Stat(STATE_FILE); os.IsNotExist(err) {
		if err := os.WriteFile(STATE_FILE, []byte("0"), 0644); err != nil {
			return fmt.Errorf("ошибка создания файла состояния: %v", err)
		}
		fmt.Printf("✅ Создан файл состояния: %s\n", STATE_FILE)
	}

	// Создаем временный файл-накопитель, если его нет
	if _, err := os.Stat(TEMP_HOURLY_LOG); os.IsNotExist(err) {
		if err := os.WriteFile(TEMP_HOURLY_LOG, []byte(""), 0644); err != nil {
			return fmt.Errorf("ошибка создания временного файла: %v", err)
		}
		fmt.Printf("✅ Создан временный файл-накопитель: %s\n", TEMP_HOURLY_LOG)
	}

	return nil
}

func (i *Installer) addCronJob() error {
	// Получаем текущий crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil && !strings.Contains(err.Error(), "no crontab") {
		return fmt.Errorf("ошибка получения текущего crontab: %v", err)
	}

	currentCrontab := string(output)

	// Проверяем, есть ли уже наша задача
	if strings.Contains(currentCrontab, i.scriptPath) {
		fmt.Println("⚠️  Задача уже существует в crontab")
		return nil
	}

	// Добавляем новую задачу
	newCronEntry := fmt.Sprintf("*/10 * * * * %s --cron\n", i.scriptPath)
	newCrontab := currentCrontab + newCronEntry

	// Создаем временный файл с новым crontab
	tempFile, err := os.CreateTemp("", "crontab_*")
	if err != nil {
		return fmt.Errorf("ошибка создания временного файла: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(newCrontab); err != nil {
		tempFile.Close()
		return fmt.Errorf("ошибка записи в временный файл: %v", err)
	}
	tempFile.Close()

	// Устанавливаем новый crontab
	cmd = exec.Command("crontab", tempFile.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка установки нового crontab: %v", err)
	}

	fmt.Println("✅ Задача добавлена в crontab: каждые 10 минут")
	return nil
}

func (i *Installer) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
