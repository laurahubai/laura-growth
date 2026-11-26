import { execFileSync, spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { carregarEnvEfetivo, validarBancoE2E, validarEnderecoApiE2E } from "./apoio-env.mjs";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));
const backendDir = path.resolve(dirAtual, "../../../backend");
const env = carregarEnvEfetivo();

let bancoE2E;
let apiEndereco;
try {
  bancoE2E = validarBancoE2E(env.LG_E2E_BANCO_URL);
  apiEndereco = validarEnderecoApiE2E(env.LG_E2E_API_ENDERECO);
} catch (erro) {
  console.error(erro instanceof Error ? erro.message : String(erro));
  process.exit(1);
}

env.LG_BANCO_URL = bancoE2E;
env.LG_API_ENDERECO = apiEndereco;

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
