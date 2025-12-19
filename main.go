package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// Цветовые коды для оформления
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
)

// Структура прокси
type Proxy struct {
	Name              string
	Type              string
	Server            string
	Port              string
	UUID              string
	Network           string
	TLS               bool
	UDP               bool
	Flow              string
	ClientFingerprint string
	ServerName        string
}

// Шаблон для вывода (Здесь мы настраиваем внешний вид YAML)
// Обратите внимание: {{ .Name }} вставит текст как есть, без экранирования
const clashTemplate = `port: 7890
socks-port: 7891
allow-lan: true
mode: rule
log-level: info
external-controller: :9090

proxies:
{{- range . }}
  - name: "{{ .Name }}"
    type: vless
    server: {{ .Server }}
    port: {{ .Port }}
    uuid: {{ .UUID }}
    network: {{ .Network }}
    tls: {{ .TLS }}
    udp: {{ .UDP }}
    flow: {{ .Flow }}
    servername: {{ .ServerName }}
    client-fingerprint: {{ .ClientFingerprint }}
{{- end }}

proxy-groups:
  - name: Proxy
    type: select
    proxies:
{{- range . }}
      - "{{ .Name }}"
{{- end }}
      - DIRECT

rules:
  - MATCH,Proxy
`

// Конфигурация приложения
type AppConfig struct {
	Language string `json:"language"`
}

// Глобальная переменная языка
var currentLang = "en"

// Словарь локализации
var messages = map[string]map[string]string{
	"en": {
		"menu_title": "       VLESS to Clash Converter v1.0",
		"menu_opt1":  "Convert direct link (vless://)",
		"menu_opt2":  "Convert subscription via URL",
		"menu_opt3":  "Exit",
		"ask_choice": "➔ Select action: ",
		"ask_link":   "Enter vless:// link: ",
		"ask_url":    "Enter subscription URL: ",
		"exit":       "Exiting...",
		"invalid":    "Invalid choice, please try again.",
		"err_parse":  "Parsing error: %v",
		"err_tmpl":   "Template error: %v",
		"err_gen":    "Config generation error: %v",
		"err_write":  "File write error: %v",
		"success":    "✔ File successfully generated: %s",
		"loading":    "Loading subscription...",
		"err_load":   "URL load error: %v",
		"err_read":   "Read error: %v",
		"err_b64":    "Base64 decode error: %v",
		"found":      "Proxies found: %d",
		"no_links":   "No valid vless links found.",
		"pause":      "\nPress Enter to continue...",
		"lang_sel":   "Select Language / Выберите язык",
	},
	"ru": {
		"menu_title": "       VLESS to Clash Converter v1.0",
		"menu_opt1":  "Конвертировать прямую ссылку (vless://)",
		"menu_opt2":  "Конвертировать подписку по URL",
		"menu_opt3":  "Выход",
		"ask_choice": "➔ Выберите действие: ",
		"ask_link":   "Введите ссылку vless://: ",
		"ask_url":    "Введите URL подписки: ",
		"exit":       "Выход...",
		"invalid":    "Неверный выбор, попробуйте снова.",
		"err_parse":  "Ошибка парсинга: %v",
		"err_tmpl":   "Ошибка шаблона: %v",
		"err_gen":    "Ошибка генерации конфига: %v",
		"err_write":  "Ошибка записи файла: %v",
		"success":    "✔ Файл успешно сгенерирован: %s",
		"loading":    "Загрузка подписки...",
		"err_load":   "Ошибка загрузки URL: %v",
		"err_read":   "Ошибка чтения ответа: %v",
		"err_b64":    "Ошибка декодирования Base64: %v",
		"found":      "Найдено прокси: %d",
		"no_links":   "Не найдено валидных vless ссылок.",
		"pause":      "\nНажмите Enter, чтобы продолжить...",
		"lang_sel":   "Select Language / Выберите язык",
	},
}

// Функция перевода
func t(key string, args ...interface{}) string {
	msg, ok := messages[currentLang][key]
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	// Инициализация языка (загрузка или выбор)
	initLanguage(scanner)

	for {
		// Очистка экрана (ANSI escape codes)
		clearScreen()
		fmt.Println(ColorCyan + `
__      __  _      ______   _____   _____ 
\ \    / / | |    |  ____| / ____| / ____|
 \ \  / /  | |    | |__   | (___  | (___  
  \ \/ /   | |    |  __|   \___ \  \___ \ 
   \  /    | |____| |____  ____) | ____) |
    \/     |______|______| |_____/ |_____/ ` + ColorReset)
		fmt.Println(ColorCyan + "\n" + t("menu_title") + ColorReset)
		fmt.Println(ColorCyan + "===========================================" + ColorReset)
		fmt.Println("")
		fmt.Println(" " + ColorBold + "[1]" + ColorReset + " " + t("menu_opt1"))
		fmt.Println(" " + ColorBold + "[2]" + ColorReset + " " + t("menu_opt2"))
		fmt.Println(" " + ColorBold + "[3]" + ColorReset + " " + ColorRed + t("menu_opt3") + ColorReset)
		fmt.Println("")
		fmt.Print(ColorYellow + t("ask_choice") + ColorReset)

		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			fmt.Print("\n" + ColorYellow + t("ask_link") + ColorReset)
			if !scanner.Scan() {
				break
			}
			rawLink := strings.TrimSpace(scanner.Text())
			if rawLink == "" {
				continue
			}

			proxy, err := ParseVless(rawLink)
			if err != nil {
				fmt.Printf(ColorRed + t("err_parse", err) + ColorReset + "\n")
			} else {
				saveConfig([]Proxy{proxy})
			}
			pause(scanner)

		case "2":
			fmt.Print("\n" + ColorYellow + t("ask_url") + ColorReset)
			if !scanner.Scan() {
				break
			}
			urlStr := strings.TrimSpace(scanner.Text())
			if urlStr == "" {
				continue
			}
			processSubscription(urlStr)
			pause(scanner)

		case "3":
			fmt.Println(ColorGreen + t("exit") + ColorReset)
			return

		default:
			fmt.Println(ColorRed + t("invalid") + ColorReset)
			pause(scanner)
		}
	}
}

