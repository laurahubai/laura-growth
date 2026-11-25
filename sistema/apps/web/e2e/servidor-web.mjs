import { execFileSync, spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(dirAtual, "..");
const nextBin = path.resolve(webDir, "node_modules/.bin/next");
const porta = process.env.LG_WEB_PORTA || "13200";

function compilarNext() {
  try {
    execFileSync(nextBin, ["build"], { cwd: webDir, env: process.env, stdio: "inherit" });
  } catch {
    // Next.js pode encerrar a primeira execução em sessões de agente para exibir o aviso de versão;
    // tentar novamente continua normalmente conforme orientado pelo próprio CLI.
    execFileSync(nextBin, ["build"], { cwd: webDir, env: process.env, stdio: "inherit" });
  }
}

compilarNext();

const filho = spawn(nextBin, ["start", "-H", "127.0.0.1", "-p", porta], {
  cwd: webDir,
  env: process.env,
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
