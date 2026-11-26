import { execFileSync } from "node:child_process";
import crypto from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test, type Page } from "@playwright/test";
import { carregarEnvEfetivo, validarBancoE2E } from "./env";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));
const backendDir = path.resolve(dirAtual, "../../../backend");

const envEfetivo = carregarEnvEfetivo();
const bancoE2E = validarBancoE2E(envEfetivo.LG_E2E_BANCO_URL);

function rodarApoio(args: string[], extras: Record<string, string> = {}) {
  execFileSync("go", ["run", "./e2e_apoio.go", ...args], {
    cwd: backendDir,
    env: {
      ...process.env,
      ...envEfetivo,
      LG_E2E_BANCO_URL: bancoE2E,
      ...extras,
    },
    stdio: "inherit",
  });
}

function senhaAleatoria(): string {
  return crypto.randomBytes(18).toString("base64url");
}

async function entrar(page: Page, email: string, senha: string) {
  await page.goto("/entrar");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel("Senha").fill(senha);
  await page.getByRole("button", { name: "Entrar na plataforma" }).click();
  await expect(page).toHaveURL(/\/inicio$/);
}

test("C2, C3, C4, C5, C7: DNA da marca — O que evitar no topo, novos campos, origens, versões, conflito 409 e alçadas", async ({
  browser,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  const orgNome = `Org Kit DNA E2E ${tag}`;
  const donoEmail = `dono-kit-${tag}@exemplo.test`;
  const donoSenha = senhaAleatoria();
  const leituraEmail = `leit-kit-${tag}@exemplo.test`;
  const leituraSenha = senhaAleatoria();
  const semAcessoEmail = `sem-kit-${tag}@exemplo.test`;
  const semAcessoSenha = senhaAleatoria();

  const apoioEnv = {
    LG_E2E_ORG_NOME: orgNome,
    LG_E2E_ORG_DONO_EMAIL: donoEmail,
    LG_E2E_ORG_DONO_SENHA: donoSenha,
    LG_E2E_OPERACAO_EMAIL: leituraEmail,
    LG_E2E_OPERACAO_SENHA: leituraSenha,
    LG_E2E_SEM_ACESSO_EMAIL: semAcessoEmail,
    LG_E2E_SEM_ACESSO_SENHA: semAcessoSenha,
  };
  rodarApoio(["preparar"], apoioEnv);

  try {
    const ctxDono = await browser.newContext();
    const pageDono = await ctxDono.newPage();
    await entrar(pageDono, donoEmail, donoSenha);

    await pageDono.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Kit de marca" }).click();
    await expect(pageDono).toHaveURL(/\/kit$/);

    // 1) Primeiro salvamento (versão 1): preenche "O que nunca dizer" e "Restrições do setor" e valida bloco no topo (C2).
    await pageDono.getByLabel("O que nunca dizer").fill("Promessas milagrosas, garantia total");
    await pageDono.getByLabel("Restrições do setor").fill("Proibido divulgar resultados sem respaldo técnico.");
    await pageDono.getByRole("button", { name: "Salvar kit de marca" }).click();
    await expect(pageDono.getByRole("status")).toContainText("Kit salvo.");
    await expect(pageDono.getByTestId("versao-atual-kit")).toHaveText("Versão 1");

    const blocoEvitar = pageDono.getByTestId("bloco-o-que-evitar");
    await expect(blocoEvitar).toBeVisible();
    await expect(blocoEvitar).toContainText("O que evitar");
    await expect(blocoEvitar).toContainText("Promessas milagrosas, garantia total");
    await expect(blocoEvitar).toContainText("Proibido divulgar resultados sem respaldo técnico.");

    const boxEvitar = await blocoEvitar.boundingBox();
    const boxForm = await pageDono.locator("form").boundingBox();
    expect(boxEvitar).not.toBeNull();
    expect(boxForm).not.toBeNull();
    expect(boxEvitar!.y).toBeLessThan(boxForm!.y);

    // 2) Preenche novos campos de Negócio e Estratégia -> gera versão 2 (C4).
    await pageDono.getByRole("button", { name: "Adicionar produto ou serviço" }).click();
    await pageDono.getByLabel("Produto ou serviço 1").fill("Consultoria de Crescimento");
    await expect(pageDono.getByTestId("aviso-produto-prioritario")).toBeVisible();
    await pageDono.getByLabel("Prioritário 1").check();
    await expect(pageDono.getByTestId("aviso-produto-prioritario")).toHaveCount(0);

    await pageDono.getByLabel("Diferencial").fill("Atendimento próximo com equipe sênior");
    await pageDono.getByLabel("Modelo de negócio").selectOption("outro");
    await pageDono.getByLabel("Qual modelo de negócio?").fill("Assinatura consultiva");
    await pageDono.getByLabel("Regiões e idiomas").fill("Brasil (português)");
    await pageDono.getByLabel("Como os clientes compram").fill("Indicação e diagnóstico gratuito");

    await pageDono.getByLabel("Maior desafio").fill("Gerar demanda qualificada todos os meses");
    await pageDono.getByLabel("Objetivo dos próximos 90 dias").fill("Aumentar agendamentos em 30%");
    const campoCanais = pageDono.getByTestId("campo-canais");
    await campoCanais.getByLabel("Instagram").check();
    await campoCanais.getByLabel("LinkedIn").check();
    await campoCanais.getByLabel("Outro canal").fill("Podcast mensal");

    await pageDono.getByRole("button", { name: "Salvar kit de marca" }).click();
    await expect(pageDono.getByRole("status")).toContainText("Kit salvo.");
    await expect(pageDono.getByTestId("versao-atual-kit")).toHaveText("Versão 2");
    await expect(pageDono.getByTestId("campo-produtos")).toContainText("Confirmado por você");
    await expect(pageDono.getByTestId("campo-canais")).toContainText("Confirmado por você");

    // 3) Abre Versões, confere versão 2 na lista e abre versão 2 e versão 1 somente leitura (C4).
    await pageDono.getByRole("link", { name: "Versões" }).click();
    await expect(pageDono).toHaveURL(/\/kit\/versoes$/);

    const linhaV2 = pageDono.getByTestId("versao-2");
    await expect(linhaV2).toBeVisible();
    await expect(linhaV2).toContainText("Versão 2");
    await expect(linhaV2).toContainText("Dono Temporário E2E");
    await expect(linhaV2).toContainText("Produtos e serviços");
    await expect(linhaV2).toContainText("Diferencial");
    await expect(linhaV2).toContainText("Canais");

    await linhaV2.getByRole("link", { name: "Ver versão" }).click();
    await expect(pageDono).toHaveURL(/\/kit\/versoes\/2$/);
    await expect(pageDono.getByTestId("aviso-versao-somente-leitura")).toBeVisible();
    await expect(pageDono.getByTestId("bloco-o-que-evitar")).toContainText("Promessas milagrosas, garantia total");
    await expect(pageDono.getByTestId("versao-campo-produtos")).toContainText("Consultoria de Crescimento");
    await expect(pageDono.getByTestId("versao-campo-produtos")).toContainText("Prioritário");
    await expect(pageDono.getByTestId("versao-campo-modeloNegocio")).toContainText("Assinatura consultiva");
    await expect(pageDono.getByTestId("versao-campo-canais")).toContainText("Instagram");
    await expect(pageDono.getByTestId("versao-campo-canais")).toContainText("Podcast mensal");
    await expect(pageDono.getByRole("button", { name: "Salvar kit de marca" })).toHaveCount(0);
    await expect(pageDono.getByRole("button", { name: "Confirmar" })).toHaveCount(0);
    await expect(pageDono.locator("input, textarea, select")).toHaveCount(0);

    // Versão 1 ainda mostra o estado anterior (sem Diferencial).
    await pageDono.getByRole("link", { name: "Voltar para versões" }).click();
    await pageDono.getByTestId("versao-1").getByRole("link", { name: "Ver versão" }).click();
    await expect(pageDono).toHaveURL(/\/kit\/versoes\/1$/);
    await expect(pageDono.getByTestId("versao-campo-diferencial")).toContainText("Não informado.");

    // 4) C3: sugestão da IA via fixture -> badge "Sugerido pela IA" -> Confirmar -> "Confirmado por você".
    rodarApoio(["sugerir-kit", orgNome, "diferencial", "Sugestão da IA para diferencial"]);
    rodarApoio(["sugerir-kit", orgNome, "regioes", "Brasil e Portugal (sugestão IA)"]);

    await pageDono.goto("/kit");
    const campoDif = pageDono.getByTestId("campo-diferencial");
    await expect(campoDif).toContainText("Sugerido pela IA");
    await expect(pageDono.getByLabel("Diferencial")).toHaveValue("Sugestão da IA para diferencial");
    await campoDif.getByRole("button", { name: "Confirmar" }).click();
    await expect(campoDif).toContainText("Confirmado por você");
    await expect(campoDif.getByRole("button", { name: "Confirmar" })).toHaveCount(0);

    // 5) C7: pessoa com ver_marca_posts (sem cmo_dna) vê somente leitura e não vê editar/confirmar; PUT/POST bloqueados (403).
    const ctxLeitura = await browser.newContext();
    const pageLeitura = await ctxLeitura.newPage();
    await entrar(pageLeitura, leituraEmail, leituraSenha);
    await pageLeitura.goto("/kit");

    await expect(pageLeitura.getByText("Modo somente leitura")).toBeVisible();
    await expect(pageLeitura.getByTestId("bloco-o-que-evitar")).toContainText("Promessas milagrosas, garantia total");
    await expect(pageLeitura.getByTestId("campo-regioes")).toContainText("Sugerido pela IA");
    await expect(pageLeitura.getByRole("button", { name: "Salvar kit de marca" })).toHaveCount(0);
    await expect(pageLeitura.getByRole("button", { name: "Confirmar" })).toHaveCount(0);

    const resPutSemAlc = await pageLeitura.request.put("/api/kit", {
      data: { versaoBase: 1 },
    });
    expect(resPutSemAlc.status()).toBe(403);

    const resConfirmSemAlc = await pageLeitura.request.post("/api/kit/confirmar", {
      data: { campos: ["regioes"], versaoBase: 1 },
    });
    expect(resConfirmSemAlc.status()).toBe(403);

    // Leitura pode ver versões, mas versão inexistente retorna 404.
    await pageLeitura.getByRole("link", { name: "Versões" }).click();
    await expect(pageLeitura).toHaveURL(/\/kit\/versoes$/);
    await expect(pageLeitura.getByTestId("versao-1")).toBeVisible();
    await pageLeitura.goto("/kit/versoes/999");
    await expect(pageLeitura.getByTestId("versao-nao-encontrada")).toBeVisible();
    const resVersao404 = await pageLeitura.request.get("/api/kit/versoes/999");
    expect(resVersao404.status()).toBe(404);
    await ctxLeitura.close();

    // Pessoa sem ver_marca_posts é bloqueada nas telas e rotas (403).
    const ctxSemAcesso = await browser.newContext();
    const pageSemAcesso = await ctxSemAcesso.newPage();
    await entrar(pageSemAcesso, semAcessoEmail, semAcessoSenha);
    await pageSemAcesso.goto("/kit/versoes");
    await expect(pageSemAcesso.getByTestId("estado-sem-acesso")).toBeVisible();
    await pageSemAcesso.goto("/kit/versoes/1");
    await expect(pageSemAcesso.getByTestId("estado-sem-acesso")).toBeVisible();
    expect((await pageSemAcesso.request.get("/api/kit")).status()).toBe(403);
    expect((await pageSemAcesso.request.get("/api/kit/versoes")).status()).toBe(403);
    expect((await pageSemAcesso.request.get("/api/kit/versoes/1")).status()).toBe(403);
    await ctxSemAcesso.close();

    // 6) C5: salvamento com versão desatualizada mostra mensagem 409 e não perde o texto digitado.
    await pageDono.reload();
    const ctxDono2 = await browser.newContext();
    const pageDono2 = await ctxDono2.newPage();
    await entrar(pageDono2, donoEmail, donoSenha);
    await pageDono2.goto("/kit");
    await pageDono2.getByLabel("Maior desafio").fill("Desafio atualizado em outra aba");
    await pageDono2.getByRole("button", { name: "Salvar kit de marca" }).click();
    await expect(pageDono2.getByRole("status")).toContainText("Kit salvo.");
    await ctxDono2.close();

    const campoObjetivo = pageDono.getByLabel("Objetivo dos próximos 90 dias");
    await campoObjetivo.fill("Texto digitado antes do conflito");
    await pageDono.getByRole("button", { name: "Salvar kit de marca" }).click();
    await expect(pageDono.getByRole("alert")).toContainText(
      "O kit foi alterado por outra pessoa. Recarregue para ver a versão atual.",
    );
    await expect(campoObjetivo).toHaveValue("Texto digitado antes do conflito");

    await ctxDono.close();
  } finally {
    rodarApoio(["limpar"], apoioEnv);
  }
});
