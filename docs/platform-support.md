# Platform Support Matrix

## Validation Priority

1. macOS
2. Windows ARM64
3. Windows x64
4. Linux

Decision 6: Windows on ARM 驅動驗證列為先行風險排除工作。

## Current Validation Window

- 四個平台的安裝包都已隨 GitHub Release 發佈（public 最新為 v0.1.20，2026-03-12）；**發佈安裝包不等於完成驗證**
- 已有正式驗證紀錄：僅 macOS Apple Silicon（見下方 Validation Snapshot）
- Windows x64、Windows ARM64、Linux x64：已出安裝包、實機驗證中
- 狀態改為 Validated 的條件：依 [Platform Validation Playbook](platform-validation-playbook.md) 完成 Shared Success Criteria，並把日期、reader、結果寫進 Validation Snapshot
- 在 Windows ARM64 完成前，不調整其 high-risk 標記

## Support Matrix

| Platform | Architecture  | Installer                | Status                       | Notes                                                                                                   |
| -------- | ------------- | ------------------------ | ---------------------------- | ------------------------------------------------------------------------------------------------------- |
| macOS    | Apple Silicon | `.pkg`（arm64）          | Validated (read/write)       | 已驗證列 reader、建立 session、讀 UID/ATR，並可對 NDEF formatted Type 2 tag 執行受控 `ndef-v1` 實體寫入 |
| macOS    | Intel         | 無                       | Not supported                | 目前只產出 arm64 binary，未驗證                                                                         |
| Windows  | x64           | `.msi`                   | 已出安裝包、驗證中           | 需驗證 PC/SC 與基本讀卡流程                                                                             |
| Windows  | ARM64         | `.msi`                   | 已出安裝包、驗證中（高風險） | 需先確認 CCID / PCSC 或 ACS 驅動可用性；SCardSvr 異常時改走 Direct IOCTL driver                         |
| Linux    | x64           | `.deb`（Ubuntu / dpkg）  | 已出安裝包、驗證中           | 需驗證 pcsc-lite 與裝置相容性                                                                           |

## Development Observations（非正式驗證）

以下是開發過程中觀察到、已反映在程式碼的現象，只作為風險紀錄，**不構成支援聲明**：

- Windows ARM64：SCardSvr 的 LRPC endpoint 可能一直無法連線，因此加入 Direct IOCTL driver（`16e645f`）
- Windows（ARM64 與部分 x64）：PC/SC context 正常但只列出通用智慧卡讀取裝置，無法處理 NFC APDU；connector 會改試 Direct IOCTL driver（`23f8a55`）
- Linux：未安裝或未啟動 `pcscd` 時 connector 以 degraded 狀態啟動並在背景重試（`0457aa4`）；`.deb` 已宣告 `pcscd` 相依（`4f62039`）

## Known Risks

- Requirement: Cross-Platform Support Matrix
  - 所有平台支援聲明都必須建立在實機驗證之上
- 驅動差異是主要平台風險
  - Windows ARM64 為最優先排雷目標
- 未驗證完成前，不可對外宣稱完整支援
  - README 的平台表格與本文件同步，不得標示比本文件更高的狀態

## Latest Validation Snapshot

- macOS 2026-03-09：`ACS ACR1252 Dual Reader PICC` 可成功建立 session 並讀得 UID `0472650DCC2A81` 與 ATR `3B8F8001804F0CA0000003060300030000000068`
- macOS 2026-03-09：`/card/write` 以 `ndef-v1` demo payload 對真卡回傳 `accepted: true`，Connector details 包含 `driver=pcsc`、`profile=ndef-write-profile/v1`、`payloadType=web-nfc-bridge/demo`、`pagesWritten=30`
- macOS 2026-03-09：實體讀回 page 4 起的資料可見 Type 2 TLV 與 NDEF MIME record 前綴：`03 74 D2 10 61 61 70 70 6C 69 63 61 74 69 6F 6E ...`，代表卡片已寫入 `application/json` payload

## Browser Policy

- 正式支援的前提是瀏覽器可穩定連線到 localhost Connector
- 不以 `Web NFC` 或 `WebUSB` 作為支援基礎

## Installer Targets

- Local build emits the artifacts the host toolchain supports (Windows MSI needs a Windows host with `wix`)
- Default version comes from git tag or `package.json` version when available
- macOS: PKG
- Windows x64: MSI on Windows hosts with `wix`
- Windows ARM64: MSI on Windows hosts with `wix`
- Linux x64: Debian package for Ubuntu / `dpkg`
- Ubuntu `.deb` installs a system-level `systemd` service and starts it automatically
