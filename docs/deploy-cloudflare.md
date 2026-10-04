# Cloudflare Wrangler 部署

這個專案的 Nuxt 站台可以直接部署到 Cloudflare Workers。部署流程分成兩條：

- 本機用 Wrangler 做 build 與預覽。
- GitHub Actions 用 Wrangler 發佈，並用 gh CLI 觸發與追蹤。

## 本機測試

先安裝依賴：

```bash
pnpm install
```

Cloudflare preset build：

```bash
pnpm run build:cf
```

本機用 Wrangler 預覽 Workers 版本：

```bash
NUXT_CONNECTOR_SHARED_SECRET=development-shared-secret pnpm run preview:cf
```

這會先做 `build:cf`，再執行 `wrangler --cwd .output dev`，直接使用 Nitro 自動產生的 Wrangler 設定。

如果要直接部署：

```bash
CLOUDFLARE_API_TOKEN=...
CLOUDFLARE_ACCOUNT_ID=...
NUXT_CONNECTOR_SHARED_SECRET=...
NUXT_PUBLIC_SITE_ORIGIN=https://nfc.yudefine.com.tw
pnpm run deploy:cf
```

這會執行 `wrangler --cwd .output deploy`，與 Nitro 官方 Cloudflare Workers 流程一致。

## GitHub Actions 與 gh CLI

Workflow 檔案位於 `.github/workflows/deploy-cloudflare.yml`。它會：

- 在 `main` push 時自動部署。
- 只在 `YuDefine/web-nfc-bridge` 執行；下游 fork 保留同一份檔案，job 會被略過（見 [release-process.md](release-process.md)）。
- 支援 `workflow_dispatch`，可用 gh CLI 對指定 branch 手動部署。
- 允許覆寫 Worker 名稱與 `site_origin`，方便先做預覽測試。

### 1. 設定 GitHub secrets 與 variables

先在 repo 根目錄登入 GitHub CLI：

```bash
gh auth login
```

設定 secrets：

```bash
gh secret set CLOUDFLARE_API_TOKEN
gh secret set CLOUDFLARE_ACCOUNT_ID
gh secret set NUXT_CONNECTOR_SHARED_SECRET
```

設定 variables：

```bash
gh variable set CLOUDFLARE_WORKER_NAME --body "web-nfc-bridge"
gh variable set NUXT_PUBLIC_CONNECTOR_BASE_URL --body "http://127.0.0.1:42619"
gh variable set NUXT_PUBLIC_SITE_ORIGIN --body "https://nfc.yudefine.com.tw"
```

### 2. 用 gh CLI 觸發預覽部署

對某個 branch 做手動部署：

```bash
gh workflow run deploy-cloudflare.yml \
  --ref your-branch \
  -f ref=your-branch \
  -f worker_name=web-nfc-bridge-preview \
  -f site_origin=https://web-nfc-bridge-preview.workers.dev
```

查看執行狀態：

```bash
gh run list --workflow deploy-cloudflare.yml --limit 5
gh run watch
```

### 3. 驗證部署站台

部署完成後，先確認這幾件事：

- 首頁可正常載入。
- `/api/console-bootstrap` 可回傳 installer 清單。
- `/api/connector-ticket` 可簽出 token。
- 前端仍使用 `http://127.0.0.1:42619` 連本機 Connector，因此可直接拿 Windows、macOS、Linux 客戶端去測同一個站台。

### 4. Connector allowlist

如果前端已部署到 Cloudflare Workers 或自訂網域，本機 Connector 的 `NFC_CONNECTOR_ALLOWED_ORIGINS` 不能只留 localhost。至少要包含部署站台 origin，例如：

```text
http://localhost:*,https://localhost:*,http://127.0.0.1:*,https://127.0.0.1:*,https://web-nfc-bridge.abcd854884.workers.dev,https://nfc.yudefine.com.tw
```

目前新產生的安裝器預設值已包含上述 Workers URL 與自訂網域；已安裝的舊版 Connector 仍需要手動更新環境變數後重啟。

部署到其他網域時，不要修改原始碼裡的預設清單，改在 build 安裝包時帶入額外 origin（`--extra-allowed-origins` 或 repository variable `NFC_CONNECTOR_EXTRA_ALLOWED_ORIGINS`），細節見 [release-process.md](release-process.md)。

## 注意事項

- 站台部署到 Cloudflare 不代表 Connector 也被搬到雲端；Connector 仍然是各平台本機安裝程式。
- installer 清單已改成讀取建置期產生的 `public/downloads/manifest.json`，避免在 Workers runtime 讀本機檔案系統。
- `pnpm run build:cf` 之後，Nitro 會自動產生 `.output/wrangler.json`；目前 repo 根目錄的 `wrangler.jsonc` 只保留共用 metadata，實際 deploy 以 `.output` 內的設定為準。
- 若要更新下載清單，執行一次 `pnpm run installers:build`，或直接更新 `public/downloads` 後重新產生 manifest。
