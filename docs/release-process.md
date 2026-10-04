# Release Process

本專案使用 GitHub Actions 產出可安裝的 release 資產，目標平台如下：

- Windows x64：`.msi`
- Windows ARM64：`.msi`
- macOS Apple Silicon：`.pkg`
- Ubuntu x64：`.deb`

## 發版方式

使用 Git tag 作為正式發版來源，**tag 只在 public repo 的 `main` 上建立**。

1. 在 `main` 上把 `package.json` 的 `version` 改成這次的版本（例如 `0.1.24`），經 PR 合入。
2. 在 `main` 的該 commit 建立 tag，例如 `v0.1.24`。
3. 只推這一個 tag：`git push origin v0.1.24`。**不要用 `git push --tags`**，本機 clone 可能帶著下游 fork 的 tag。
4. `release-installers` workflow 建置四個平台安裝包並發佈到 GitHub Release。

版本號只能遞增。Windows MSI 以 `MajorUpgrade` 升級，裝了較新版本的機器不能再裝較舊版本。

## 上游與下游 fork

public repo（`YuDefine/web-nfc-bridge`）是上游，下游 fork 只是部署設定不同的同一份程式碼。

### 單向同步

- 程式碼只從上游流向下游：下游用 `git merge` 合入上游 `main`，不 rebase、不挑 commit。
- 通用修正一律先進上游 `main`（PR），再由下游 merge 取得。若下游先修了緊急問題，要把該修正 cherry-pick 到上游開 PR，下次 merge 時自然收斂。
- 下游不保留任何程式碼或 workflow 差異。部署差異只放在 repository variable：

| Repository variable                   | 用途                                                                    | public repo |
| ------------------------------------- | ----------------------------------------------------------------------- | ----------- |
| `NFC_CONNECTOR_EXTRA_ALLOWED_ORIGINS` | 額外允許的網站 origin，逗號分隔；由 `release-installers` 帶入安裝包 | 不設定      |

- `deploy-cloudflare` 只在 `github.repository == 'YuDefine/web-nfc-bridge'` 時執行，下游不必刪檔。
- 檢查方式：下游 merge 完上游後，`git diff <upstream>/main <downstream>/main` 應為空。

### 下游發版

下游沿用上游同一個 tag，不自行建立版本號：

1. 下游 merge 上游 `main`（需包含該 tag 指向的 commit）。
2. 把同一個 tag 推到下游 repo：`git push <downstream-remote> v0.1.24`。
3. 下游的 `release-installers` 以同一份程式碼、加上下游自己的 `NFC_CONNECTOR_EXTRA_ALLOWED_ORIGINS` 建置，發佈到下游 repo 的同名 Release。

因為額外 origin 會寫進安裝包，下游使用者必須從下游 repo 的 Release 下載，不能使用上游的安裝包。

本機建置等效指令：`node ./scripts/build-installers.mjs --platform linux-x64 --extra-allowed-origins https://example.com`（也可改用同名環境變數）。

### v0.1.20–v0.1.23 的處理

`v0.1.21`、`v0.1.22`、`v0.1.23` 是在下游 fork 線上建立的 tag，上游沒有對應的 tag 與 Release：

- 這三個版本號在上游**永久跳過**，上游 `v0.1.20` 之後的下一版是 `v0.1.24`。
- 下游既有的這三個 tag 與 Release 保留不動（已有使用者安裝，改寫會讓版本來源無從追查），也**不推到上游**。
- 它們包含的通用修正（Windows MSI 升級時停止舊 connector、Windows 檔案 log）已 cherry-pick 進上游 `main`，隨 `v0.1.24` 發佈；下游專屬的部分改由上述 repository variable 與 workflow 條件取代。
- 下游從 `v0.1.23` 升到 `v0.1.24` 時，版本號仍然遞增，Windows MSI 可正常升級。

`v0.1.20` 則是**同名異物**：上游與下游各自建立了 `v0.1.20`，指向不同 commit（下游那個已含下游專屬 origin），兩邊也都已發佈 Release。

- 兩個 `v0.1.20` 都已有人安裝，保持原狀，不刪除、不移動。
- 本機同時設了上游與下游 remote 時，下游 remote 設 `git config remote.<downstream>.tagOpt --no-tags`，避免 fetch 時把同名 tag 混在一起；本機的 tag 一律以上游為準。
- 回報問題時，`v0.1.20` 需註明是上游或下游的 Release。

## Workflows

- `ci`：在 pull request 與 `main` push 時執行 lint、typecheck、build script 測試、Nuxt build 與 connector 測試。
- `release-installers`：在 `v*` tag push 時建置安裝包並發佈 release。
- `deploy-cloudflare`：只在上游 repo 部署 demo 站台。

## 本機指令

- `pnpm run build:app`：只建置 Nuxt 應用程式。
- `pnpm run connector:test`：執行 connector 測試。
- `pnpm run lint`：以 Vite+（`vp lint`，oxlint）檢查 JS / TS / Vue，warning 也視為失敗。
- `pnpm run typecheck`：Nuxt / Vue 型別檢查。
- `pnpm run test:scripts`：build script 的單元測試（含 Go 與 JS public origin 清單一致性檢查）。
- `pnpm run ci`：執行本機 CI 等級驗證。
- `pnpm run build`：建置應用程式並產出本機可用安裝包。

## 簽章狀態

目前 release 流程會產出可安裝的安裝包，但尚未包含正式簽章流程。

- Windows：未做 Authenticode 簽章時，可能出現 SmartScreen 警告。
- macOS：未做 Developer ID 簽章與 notarization 時，可能出現 Gatekeeper 警告。
- Ubuntu：`.deb` 可安裝，但若未搭配額外套件來源簽署，不等同於 repository trust chain。

若後續要對外正式發佈，建議新增：

1. Windows code signing。
2. macOS Developer ID signing。
3. macOS notarization。
