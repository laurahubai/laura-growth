import { execFileSync, spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { carregarEnvEfetivo, validarEnderecoApiE2E, validarPortaWebE2E } from "./apoio-env.mjs";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(dirAtual, "..");
const nextBin = path.resolve(webDir, "node_modules/.bin/next");

const env = carregarEnvEfetivo();
let apiEndereco;
let porta;
try {
  apiEndereco = validarEnderecoApiE2E(env.LG_E2E_API_ENDERECO);
  porta = validarPortaWebE2E(env.LG_E2E_WEB_PORTA);
} catch (erro) {
  console.error(erro instanceof Error ? erro.message : String(erro));
  process.exit(1);
}

env.LG_API_INTERNA = `http://${apiEndereco}`;
env.LG_NEXT_DIST_DIR = ".next-e2e";

function compilarNext() {
  try {
    execFileSync(nextBin, ["build"], { cwd: webDir, env, stdio: "inherit" });
  } catch {
    // Next.js pode encerrar a primeira execução em sessões de agente para exibir o aviso de versão;
    // tentar novamente continua normalmente conforme orientado pelo próprio CLI.
    execFileSync(nextBin, ["build"], { cwd: webDir, env, stdio: "inherit" });
  }
}

compilarNext();

const filho = spawn(nextBin, ["start", "-H", "127.0.0.1", "-p", porta], {
  cwd: webDir,
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
