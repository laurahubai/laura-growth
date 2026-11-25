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

const backendDir = path.resolve(dirAtual, "../../../backend");
const env = carregarEnv();

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
