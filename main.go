package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func main() {
	scanner := bufio.NewScanner(os.Stdin)

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
		fmt.Println(ColorCyan + "\n       VLESS to Clash Converter v1.0" + ColorReset)
		fmt.Println(ColorCyan + "===========================================" + ColorReset)
		fmt.Println("")
		fmt.Println(" " + ColorBold + "[1]" + ColorReset + " Конвертировать прямую ссылку (vless://)")
		fmt.Println(" " + ColorBold + "[2]" + ColorReset + " Конвертировать подписку по URL")
		fmt.Println(" " + ColorBold + "[3]" + ColorReset + " " + ColorRed + "Выход" + ColorReset)
		fmt.Println("")
		fmt.Print(ColorYellow + "➔ Выберите действие: " + ColorReset)

		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			fmt.Print("\n" + ColorYellow + "Введите ссылку vless://: " + ColorReset)
			if !scanner.Scan() {
				break
			}
			rawLink := strings.TrimSpace(scanner.Text())
			if rawLink == "" {
				continue
			}

			proxy, err := ParseVless(rawLink)
			if err != nil {
				fmt.Printf(ColorRed+"Ошибка парсинга: %v"+ColorReset+"\n", err)
			} else {
				saveConfig([]Proxy{proxy})
			}
			pause(scanner)

		case "2":
			fmt.Print("\n" + ColorYellow + "Введите URL подписки: " + ColorReset)
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
			fmt.Println(ColorGreen + "Выход..." + ColorReset)
			return

		default:
			fmt.Println(ColorRed + "Неверный выбор, попробуйте снова." + ColorReset)
			pause(scanner)
		}
	}
}

func saveConfig(proxies []Proxy) {
	// 2. Генерируем красивый YAML через шаблон
	tmpl, err := template.New("proxy").Parse(clashTemplate)
	if err != nil {
		fmt.Printf(ColorRed+"Ошибка шаблона: %v"+ColorReset+"\n", err)
		return
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, proxies)
	if err != nil {
		fmt.Printf(ColorRed+"Ошибка генерации конфига: %v"+ColorReset+"\n", err)
		return
	}

	// 3. Сохраняем результат в файл (в текущей директории)
	outputFile := "vless_generated.yaml"
	if err := os.WriteFile(outputFile, output.Bytes(), 0644); err != nil {
		fmt.Printf(ColorRed+"Ошибка записи файла: %v"+ColorReset+"\n", err)
		return
	}

	fmt.Printf(ColorGreen+"✔ Файл успешно сгенерирован: %s"+ColorReset+"\n", outputFile)
}

func processSubscription(urlStr string) {
	fmt.Println(ColorCyan + "Загрузка подписки..." + ColorReset)
	resp, err := http.Get(urlStr)
	if err != nil {
		fmt.Printf(ColorRed+"Ошибка загрузки URL: %v"+ColorReset+"\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf(ColorRed+"Ошибка чтения ответа: %v"+ColorReset+"\n", err)
		return
	}

	// Очистка и декодирование Base64
	encoded := strings.TrimSpace(string(body))
	// Пытаемся декодировать стандартным способом, если ошибка - пробуем Raw (без паддинга)
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decodedBytes, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			fmt.Printf(ColorRed+"Ошибка декодирования Base64: %v"+ColorReset+"\n", err)
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
		fmt.Printf(ColorGreen+"Найдено прокси: %d"+ColorReset+"\n", len(proxies))
		saveConfig(proxies)
	} else {
		fmt.Println(ColorRed + "Не найдено валидных vless ссылок." + ColorReset)
	}
}

// Вспомогательная функция для паузы
func pause(scanner *bufio.Scanner) {
	fmt.Println("\nНажмите Enter, чтобы продолжить...")
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
