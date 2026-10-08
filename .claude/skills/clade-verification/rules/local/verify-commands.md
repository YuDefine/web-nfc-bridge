---
description: 本 consumer 的 verify gate chain 指令清單——iterate-until-green 迴圈跑這些指令判定 PASS/FAIL
paths: ['**/*.ts', '**/*.vue', '**/*.tsx', 'package.json', 'packages/*/package.json', 'pnpm-lock.yaml', 'pnpm-workspace.yaml']
---
<!-- Clade native rule; source: .clade/rules/verify-commands.md; edit canonical source -->
<!-- clade-targets: claude,codex -->

# Verify Commands

本檔定義此 consumer 的 gate chain 指令。被 [[verify-gate-chain]]（clade 散播的核心規約）引用。

## Gate Chain

每條指令的 exit code 是判定依據：exit 0 = PASS，non-zero = FAIL。

```
L0: vp check
L1: pnpm typecheck
L2: pnpm test --run
L3: curl -sf http://localhost:__PORT__/api/health
```

## 填寫說明

<!-- 新來源建立於 .clade/rules/verify-commands.md，frontmatter 後宣告經審閱的 clade-targets audience；init-consumer 會產生三端 seed。既有 legacy local rule 先走明確 adoption，保留原內容與所有權。修改以下項目：

1. L0：若 consumer 不用 vp，改為該 consumer 的 lint+fmt 指令（如 `pnpm lint && pnpm fmt:check`）
2. L1：typecheck 指令。大部分 consumer 用 `pnpm typecheck`（Nuxt: `nuxt typecheck`）
3. L2：test 指令。`pnpm test --run`（Vitest）或 `pnpm test`（Jest）。若無 test suite 標註 `L2: (skip — no test suite)`
4. L3：dev server health endpoint。改 __PORT__ 為實際 port（見 dev-port-allocation rule）
   若 consumer 無 health endpoint，標註 `L3: (skip — no health endpoint)`

注意：
- L0–L2 是 MUST（全 PASS 才算 done）
- L3 是 SHOULD（dev server 未起時 skip 不算 FAIL）
- 指令 MUST 可在 consumer repo root 直接執行（不用 cd）
-->
