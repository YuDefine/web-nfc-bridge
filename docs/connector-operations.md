# Connector Operations

## Packaging Plan

- Local build 會固定輸出四個平台檔案
- 未指定 `--version` 時，預設會依序讀取 git tag、`package.json` version，再退回保底版本
- macOS Apple Silicon: `.pkg`，內含 arm64 connector binary
- Windows x64: 只有在 Windows host 且可用 `wix` 時才產生 `.msi`
- Windows ARM64: 只有在 Windows host 且可用 `wix` 時才產生 `.msi`
- Linux x64: 產生 Ubuntu `dpkg` 可安裝的 `.deb`
- 每次 build 完成後，`public/downloads` 只保留各平台最新版本的產物
- 額外允許的網站 origin 以 `--extra-allowed-origins`（或環境變數 `NFC_CONNECTOR_EXTRA_ALLOWED_ORIGINS`）在 build 時帶入：寫進 macOS plist、Linux `/etc/default`，並以 `-ldflags -X main.extraAllowedOrigins` 編進 binary（Windows MSI 不設環境變數，只能靠這個預設值）；未帶入時只含 public origin

## Startup Strategy

- macOS: `launchd` agent
- Windows: Run key or Windows Service，視最終權限模型決定
- Linux: system-level `systemd` service，安裝 `.deb` 後自動 `enable --now`

## Windows Upgrade & Logging

- MSI 使用 `MajorUpgrade`（`afterInstallInitialize`），升級時先移除舊版；偵測到升級（`WIX_UPGRADE_DETECTED`）時以 `taskkill /F /IM nfc-connector.exe` 停掉執行中的 connector，安裝完成後再以 `--watchdog` 啟動新版
- Windows build 使用 `-H=windowsgui`，沒有 console；log 寫到 `%LOCALAPPDATA%\Web NFC Bridge Connector\connector.log`
- watchdog 啟動時若 log 超過 1 MB，改名為 `connector.log.old` 再開新檔
- watchdog 子行程沿用 watchdog 的 log 檔（`NFC_CONNECTOR_SUPERVISED=1`），未處理的 panic 與 runtime fatal error 也會寫進同一個檔
- 下游 fork 的 v0.1.23 曾把 log 寫在 `%APPDATA%`（Roaming）下的同名資料夾；升級後可手動刪除

## Ubuntu `.deb` Behavior

- 安裝路徑：`/usr/local/libexec/web-nfc-bridge/nfc-connector`
- 命令包裝：`/usr/local/bin/web-nfc-bridge-connector`
- systemd unit：`/lib/systemd/system/web-nfc-bridge-connector.service`
- 預設環境檔：`/etc/default/web-nfc-bridge-connector`
- `postinst` 會執行 `systemctl daemon-reload`、`enable` 與 `start/restart`
- `prerm` 會在移除前停止並停用 service
- `postrm` 會在移除後執行 `daemon-reload` 與 `reset-failed`

## macOS Upgrade Safety

- macOS pkg install 前會先執行 `preinstall`，卸載既有 `com.web-nfc-bridge.connector` agent
- `preinstall` 會移除 `/usr/local/libexec/web-nfc-bridge/nfc-connector`
- `preinstall` 會移除 `/usr/local/bin/web-nfc-bridge-connector`
- `preinstall` 會移除 `/Library/LaunchAgents/com.web-nfc-bridge.connector.plist`
- 安裝完成後再由 `postinstall` 重新 bootstrap 新版 agent
- 這代表同一路徑上的舊 binary 與舊 plist 不會在升級後殘留

## Auto-Update Strategy

- Connector 啟動時檢查版本清單
- 只下載與當前平台匹配的安裝包
- 預設靜默下載，下一次重啟套用
- 寫卡流程進行中不可強制更新

## Troubleshooting

### Connector offline

- 確認 localhost base URL 與頁面顯示一致
- 確認 Connector process 已啟動
- 確認 `NFC_CONNECTOR_ALLOWED_ORIGINS` 包含網站 origin
- Windows 上先看 `%LOCALAPPDATA%\Web NFC Bridge Connector\connector.log`
- 開發模式可使用 `http://localhost:*`、`https://localhost:*`、`http://127.0.0.1:*` 這類 wildcard 規則允許本機不同 port
- 若 Web console 部署在 Cloudflare Workers 或自訂網域，allowlist 也要包含正式站台 origin，例如 `https://web-nfc-bridge.abcd854884.workers.dev` 或 `https://nfc.yudefine.com.tw`

### Reader missing

- 確認 ACR1252U-M1 已連接
- 確認作業系統 PC/SC 服務正常
- 在 macOS 上可先用 Connector `/readers` 驗證是否列出裝置
- ACR1252U-M1 在 macOS 可能列出兩個 logical readers：`ACS ACR1252 Dual Reader SAM` 與 `ACS ACR1252 Dual Reader PICC`；NFC 卡片應放在 `PICC` reader 驗證
- macOS `launchd` 常駐模式下，不要把 `/readers` 的 `cardPresent` 當成硬性前置條件；目前以實際 `session/connect`、`card/read`、`card/write` 重試結果作為是否已貼卡的判斷，避免 ACR1252 的 PC/SC `Connect` 探測把 reader 枚舉卡住

### Card not detected

- 若 `/session/connect` 回傳 `No smart card inserted`，先確認卡片是否實際放在 `PICC` 感應區
- 可先用 `/readers` 檢查 `ACS ACR1252 Dual Reader PICC` 的 `cardPresent` 是否為 `true`
- 已於 macOS 實測驗證：當 `cardPresent=true` 時，可成功建立 session 並透過 `/card/read` 取得 UID 與 ATR

### Token rejected

- 確認 Nuxt server 與 Connector 使用同一組 shared secret
- 確認 token 沒過期
- 確認 Web app origin 與 ticket 中的 origin 完全一致

### Write rejected

- 確認請求使用 `ndef-v1` profile，而不是舊的 `demo-note` payload
- 確認 payload type 為 `web-nfc-bridge/demo` 或 `web-nfc-bridge/ref`
- `web-nfc-bridge/ref` 必須帶 `token`，`web-nfc-bridge/demo` 只允許小型 demo 欄位
- 若錯誤訊息提到 safe limit，請縮小 JSON payload 到 `256` bytes 以內
- 若錯誤訊息提到 `card is not NDEF formatted`，代表目前卡片不在 v1 支援範圍內
- 若錯誤訊息提到 `card is NDEF read-only`，代表卡片 Capability Container 指出目前不可寫
- 若錯誤訊息提到 `payload requires ... bytes`，代表卡片 user data area 小於這次 TLV 所需空間
- 若錯誤訊息提到 `card rejected write at first data page 4: 63 00`，代表卡片在第一個 user data page 就拒絕寫入；通常是卡片已被 lock、實際不可寫，或需要先重新格式化
- 若錯誤訊息提到 `unexpected read response` 或 `unexpected write response`，代表讀卡機或卡片對該次 PC/SC page I/O 沒有給出成功狀態，應先重新放卡再試

### Write completed

- 在 macOS `pcsc` driver 上，Connector 只有在 Type 2 page write 完成且 read-back verify 通過後，才會回傳 `accepted: true`
- 成功回應會包含 `profile`、`payloadType`、`ndefBytes`、`tlvBytes` 與 `pagesWritten`，可用來核對本次實體寫入範圍
