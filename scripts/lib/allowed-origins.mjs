// Origins every build allows. Must match publicAllowedOrigins in
// connector/cmd/nfc-connector/main.go (checked by allowed-origins.test.mjs).
// Downstream deployments add their own origins at build time instead of
// editing this list; see docs/release-process.md.
export const publicAllowedOrigins = [
  "http://localhost:*",
  "https://localhost:*",
  "http://127.0.0.1:*",
  "https://127.0.0.1:*",
  "https://web-nfc-bridge.abcd854884.workers.dev",
  "https://web-nfc-bridge.abcd854884.workers.dev.",
  "https://nfc.yudefine.com.tw",
  "https://nfc.yudefine.com.tw.",
];

export const extraAllowedOriginsEnv = "NFC_CONNECTOR_EXTRA_ALLOWED_ORIGINS";

// LDH hostname (letters, digits, hyphen; dot-separated, optional trailing dot).
// URL parsing alone accepts characters such as & " $ ` in hosts, which would
// break the plist XML or the shell environment file.
const hostnamePattern = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*\.?$/;

// Parses a comma- or whitespace-separated list of exact origins
// (scheme://host[:port], no path, no wildcard). The values end up in a plist,
// a shell environment file and Go -ldflags, so anything else is rejected.
export function parseExtraAllowedOrigins(raw) {
  const origins = [];
  for (const entry of (raw || "").split(/[\s,]+/)) {
    if (!entry) {
      continue;
    }

    if (entry.includes("*")) {
      throw new Error(
        `Invalid extra allowed origin '${entry}': wildcards are not supported`,
      );
    }

    let url;
    try {
      url = new URL(entry);
    } catch {
      throw new Error(`Invalid extra allowed origin '${entry}': not a URL`);
    }

    if (url.protocol !== "https:" && url.protocol !== "http:") {
      throw new Error(
        `Invalid extra allowed origin '${entry}': only http and https are allowed`,
      );
    }
    if (!hostnamePattern.test(url.hostname)) {
      throw new Error(
        `Invalid extra allowed origin '${entry}': hostname may only contain letters, digits, '-' and '.'`,
      );
    }
    if (url.origin !== entry.replace(/\/$/, "")) {
      throw new Error(
        `Invalid extra allowed origin '${entry}': expected an origin like ${url.origin}`,
      );
    }

    if (!origins.includes(url.origin)) {
      origins.push(url.origin);
    }
  }
  return origins;
}

// Returns the extra origins (not already public) and the full list written to
// installer environment files.
export function resolveAllowedOrigins(extraRaw) {
  const extra = parseExtraAllowedOrigins(extraRaw).filter(
    (origin) => !publicAllowedOrigins.includes(origin),
  );
  return { extra, all: [...publicAllowedOrigins, ...extra] };
}
