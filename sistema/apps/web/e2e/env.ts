import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));

export function carregarEnvSistema(): Record<string, string> {
  const caminho = path.resolve(dirAtual, "../../../.env");
  const env: Record<string, string> = {};
  if (!fs.existsSync(caminho)) {
    return env;
  }
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
    env[chave] = valor;
  }
  return env;
}
