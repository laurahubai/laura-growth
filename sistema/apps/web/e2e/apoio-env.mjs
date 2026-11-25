import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));

// Precedência consistente em todo o fluxo E2E: variáveis de process.env sobrescrevem sistema/.env.
export function carregarEnvEfetivo() {
  const caminho = path.resolve(dirAtual, "../../../.env");
  const env = {};
  if (fs.existsSync(caminho)) {
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
  }
  for (const [k, v] of Object.entries(process.env)) {
    if (v !== undefined) env[k] = v;
  }
  return env;
}

export function validarBancoE2E(urlBruta) {
  const valor = (urlBruta ?? "").trim();
  if (!valor) {
    throw new Error(
      "LG_E2E_BANCO_URL não definida; configure um banco exclusivo terminado em _teste ou _e2e.",
    );
  }
  let parsed;
  try {
    parsed = new URL(valor);
  } catch {
    throw new Error("LG_E2E_BANCO_URL inválida.");
  }
  const nomeBanco = parsed.pathname.replace(/^\/+/, "");
  if (!nomeBanco.endsWith("_teste") && !nomeBanco.endsWith("_e2e")) {
    throw new Error(
      `Recusado: o banco E2E (${nomeBanco || "vazio"}) precisa terminar com _teste ou _e2e.`,
    );
  }
  return valor;
}

export function validarEnderecoApiE2E(enderecoBruto) {
  const endereco = (enderecoBruto ?? "").trim() || "127.0.0.1:18090";
  let parsed;
  try {
    parsed = new URL(`http://${endereco}`);
  } catch {
    throw new Error(`LG_E2E_API_ENDERECO inválido: ${endereco}`);
  }
  if (!parsed.port || parsed.port === "8090") {
    throw new Error(`Recusado: a porta da API E2E (${parsed.port || "vazia"}) não pode ser 8090.`);
  }
  return endereco;
}

export function validarPortaWebE2E(portaBruta) {
  const porta = (portaBruta ?? "").trim() || "13200";
  if (!/^\d+$/.test(porta) || Number(porta) <= 0 || porta === "3200") {
    throw new Error(`Recusado: a porta web E2E (${porta}) é inválida ou usa a porta 3200.`);
  }
  return porta;
}
