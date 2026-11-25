import { defineConfig, devices } from "@playwright/test";
import {
  carregarEnvEfetivo,
  validarBancoE2E,
  validarEnderecoApiE2E,
  validarPortaWebE2E,
} from "./e2e/env";

const envEfetivo = carregarEnvEfetivo();
const bancoE2E = validarBancoE2E(envEfetivo.LG_E2E_BANCO_URL);
const apiEndereco = validarEnderecoApiE2E(envEfetivo.LG_E2E_API_ENDERECO);
const webPorta = validarPortaWebE2E(envEfetivo.LG_E2E_WEB_PORTA);
const apiUrl = `http://${apiEndereco}`;
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
        ...envEfetivo,
        LG_E2E_BANCO_URL: bancoE2E,
        LG_E2E_API_ENDERECO: apiEndereco,
      },
    },
    {
      command: "node e2e/servidor-web.mjs",
      url: `${webUrl}/entrar`,
      timeout: 180_000,
      reuseExistingServer: false,
      env: {
        ...envEfetivo,
        LG_E2E_API_ENDERECO: apiEndereco,
        LG_E2E_WEB_PORTA: webPorta,
        LG_NEXT_DIST_DIR: ".next-e2e",
      },
    },
  ],
});
