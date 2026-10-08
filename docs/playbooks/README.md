# web-nfc-bridge gate playbooks（對話式 SOP）

HANDOFF `## User-gate board` 是狀態 SoT。本目錄是 **AI session 逐步執行** 的腳本，不是丟給人的 URL 手冊。

**Iron Law**：session 做得到的動作 NEVER 交給人。停手條件只有「該動作自己的失敗輸出」＋「人類專屬能力」（生物辨識 / CLI 做不到的外部授權點擊 / 不可逆商業拍板）。

## Task todos（每個新 session 開工必做）

每個 gate **一支** Todo（`TodoWrite` merge true）。`content` 必須含 playbook 路徑。人看到的 Task 清單就是這五格：

| todo id | playbook |
| --- | --- |
| `gate-01-dev-database` | [01-dev-database.md](./01-dev-database.md) |
| `gate-02-ssh-config` | [02-ssh-config.md](./02-ssh-config.md) |
| `gate-03-oauth` | [03-google-oauth.md](./03-google-oauth.md) |
| `gate-04-deploy-prod-db` | [04-deploy-prod-db.md](./04-deploy-prod-db.md) |
| `gate-05-post-verify` | [05-post-gate-verify.md](./05-post-gate-verify.md) |

規則：當步 `in_progress`、後面 `pending`、該 SOP success probe 通過才 `completed`。一次只推進一格（03 可與 01/02 並行）。逐字內容見 [GATE-TODOS.md](./GATE-TODOS.md)。

## 狀態機（看板）

| 狀態 | 意思 | 誰改 |
| --- | --- | --- |
| `ready-for-user` | SOP 已落地、下一步尚未開始（歷史名；執行者仍是 agent） | playbook 剛寫好時 |
| `waiting` | agent 正在跑、或等外部生效（MagicDNS / OAuth API 傳播） | 動手的那一方 |
| `user-done-unverified` | 外部授權剛做完、success probe 還沒跑 | 僅當真的走過人類專屬授權 |
| `verified` | success probe 通過（可分岔） | agent |
| `blocked-unexpected` | 該次指令失敗且失敗分支走完仍卡 | agent（附失敗原文） |

## 每支 SOP 的共同形狀

1. 開頭列 **todo id / 路徑 / Ready checklist**。Ready 沒綠 → 不要開這支，回到上一格。
2. **Numbered turns**：AI 跑什麼、說什麼、等什麼觀測。失敗 = 下一 turn，不是「請你去點」。
3. 每 turn 結束：寫看板狀態 + 往 [PROGRESS.md](./PROGRESS.md) append 一行（日期 · gate id · 指令 · exit · 訊號 · 結果）。

## Browser 分流

開 Tailscale login / Google Console / OAuth callback **之前**先判目標是**公網 HTTPS**還是**本機服務**。01 Turn 4、03 Turn 3、05 若要回 Console，都指回本節。

**NEVER** 派沒有 browser 工具的 subagent 假裝操控 Chromium。

### 開頁架構

| 目標 | 怎麼開 | port forward |
| --- | --- | --- |
| Tailscale `https://login.tailscale.com/a/…`、Google Console `https://console.cloud.google.com/apis/credentials` | **公網 HTTPS**。走 `agent-browser`（仍是 agent 自己開；**NEVER** 第一手叫人） | **不要**。不要為 HTTPS 轉 443 |
| OAuth 完成後瀏覽器打 `http://127.0.0.1:3030/auth/google` | **本機服務**。Mac 的 `127.0.0.1:3030` 必須打到這台遠端的 3030 | **要**：Mac 上 `ssh -L 3030:127.0.0.1:3030`。dev server 還沒聽也先備 forward |

帳號選擇／2FA／passkey 才是人類專屬：把**活的** URL 寫進 PROGRESS。過期 URL **不是**行動項。

## 誰更新哪一欄

- **agent**：跑每一 turn、改看板、append PROGRESS。
- **人**：只在某 turn 的失敗輸出證明是人類專屬能力時介入；agent 必須貼出指令 + exit + 原文。
- **NEVER** 把過期 Tailscale login URL 當行動項。現場重取。**NEVER** `tailscale logout`。**NEVER** 動 {{NEVER_TOUCH_PEER}}。

## 順序

01 → 02 →（03 可並行）→ 04 套用預設（CI build-only）→ 05。

## Starter contract

`/project-bootstrap` / starter onboarding **MUST** 在任何「請點」之前 mint 本目錄這包。契約 SoT 在 clade `vendor/snippets/new-consumer-gate-playbooks/`。
