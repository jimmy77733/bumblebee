# Bumblebee

獨立 fork，以 [perplexityai/bumblebee](https://github.com/perplexityai/bumblebee) 為基底，加上 Windows 支援與桌面 GUI。

Bumblebee 是唯讀的套件／擴充／開發工具中繼資料清點工具。給定一份已知曝光清單時，它只做精確的 `(ecosystem, 套件, 版本)` 比對，用來回答：這台機器的磁碟中繼資料現在有沒有命中已知項目。

這**不是** CVE 掃描、原始碼掃描、EDR，也不會執行套件管理器。清單來自公開報導，不保證即時或完整。

## Windows GUI

發佈目錄：

```
Bumblebee.exe
bumblebee-cli.exe
threat_intel\
threat_intel\.upstream-revision
```

| 按鈕 | 行為 |
|---|---|
| 掃描指定目錄 | 資料夾選擇器 → `deep` 掃描該路徑 |
| 智能掃描 | `baseline`（外掛／MCP／套件根）+ `deep` 掃描 `%USERPROFILE%` |

不會掃描整顆 C 槽。掃完會寫出離線 HTML 報告並用瀏覽器開啟。

官方 `threat_intel` 只在你確認後才下載覆蓋本機清單。程式啟動時會檢查一次，並登錄每日 09:00 的工作排程器靜默檢查；離線則沿用本機清單。

在儲存庫根目錄建置：

```powershell
powershell -File .\scripts\build-windows.ps1
```

需要 Go 1.25+，以及 Windows 上編譯 Fyne 所需的 C 編譯器（例如 WinLibs MinGW）。GUI 依賴 [Fyne](https://fyne.io/)。Windows 檔名不分大小寫，CLI 因此命名為 `bumblebee-cli.exe`，避免蓋掉 GUI。

靜默檢查清單：

```text
Bumblebee.exe --check-catalog
```

## CLI

CLI 仍可在 macOS、Linux、Windows 使用，行為與上游接近。

```sh
go build -o bumblebee ./cmd/bumblebee
go test ./...
bumblebee selftest
```

| Profile | 掃描範圍 |
|---|---|
| `baseline` | 常見全域／使用者套件根、工具鏈、編輯器與瀏覽器擴充、MCP 設定 |
| `project` | 設定好的開發目錄，例如 `~/code`、`~/src` |
| `deep` | 明確的 `--root`，可含使用者家目錄 |

`baseline` 與 `project` 拒絕裸家目錄；只有 `deep` 會走家目錄。

```sh
bumblebee scan --profile baseline --exposure-catalog ./threat_intel
bumblebee scan --profile deep --root "%USERPROFILE%" --exposure-catalog ./threat_intel --findings-only
bumblebee roots --profile baseline
```

輸出為 NDJSON。診斷寫到 stderr。細節見 [docs/inventory-sources.md](docs/inventory-sources.md)、[docs/transport.md](docs/transport.md)、[docs/state-model.md](docs/state-model.md)。

## 曝光清單

[`threat_intel/`](threat_intel/) 內的 JSON 依公開威脅情報維護。比對規則是精確名稱與版本；`versions: ["*"]` 可匹配該套件任何版本。GUI 只追蹤上游這個目錄，不會自動合併上游 scanner 程式碼。

## License

Apache License 2.0. See [LICENSE](LICENSE).
