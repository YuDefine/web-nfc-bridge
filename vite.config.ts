import { defineConfig } from "vite-plus";

// Lint only (`pnpm lint` → `vp lint`). Nuxt does not read this file; its Vite
// config lives in nuxt.config.ts.
export default defineConfig({
  lint: {
    categories: {
      correctness: "error",
      suspicious: "warn",
      perf: "warn",
    },
    rules: {
      eqeqeq: ["error", "always"],
      // Tap/write retries poll sequentially on purpose.
      "no-await-in-loop": "off",
      // Composables keep their helpers next to the state they serve.
      "unicorn/consistent-function-scoping": "off",
    },
    ignorePatterns: [
      ".nuxt/**",
      ".output/**",
      "node_modules/**",
      "public/**",
      ".github/skills/**",
      ".github/prompts/**",
    ],
  },
});
