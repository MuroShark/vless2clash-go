# 🚀 VLESS to Clash Converter

[![Ru](https://img.shields.io/badge/lang-Ru-red.svg)](README_RU.md)
![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

A simple and fast CLI utility written in Go to convert VLESS links and subscriptions into **Clash** configuration format (YAML).

## ✨ Features

- 🔗 **Direct Links**: Convert single `vless://` links.
- 📦 **Subscriptions**: Download and parse Base64 subscriptions via URL.
- 🎨 **Clash Config**: Automatically generate a valid `config.yaml` with rules and groups.
- 🛠 **Cross-platform**: Works on macOS, Linux, and Windows.

## 📥 Installation

### From Source

You need Go installed.

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/vless2clash-go.git
   cd vless2clash-go
   ```

2. Run the program:
   ```bash
   go run main.go
   ```

3. Or build the binary:
   ```bash
   go build -o vless2clash
   ./vless2clash
   ```

## 📖 Usage

Run the utility and follow the interactive menu:

```text
__      __  _      ______   _____   _____ 
\ \    / / | |    |  ____| / ____| / ____|
 \ \  / /  | |    | |__   | (___  | (___  
  \ \/ /   | |    |  __|   \___ \  \___ \ 
   \  /    | |____| |____  ____) | ____) |
    \/     |______|______| |_____/ |_____/ 

       VLESS to Clash Converter v1.0
===========================================

 [1] Convert direct link (vless://)
 [2] Convert subscription via URL
 [3] Exit
```

The result will be saved to the `vless_generated.yaml` file in the current directory.

## 📄 Example Result

The generated `vless_generated.yaml` file looks like this:

```yaml
port: 7890
socks-port: 7891
allow-lan: true
mode: rule
log-level: info
external-controller: :9090

proxies:
  - name: "My VLESS Server"
    type: vless
    server: example.com
    port: 443
    uuid: 12345678-1234-1234-1234-1234567890ab
    network: ws
    tls: true
    udp: true
    flow: 
    servername: example.com
    client-fingerprint: chrome

proxy-groups:
  - name: Proxy
    type: select
    proxies:
      - "My VLESS Server"
      - DIRECT

rules:
  - MATCH,Proxy
```

## ⚙️ Technical Details

- Supported parameters: `tls`, `reality`, `flow`, `sni`, `fp`.
- Automatic name decoding (URL decode).
- Templating via `text/template`.

## 📄 License

MIT License.