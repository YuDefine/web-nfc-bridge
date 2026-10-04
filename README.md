# web-nfc-bridge

[繁體中文](README.zh-TW.md) | English

A working example of reading and writing NFC cards from a web browser — using a localhost connector architecture instead of the experimental Web NFC API.

```
Browser (HTTPS)  ←→  Localhost Connector (Go)  ←→  PC/SC  ←→  NFC Reader
```

## Why

The [Web NFC API](https://developer.mozilla.org/en-US/docs/Web/API/Web_NFC_API) is Chrome-only, Android-only, and still experimental. This project demonstrates an alternative approach: a lightweight Go service running on `127.0.0.1` that bridges any web app to a USB NFC reader via the standard PC/SC interface.

This architecture works on **macOS, Windows (x64/ARM64), and Linux** — anywhere a browser and a USB reader can coexist.

## Features

- **Read** — Extract UID and JSON-encoded NDEF data from NFC Type 2 cards
- **Write** — Write structured JSON payloads as NDEF records with read-back verification
- **Real-time events** — WebSocket stream for card presence, reader status, and operation results
- **Security by design** — HMAC-signed short-lived tokens, origin allowlist, no arbitrary APDU passthrough
- **Cross-platform installers** — macOS (Apple Silicon), Windows x64/ARM64 and Linux x64 packages built on PC/SC, with a Direct IOCTL fallback driver on Windows. Only macOS is validated so far; see [Platform Support](#platform-support)

## Architecture

```
┌─────────────────────────────────────────────┐
│  Browser                                    │
│  Nuxt 4 + Nuxt UI (deployed to Cloudflare)  │
└──────────────┬──────────────────────────────┘
               │ HTTP / WebSocket
               ▼
┌──────────────────────────────────────┐
│  Localhost Connector (Go)            │
│  http://127.0.0.1:42619             │
│                                      │
│  ┌────────────┐  ┌────────────────┐  │
│  │ PCSC Driver │  │ Direct Driver  │  │
│  │ (all OS)    │  │ (Win ARM64)    │  │
│  └──────┬─────┘  └───────┬────────┘  │
└─────────┼────────────────┼───────────┘
          ▼                ▼
   ┌─────────────────────────────┐
   │  NFC Reader (ACR1252U-M1)   │
   │  ISO-14443 Type 2 cards     │
   └─────────────────────────────┘
```

## Quick Start

### Prerequisites

- [Node.js](https://nodejs.org/) 22+ and [pnpm](https://pnpm.io/) 10+
- [Go](https://go.dev/) 1.26+
- A PC/SC-compatible NFC reader (tested with ACS ACR1252U-M1)
- PC/SC service: built-in on macOS/Windows; install `pcscd` on Linux

### Run in Development

```bash
# Install frontend dependencies
pnpm install

# Terminal 1 — Start the web app
pnpm dev

# Terminal 2 — Start the connector
pnpm connector:dev
```

Open http://localhost:3000 — the web UI will automatically detect the connector and list available readers.

### Build

```bash
# Build everything (web app + connector + installers)
pnpm build

# Or build individually
pnpm build:app              # Nuxt web app (Node.js)
pnpm build:cf               # Nuxt web app (Cloudflare Workers)
pnpm connector:build        # Go connector binary
```

### Build Installers

```bash
pnpm installers:build:macos          # macOS .pkg
pnpm installers:build:windows-x64    # Windows x64 .msi
pnpm installers:build:windows-arm64  # Windows ARM64 .msi
pnpm installers:build:linux-x64      # Linux .deb
```

## Connector HTTP API

Base URL: `http://127.0.0.1:42619`

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Connector health check |
| `GET` | `/readers` | List available NFC readers |
| `POST` | `/session/connect` | Start a reader session |
| `POST` | `/card/read` | Read card UID + NDEF data |
| `POST` | `/card/write` | Write NDEF payload to card |
| `WS` | `/events` | Real-time event stream |

All endpoints (except `/health`) require an `X-Bridge-Token` header with a signed ticket issued by the web app's server route `POST /api/connector-ticket`.

See [docs/connector-contract.md](docs/connector-contract.md) for full API details.

## Write Payload Example

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

Constraints:
- `content` must be a flat JSON object (no nesting), max 8 fields
- Values: string, number, boolean, or null only
- Max encoded NDEF size: 256 bytes
- Sensitive fields (`password`, `secret`, `email`, etc.) are rejected

## Security Model

1. **No raw APDU** — The connector only accepts pre-defined operations, never arbitrary card commands
2. **Short-lived tokens** — HMAC-signed tickets expire in 60 seconds, scoped to `read`, `write`, `events`, or `all`
3. **Origin allowlist** — Only whitelisted origins can communicate with the connector
4. **Payload validation** — Write payloads are validated for structure, size, and forbidden fields before any card I/O
5. **Read-back verification** — After writing, the connector reads the card back to confirm data integrity

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | [Nuxt 4](https://nuxt.com/) + [Nuxt UI](https://ui.nuxt.com/) + Tailwind CSS |
| Connector | [Go](https://go.dev/) + [gorilla/websocket](https://github.com/gorilla/websocket) |
| Smart Card | [ebfe/scard](https://github.com/ebfe/scard) (PC/SC binding) |
| Deployment | [Cloudflare Workers](https://workers.cloudflare.com/) (web app only) |

## Platform Support

Installers are published for every platform below, but only platforms with a recorded hardware validation are marked Validated. [docs/platform-support.md](docs/platform-support.md) is the source of truth; this table must not claim more than it does.

| Platform | Architecture | Driver | Status |
|----------|-------------|--------|--------|
| macOS | Apple Silicon | PC/SC | Validated (read/write) |
| Windows | x64 | PC/SC, Direct IOCTL fallback | Installer available, validation in progress |
| Windows | ARM64 | PC/SC, Direct IOCTL fallback | Installer available, validation in progress (high risk) |
| Linux | x64 | PC/SC | Installer available, validation in progress |

macOS Intel has no installer and is not supported.

### Test Environments

| Platform | Device | OS Version | Validation |
|----------|--------|------------|------------|
| macOS | Mac mini M4 | macOS Tahoe 26.3.1 | Validated |
| Windows x64 | Intel NUC | Windows 11 25H2 | In progress |
| Windows ARM64 | ASUS Zenbook A14 (Snapdragon X - X126100) | Windows 11 ARM 25H2 | In progress |
| Linux x64 | — | Ubuntu Server 24.04 | In progress |

- **Reader**: ACS ACR1252U-M1
- **Card**: NTAG 216 (NFC Type 2)

## Project Structure

```
├── pages/                  # Nuxt pages (web UI)
├── composables/            # Vue composables (connector communication)
├── server/api/             # Nuxt server routes (ticket issuance)
├── connector/
│   ├── cmd/nfc-connector/  # Go entrypoint
│   └── internal/
│       ├── api/            # HTTP + WebSocket server
│       ├── auth/           # Token verification
│       └── bridge/         # Driver layer (PCSC, Direct IOCTL, NDEF codec)
├── scripts/                # Installer build scripts
└── docs/                   # Architecture & API documentation
```

## Configuration

Environment variables for the connector:

| Variable | Default | Description |
|----------|---------|-------------|
| `NFC_CONNECTOR_ADDR` | `127.0.0.1:42619` | Connector listen address |
| `NFC_CONNECTOR_SHARED_SECRET` | `development-shared-secret` | HMAC signing key |
| `NFC_CONNECTOR_ALLOWED_ORIGINS` | localhost / 127.0.0.1 (any port) + the public demo sites, plus any origins added at build time | Comma-separated origin allowlist; replaces the default when set |

## Documentation

- [Architecture](docs/architecture.md) — System layers and design decisions
- [Connector Contract](docs/connector-contract.md) — Full HTTP/WebSocket API specification
- [Connector Operations](docs/connector-operations.md) — Operational details
- [Platform Support](docs/platform-support.md) — Platform validation matrix

## Credits

Built by [Yudefine - 域定資訊工作室](https://yudefine.com.tw)

## License

[MIT](LICENSE)
