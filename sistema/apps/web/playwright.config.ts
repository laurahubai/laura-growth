import { defineConfig, devices } from "@playwright/test";
import { carregarEnvSistema, validarBancoE2E } from "./e2e/env";

const envSistema = carregarEnvSistema();
const bancoE2E = validarBancoE2E(process.env.LG_E2E_BANCO_URL || envSistema.LG_E2E_BANCO_URL);
const apiEndereco = process.env.LG_API_ENDERECO || envSistema.LG_API_ENDERECO || "127.0.0.1:18090";
const apiUrl = `http://${apiEndereco}`;
const webPorta = 13200;
const webUrl = `http://127.0.0.1:${webPorta}`;

export default defineConfig({
  testDir: "./e2e",
  timeout: 90_000,
  expect: {
    timeout: 15_000,
  },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: "list",
  use: {
    baseURL: webUrl,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: [
    {
      command: "node e2e/servidor-api.mjs",
      url: `${apiUrl}/api/saude`,
      timeout: 120_000,
      reuseExistingServer: false,
      env: {
        ...envSistema,
        LG_E2E_BANCO_URL: bancoE2E,
        LG_API_ENDERECO: apiEndereco,
      },
    },
    {
      command: "node e2e/servidor-web.mjs",
      url: `${webUrl}/entrar`,
      timeout: 180_000,
      reuseExistingServer: false,
      env: {
        LG_API_INTERNA: apiUrl,
        LG_WEB_PORTA: String(webPorta),
      },
    },
  ],
});
