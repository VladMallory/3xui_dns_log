package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"xui_log_archiver/archiver"
	"xui_log_archiver/installer"
)

func main() {
	// Проверяем, запущена ли программа с аргументом для cron
	if len(os.Args) > 1 && os.Args[1] == "--cron" {
		runArchiving()
		return
	}

	// Интерактивное меню
	showMenu()
}

func showMenu() {
	for {
		fmt.Println("\n=== X-UI Log Archiver ===")
		fmt.Println("1. Сделать архивирование сейчас")
		fmt.Println("2. Добавить в автозапуск (каждые 10 минут, архивирование в 00 минут часа)")
		fmt.Println("3. Удалить из автозапуска")
		fmt.Println("4. Показать статус автозапуска")
		fmt.Println("0. Выход")
		fmt.Print("\nВыберите действие (0-4): ")

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Println("\nЗапуск архивирования...")
			runArchiving()
		case "2":
			installAutostart()
		case "3":
			removeAutostart()
		case "4":
			showAutostartStatus()
		case "0":
			fmt.Println("До свидания!")
			return
		default:
			fmt.Println("Неверный выбор. Попробуйте снова.")
		}
	}
}

func runArchiving() {
	arch := archiver.New()
	if err := arch.RunArchiving(); err != nil {
		fmt.Printf("Ошибка архивирования: %v\n", err)
	}
}

func installAutostart() {
	inst := installer.New()
	if err := inst.InstallAutostart(); err != nil {
		fmt.Printf("Ошибка установки автозапуска: %v\n", err)
	}
}

func removeAutostart() {
	inst := installer.New()
	if err := inst.RemoveAutostart(); err != nil {
		fmt.Printf("Ошибка удаления автозапуска: %v\n", err)
	}
}

func showAutostartStatus() {
	inst := installer.New()
	inst.ShowAutostartStatus()
}
