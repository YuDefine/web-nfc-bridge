# web-nfc-bridge

繁體中文 | [English](README.md)

一個從網頁瀏覽器讀寫 NFC 卡片的完整範例 — 採用 localhost connector 架構，取代實驗性的 Web NFC API。

```
瀏覽器 (HTTPS)  ←→  本機 Connector (Go)  ←→  PC/SC  ←→  NFC 讀卡機
```

## 為什麼做這個

[Web NFC API](https://developer.mozilla.org/en-US/docs/Web/API/Web_NFC_API) 僅限 Chrome、僅限 Android，且仍處於實驗階段。本專案示範另一種做法：在 `127.0.0.1` 執行一個輕量 Go 服務，透過標準 PC/SC 介面將任何 Web 應用程式橋接到 USB NFC 讀卡機。

此架構可在 **macOS、Windows（x64/ARM64）及 Linux** 上運作 — 只要有瀏覽器和 USB 讀卡機的環境都能使用。

## 功能特色

- **讀卡** — 從 NFC Type 2 卡片擷取 UID 及 JSON 編碼的 NDEF 資料
- **寫卡** — 將結構化 JSON 以 NDEF 記錄寫入卡片，並回讀驗證
- **即時事件** — 透過 WebSocket 串流卡片感應、讀卡機狀態與操作結果
- **安全設計** — HMAC 簽章的短效 token、來源白名單、禁止任意 APDU 直通
- **跨平台安裝包** — 提供 macOS（Apple Silicon）、Windows x64/ARM64、Linux x64 安裝包，以 PC/SC 為主，Windows 另有 Direct IOCTL 備援驅動。目前僅 macOS 完成驗證，詳見[平台支援](#平台支援)

## 架構

```
┌───────────────────────────────────────────────┐
│  瀏覽器                                       │
│  Nuxt 4 + Nuxt UI（部署至 Cloudflare）         │
└──────────────┬────────────────────────────────┘
               │ HTTP / WebSocket
               ▼
┌──────────────────────────────────────┐
│  本機 Connector (Go)                 │
│  http://127.0.0.1:42619             │
│                                      │
│  ┌────────────┐  ┌────────────────┐  │
│  │ PCSC 驅動   │  │ Direct 驅動    │  │
│  │ (全平台)    │  │ (Win ARM64)    │  │
│  └──────┬─────┘  └───────┬────────┘  │
└─────────┼────────────────┼───────────┘
          ▼                ▼
   ┌─────────────────────────────┐
   │  NFC 讀卡機 (ACR1252U-M1)   │
   │  ISO-14443 Type 2 卡片      │
   └─────────────────────────────┘
```

## 快速開始

### 前置需求

- [Node.js](https://nodejs.org/) 22+ 及 [pnpm](https://pnpm.io/) 10+
- [Go](https://go.dev/) 1.26+
- PC/SC 相容的 NFC 讀卡機（已測試 ACS ACR1252U-M1）
- PC/SC 服務：macOS/Windows 內建；Linux 需安裝 `pcscd`

### 開發模式執行

```bash
# 安裝前端依賴
pnpm install

# 終端機 1 — 啟動 Web 應用程式
pnpm dev

# 終端機 2 — 啟動 Connector
pnpm connector:dev
```

開啟 http://localhost:3000 — Web 介面會自動偵測 Connector 並列出可用的讀卡機。

### 建置

```bash
# 建置全部（Web 應用程式 + Connector + 安裝包）
pnpm build

# 或個別建置
pnpm build:app              # Nuxt Web 應用程式（Node.js）
pnpm build:cf               # Nuxt Web 應用程式（Cloudflare Workers）
pnpm connector:build        # Go Connector 執行檔
```

### 建置安裝包

```bash
pnpm installers:build:macos          # macOS .pkg
pnpm installers:build:windows-x64    # Windows x64 .msi
pnpm installers:build:windows-arm64  # Windows ARM64 .msi
pnpm installers:build:linux-x64      # Linux .deb
```

## Connector HTTP API

Base URL：`http://127.0.0.1:42619`

| 方法 | 路徑 | 說明 |
|------|------|------|
| `GET` | `/health` | Connector 健康檢查 |
| `GET` | `/readers` | 列出可用的 NFC 讀卡機 |
| `POST` | `/session/connect` | 建立讀卡工作階段 |
| `POST` | `/card/read` | 讀取卡片 UID + NDEF 資料 |
| `POST` | `/card/write` | 寫入 NDEF 資料至卡片 |
| `WS` | `/events` | 即時事件串流 |

除 `/health` 外，所有端點皆需在 `X-Bridge-Token` header 中帶入由 Web 應用程式的 `POST /api/connector-ticket` 簽發的 token。

完整 API 規格請參閱 [docs/connector-contract.md](docs/connector-contract.md)。

## 寫卡 Payload 範例

```json
{
  "sessionId": "1710000000000000000",
  "operation": "ndef-v1",
  "payload": {
    "version": 1,
    "type": "web-nfc-bridge/demo",
    "label": "MO-20260309-001",
    "content": {
      "documentNo": "MO-20260309-001",
      "itemCode": "FG-1001",
      "workstation": "PACK-01",
      "quantity": 24,
      "status": "ready"
    },
    "updatedAt": "2026-03-09T01:35:40Z"
  }
}
```

限制條件：
- `content` 必須是單層 JSON 物件（不可巢狀），最多 8 個欄位
- 值僅允許字串、數字、布林或 null
- NDEF 編碼後上限 256 bytes
- 敏感欄位（`password`、`secret`、`email` 等）會被拒絕

## 安全模型

1. **禁止原始 APDU** — Connector 僅接受預先定義的操作，不允許任意卡片指令
2. **短效 Token** — HMAC 簽章的 ticket 60 秒過期，可指定 `read`、`write`、`events` 或 `all` 範圍
3. **來源白名單** — 僅允許白名單中的來源與 Connector 通訊
4. **Payload 驗證** — 寫卡 payload 在任何卡片 I/O 前先驗證結構、大小與禁止欄位
5. **回讀驗證** — 寫入後 Connector 會回讀卡片以確認資料完整性

## 技術棧

| 層級 | 技術 |
|------|------|
| 前端 | [Nuxt 4](https://nuxt.com/) + [Nuxt UI](https://ui.nuxt.com/) + Tailwind CSS |
| Connector | [Go](https://go.dev/) + [gorilla/websocket](https://github.com/gorilla/websocket) |
| 智慧卡 | [ebfe/scard](https://github.com/ebfe/scard)（PC/SC binding） |
| 部署 | [Cloudflare Workers](https://workers.cloudflare.com/)（僅 Web 應用程式） |

## 平台支援

下列平台都已發佈安裝包，但只有留下實機驗證紀錄的平台才標為已驗證。以 [docs/platform-support.md](docs/platform-support.md) 為準，本表不得標示比它更高的狀態。

| 平台 | 架構 | 驅動 | 狀態 |
|------|------|------|------|
| macOS | Apple Silicon | PC/SC | 已驗證（讀／寫） |
| Windows | x64 | PC/SC，Direct IOCTL 備援 | 已出安裝包、驗證中 |
| Windows | ARM64 | PC/SC，Direct IOCTL 備援 | 已出安裝包、驗證中（高風險） |
| Linux | x64 | PC/SC | 已出安裝包、驗證中 |

macOS Intel 沒有安裝包，不支援。

### 測試環境

| 平台 | 裝置 | 作業系統版本 | 驗證狀態 |
|------|------|------------|----------|
| macOS | Mac mini M4 | macOS Tahoe 26.3.1 | 已驗證 |
| Windows x64 | Intel NUC | Windows 11 25H2 | 驗證中 |
| Windows ARM64 | ASUS Zenbook A14 (Snapdragon X - X126100) | Windows 11 ARM 25H2 | 驗證中 |
| Linux x64 | — | Ubuntu Server 24.04 | 驗證中 |

- **讀卡機**：ACS ACR1252U-M1
- **卡片**：NTAG 216（NFC Type 2）

## 專案結構

```
├── pages/                  # Nuxt 頁面（Web 介面）
├── composables/            # Vue composables（Connector 通訊）
├── server/api/             # Nuxt server routes（token 簽發）
├── connector/
│   ├── cmd/nfc-connector/  # Go 進入點
│   └── internal/
│       ├── api/            # HTTP + WebSocket 伺服器
│       ├── auth/           # Token 驗證
│       └── bridge/         # 驅動層（PCSC、Direct IOCTL、NDEF 編解碼）
├── scripts/                # 安裝包建置腳本
└── docs/                   # 架構與 API 文件
```

## 設定

Connector 環境變數：

| 變數 | 預設值 | 說明 |
|------|--------|------|
| `NFC_CONNECTOR_ADDR` | `127.0.0.1:42619` | Connector 監聽位址 |
| `NFC_CONNECTOR_SHARED_SECRET` | `development-shared-secret` | HMAC 簽章金鑰 |
| `NFC_CONNECTOR_ALLOWED_ORIGINS` | localhost／127.0.0.1（任意 port）＋ public demo 站台，以及 build 時加入的 origin | 以逗號分隔的來源白名單；有設定時取代預設值 |

## 文件

- [架構](docs/architecture.md) — 系統分層與設計決策
- [Connector 契約](docs/connector-contract.md) — 完整 HTTP/WebSocket API 規格
- [Connector 操作](docs/connector-operations.md) — 運作細節
- [平台支援](docs/platform-support.md) — 平台驗證矩陣

## Credits

Built by [Yudefine - 域定資訊工作室](https://yudefine.com.tw)

## License

[MIT](LICENSE)
