package main

import (
	"errors"
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

//go:embed web/*
var webContent embed.FS

// Проверка версии Ansible
func checkAnsibleVersion() error {
	cmd := exec.Command("ansible", "--version")
	output, err := cmd.Output()

	if err != nil {
		return err // Ни python3, ни python не найдены
	}

	versionStr := string(output)

	// Извлекаем версию вида X.Y.Z
	re := regexp.MustCompile(`core (\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		log.Printf("unable to dertermine Ansible: %s", versionStr)
		return errors.New("Unknown Ansible version")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	// Формируем числовой аналог версии: 3.8.0 → 30800, 3.12.1 → 31201
	versionNum := major*10000 + minor*100 + patch
	const (
		minVersion = 21100 // 3.8.0
		maxTested = 21899 // Условно: до 3.11.99 (т. е. 3.11.x)
	)

	if versionNum < minVersion {
		log.Printf(
			"Ansible ≥ 2.11 is requires. Version %d.%d.%d is located. Please, install newer version of Ansible.",
			major, minor, patch,
		)
		return errors.New("Wrong Ansible version")
	}

	if versionNum > maxTested {
		log.Printf(
			"[WARNING] Ansible %d.%d.%d was not tested yet. Possible compatibility problmes.",
			major, minor, patch,
		)
		// Продолжаем, но с предупреждением
	} else {
		log.Printf("Ansible %d.%d.%d is compatible.", major, minor, patch)
	}
	return nil
}

// Проверка версии Python
func checkPythonVersion() error {
	// Пытаемся выполнить python3, затем python (в порядке приоритета)
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()

	if err != nil {
		// Если python3 не найден, пробуем python
		cmd = exec.Command("python", "--version")
		output, err = cmd.Output()
		if err != nil {
			return err // Ни python3, ни python не найдены
		}
	}

	versionStr := string(output)

	// Извлекаем версию вида X.Y.Z
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		log.Printf("unable to dertermine Python: %s", versionStr)
		return errors.New("Unknown python version")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	// Формируем числовой аналог версии: 3.8.0 → 30800, 3.12.1 → 31201
	versionNum := major*10000 + minor*100 + patch

	const (
		minVersion = 30800 // 3.8.0
		maxTested = 31399 // Условно: до 3.11.99 (т. е. 3.11.x)
	)

	if versionNum < minVersion {
		log.Printf(
			"Python ≥ 3.8 is requires. Version %d.%d.%d is located. Please, install newer version of Python.",
			major, minor, patch,
		)
		return errors.New("Wrong Python version")
	}

	if versionNum > maxTested {
		log.Printf(
			"[WARNING] Python %d.%d.%d was not tested yet. Possible compatibility problmes.",
			major, minor, patch,
		)
		// Продолжаем, но с предупреждением
	} else {
		log.Printf("Python %d.%d.%d is compatible.", major, minor, patch)
	}
	return nil
}

func main() {

	// 1. Проверяем версию Python перед стартом сервера
	if err := checkPythonVersion(); err != nil {
		log.Fatalf("Unable to check Python: %v", err)
	}
	if err := checkAnsibleVersion(); err != nil {
		log.Fatalf("Unable to check Ansible: %v", err)
	}

	port := "8080"
	url := "http://localhost:" + port + "/static/web"

	mux := http.NewServeMux()

	// Статические файлы (HTML, CSS, JS)
	fs := http.FileServer(http.FS(webContent))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/index.html", http.StatusFound)
	})

	// API: проверка системы
	mux.HandleFunc("/api/check-system", func(w http.ResponseWriter, r *http.Request) {
		var memOk bool
		// Здесь должна быть реальная проверка памяти (например, через sysinfo)
		memOk = true // упрощённо

		result := map[string]interface{}{
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"memoryOk": memOk,
			"canInstall": memOk, // можно устанавливать, если всё ок
		}
		json.NewEncoder(w).Encode(result)
	})

	// API: запуск установки
	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]int
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		nodes := req["nodes"]
		port := req["port"]

		log.Printf("Запуск установки: %d узлов, порт %d", nodes, port)

		// Здесь вызываются реальные команды установки YDB
		// Например: exec.Command("ydb", "...")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "installing",
			"msg":    "Установка запущена",
		})
	})

	// API: завершение работы сервера
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Получен запрос на завершение работы")
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
		w.WriteHeader(http.StatusNoContent)
	})

	log.Printf("YDB Installer is running on %s\n",url)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		log.Printf("Unable to open browser automatically: %s", runtime.GOOS)
		return
	}

	// Выполняем команду
	if err := cmd.Start(); err != nil {
		log.Printf("Не удалось открыть браузер: %v", err)
		return
	}

	log.Fatal(http.ListenAndServe(":"+port, mux))

}
