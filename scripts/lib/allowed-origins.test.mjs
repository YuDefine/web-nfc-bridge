import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import {
  parseExtraAllowedOrigins,
  publicAllowedOrigins,
  resolveAllowedOrigins,
} from "./allowed-origins.mjs";

test("public origins match the Go connector defaults", () => {
  const mainGo = readFileSync(
    fileURLToPath(
      new URL("../../connector/cmd/nfc-connector/main.go", import.meta.url),
    ),
    "utf8",
  );
  const match = mainGo.match(/const publicAllowedOrigins = "([^"]*)"/);
  assert.ok(match, "publicAllowedOrigins const not found in main.go");
  assert.deepEqual(match[1].split(","), publicAllowedOrigins);
});

test("defaults contain only public origins", () => {
  assert.deepEqual(resolveAllowedOrigins(undefined), {
    extra: [],
    all: publicAllowedOrigins,
  });
  assert.deepEqual(resolveAllowedOrigins(""), {
    extra: [],
    all: publicAllowedOrigins,
  });
});

test("extra origins are appended after public origins", () => {
  const { extra, all } = resolveAllowedOrigins(
    "https://downstream.example, https://other.example:8443",
  );
  assert.deepEqual(extra, [
    "https://downstream.example",
    "https://other.example:8443",
  ]);
  assert.deepEqual(all, [...publicAllowedOrigins, ...extra]);
});

test("extra origins already in the public list are not duplicated", () => {
  const { extra, all } = resolveAllowedOrigins(
    "https://nfc.yudefine.com.tw https://downstream.example https://downstream.example/",
  );
  assert.deepEqual(extra, ["https://downstream.example"]);
  assert.equal(all.length, publicAllowedOrigins.length + 1);
});

test("IP, port and punycode hostnames are accepted", () => {
  assert.deepEqual(
    parseExtraAllowedOrigins(
      "http://192.168.1.10:8080 https://xn--r8jz45g.jp https://sub-domain.example.com",
    ),
    [
      "http://192.168.1.10:8080",
      "https://xn--r8jz45g.jp",
      "https://sub-domain.example.com",
    ],
  );
});

test("trailing-dot hostnames are accepted as exact origins", () => {
  assert.deepEqual(parseExtraAllowedOrigins("https://downstream.example."), [
    "https://downstream.example.",
  ]);
});

for (const [value, reason] of [
  ["downstream.example", /not a URL/],
  ["ftp://downstream.example", /only http and https/],
  ["https://downstream.example/path", /expected an origin/],
  ["https://user@downstream.example", /expected an origin/],
  ["https://Downstream.example", /expected an origin/],
  ["https://*.downstream.example", /wildcards/],
  ["http://localhost:*", /wildcards/],
  ['https://a.example"/><key>x', /hostname may only contain/],
  ["https://a&b.example", /hostname may only contain/],
  ['https://a"b.example', /hostname may only contain/],
  ["https://a$b.example", /hostname may only contain/],
  ["https://a`b.example", /hostname may only contain/],
  ["https://a_b.example", /hostname may only contain/],
  ["https://-a.example", /hostname may only contain/],
  ["https://a..example", /hostname may only contain/],
  ["https://[::1]", /hostname may only contain/],
]) {
  test(`rejects ${value}`, () => {
    assert.throws(() => parseExtraAllowedOrigins(value), reason);
  });
}