func saveConfig(proxies []Proxy) {
	// 2. Генерируем красивый YAML через шаблон
	tmpl, err := template.New("proxy").Parse(clashTemplate)
	if err != nil {
		fmt.Printf(ColorRed + t("err_tmpl", err) + ColorReset + "\n")
		return
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, proxies)
	if err != nil {
		fmt.Printf(ColorRed + t("err_gen", err) + ColorReset + "\n")
		return
	}

	// 3. Сохраняем результат в файл (в текущей директории)
	outputFile := "vless_generated.yaml"
	if err := os.WriteFile(outputFile, output.Bytes(), 0644); err != nil {
		fmt.Printf(ColorRed + t("err_write", err) + ColorReset + "\n")
		return
	}

	fmt.Printf(ColorGreen + t("success", outputFile) + ColorReset + "\n")
}

func processSubscription(urlStr string) {
	fmt.Println(ColorCyan + t("loading") + ColorReset)
	resp, err := http.Get(urlStr)
	if err != nil {
		fmt.Printf(ColorRed + t("err_load", err) + ColorReset + "\n")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf(ColorRed + t("err_read", err) + ColorReset + "\n")
		return
	}

	// Очистка и декодирование Base64
	encoded := strings.TrimSpace(string(body))
	// Пытаемся декодировать стандартным способом, если ошибка - пробуем Raw (без паддинга)
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decodedBytes, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			fmt.Printf(ColorRed + t("err_b64", err) + ColorReset + "\n")
			return
		}
	}

	lines := strings.Split(string(decodedBytes), "\n")
	var proxies []Proxy

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vless://") {
			if p, err := ParseVless(line); err == nil {
				proxies = append(proxies, p)
			}
		}
	}

	if len(proxies) > 0 {
		fmt.Printf(ColorGreen + t("found", len(proxies)) + ColorReset + "\n")
		saveConfig(proxies)
	} else {
		fmt.Println(ColorRed + t("no_links") + ColorReset)
	}
}

// Вспомогательная функция для паузы
func pause(scanner *bufio.Scanner) {
	fmt.Println(t("pause"))
	scanner.Scan()
}

// Функция очистки экрана (кроссплатформенная)
func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

// Функция очистки (Unescape x2)
func cleanName(raw string) string {
	result := raw
	for i := 0; i < 2; i++ {
		decoded, err := url.QueryUnescape(result)
		if err != nil {
			return result
		}
		if decoded == result {
			break
		}
		result = decoded
	}
	return result
}

func ParseVless(link string) (Proxy, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Proxy{}, err
	}

	query := u.Query()
	decodedName := cleanName(u.Fragment)

	p := Proxy{
		Name:              decodedName,
		Type:              "vless",
		Server:            u.Hostname(),
		Port:              u.Port(),
		UUID:              u.User.Username(),
		Network:           query.Get("type"),
		TLS:               query.Get("security") == "tls" || query.Get("security") == "reality",
		UDP:               true,
		ClientFingerprint: query.Get("fp"),
		ServerName:        query.Get("sni"),
		Flow:              query.Get("flow"),
	}

	return p, nil
}

// Функция инициализации языка
func initLanguage(scanner *bufio.Scanner) {
	// Используем системную папку конфигурации (AppData/Application Support),
	// чтобы настройки сохранялись глобально и не мусорили рядом с exe файлом.
	configDir, err := os.UserConfigDir()
	configFile := "settings.json" // Fallback на текущую папку, если системная недоступна
	if err == nil {
		appDir := filepath.Join(configDir, "vless2clash")
		_ = os.MkdirAll(appDir, 0755) // Создаем папку, если её нет
		configFile = filepath.Join(appDir, "settings.json")
	}

	// 1. Пытаемся загрузить конфиг
	if data, err := os.ReadFile(configFile); err == nil {
		var cfg AppConfig
		if err := json.Unmarshal(data, &cfg); err == nil && (cfg.Language == "en" || cfg.Language == "ru") {
			currentLang = cfg.Language
			return
		}
	}

	// 2. Если конфига нет или он битый, спрашиваем пользователя
	clearScreen()
	fmt.Println(ColorCyan + "Select Language / Выберите язык" + ColorReset)
	fmt.Println("[1] English")
	fmt.Println("[2] Русский")
	fmt.Print(ColorYellow + "➔ " + ColorReset)

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "1" {
			currentLang = "en"
			break
		} else if text == "2" {
			currentLang = "ru"
			break
		}
		fmt.Print(ColorYellow + "➔ " + ColorReset)
	}

	// 3. Сохраняем выбор
	cfg := AppConfig{Language: currentLang}
	if data, err := json.Marshal(cfg); err == nil {
		_ = os.WriteFile(configFile, data, 0644)
	}
}
