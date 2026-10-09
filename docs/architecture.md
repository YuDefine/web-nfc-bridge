# NFC Web Platform Architecture

## Scope Mapping

- Requirement: Local Connector Architecture
  - Web 前端負責使用者流程與狀態呈現
  - localhost Connector 負責與 ACR1252U-M1 溝通
  - PC/SC 為唯一正式硬體通訊層
- Requirement: Low-Friction User Experience
  - 首次安裝由 Web 前端引導完成
  - 後續回訪由首頁直接偵測 Connector 狀態
- Decision 1: 產品架構採用 HTTPS 網站 + localhost Connector + PC/SC
  - HTTPS 網站與 localhost Connector 為固定分層
- Decision 3: Web 前端使用 Nuxt + Nuxt UI
  - Web app 建立於 repo root
  - UI 使用 `Nuxt UI` 元件與 Tailwind CSS v4 樣式層
- Decision 2: 本機 Connector 使用 Go 實作
  - Go module 建立於 repo root
  - Connector 入口為 `connector/cmd/nfc-connector`

## System Layers

1. Browser
2. HTTPS Web App
3. localhost Connector
4. PC/SC
5. ACR1252U-M1

## Current Repository Layout

- `docs/`: 架構、契約與支援文件
- `pages/`, `composables/`, `assets/`: Nuxt 前端
- `connector/`: Go Connector service
- `server/api/`: Web app server routes for connector ticket 與 console bootstrap

## Explicit Non-Primary Paths

- `Web NFC` 不作為桌面產品主路徑
- `WebUSB` 不作為正式產品主路徑
