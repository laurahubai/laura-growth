import { execFileSync, spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));

function carregarEnv() {
  const caminho = path.resolve(dirAtual, "../../../.env");
  const env = {};
  for (const [k, v] of Object.entries(process.env)) {
    if (v !== undefined) env[k] = v;
  }
  if (!fs.existsSync(caminho)) return env;
  const conteudo = fs.readFileSync(caminho, "utf8");
  for (const linhaBruta of conteudo.split(/\r?\n/)) {
    const linha = linhaBruta.trim();
    if (!linha || linha.startsWith("#")) continue;
    const idx = linha.indexOf("=");
    if (idx <= 0) continue;
    const chave = linha.slice(0, idx).trim();
    let valor = linha.slice(idx + 1).trim();
    if (
      (valor.startsWith('"') && valor.endsWith('"')) ||
      (valor.startsWith("'") && valor.endsWith("'"))
    ) {
      valor = valor.slice(1, -1);
    }
    if (env[chave] === undefined) env[chave] = valor;
  }
  return env;
}

function exigirBancoE2E(urlBruta) {
  const valor = (urlBruta ?? "").trim();
  if (!valor) {
    console.error("LG_E2E_BANCO_URL não definida; configure um banco exclusivo terminado em _teste ou _e2e.");
    process.exit(1);
  }
  let parsed;
  try {
    parsed = new URL(valor);
  } catch {
    console.error("LG_E2E_BANCO_URL inválida.");
    process.exit(1);
  }
  const nomeBanco = parsed.pathname.replace(/^\/+/, "");
  if (!nomeBanco.endsWith("_teste") && !nomeBanco.endsWith("_e2e")) {
    console.error(`Recusado: o banco E2E (${nomeBanco || "vazio"}) precisa terminar com _teste ou _e2e.`);
    process.exit(1);
  }
  return valor;
}

const backendDir = path.resolve(dirAtual, "../../../backend");
const env = carregarEnv();
const bancoE2E = exigirBancoE2E(env.LG_E2E_BANCO_URL);
env.LG_BANCO_URL = bancoE2E;

execFileSync("go", ["run", ".", "migrar"], { cwd: backendDir, env, stdio: "inherit" });
execFileSync("go", ["run", ".", "semear"], { cwd: backendDir, env, stdio: "inherit" });

const filho = spawn("go", ["run", ".", "servir"], {
  cwd: backendDir,
  env,
  stdio: "inherit",
});

function encerrar(sinal) {
  if (!filho.killed) {
    filho.kill(sinal);
  }
}

process.on("SIGTERM", () => encerrar("SIGTERM"));
process.on("SIGINT", () => encerrar("SIGINT"));

filho.on("exit", (code) => {
  process.exit(code ?? 0);
});
