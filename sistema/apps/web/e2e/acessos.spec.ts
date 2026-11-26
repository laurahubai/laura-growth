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
const donoEmail = envEfetivo.LG_SEMENTE_EMAIL ?? "admin-e2e@exemplo.test";
const donoSenha = envEfetivo.LG_SEMENTE_SENHA ?? "";
const operacaoEmail = "operacao-e2e@exemplo.test";
const operacaoSenha = envEfetivo.LG_E2E_OPERACAO_SENHA || crypto.randomBytes(18).toString("base64url");
const semAcessoEmail = "sem-acesso-e2e@exemplo.test";
const semAcessoSenha = crypto.randomBytes(18).toString("base64url");

async function entrar(page: Page, email: string, senha: string) {
  await page.goto("/entrar");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel("Senha").fill(senha);
  await page.getByRole("button", { name: "Entrar na plataforma" }).click();
  await expect(page).toHaveURL(/\/inicio$/);
}

test.beforeAll(() => {
  if (!donoSenha) {
    throw new Error("LG_SEMENTE_SENHA não definida em sistema/.env");
  }
  execFileSync("go", ["run", "./e2e_apoio.go", "preparar"], {
    cwd: backendDir,
    env: {
      ...process.env,
      ...envEfetivo,
      LG_E2E_BANCO_URL: bancoE2E,
      LG_E2E_OPERACAO_EMAIL: operacaoEmail,
      LG_E2E_OPERACAO_SENHA: operacaoSenha,
      LG_E2E_SEM_ACESSO_EMAIL: semAcessoEmail,
      LG_E2E_SEM_ACESSO_SENHA: semAcessoSenha,
    },
    stdio: "inherit",
  });
});

test.afterAll(() => {
  execFileSync("go", ["run", "./e2e_apoio.go", "limpar"], {
    cwd: backendDir,
    env: {
      ...process.env,
      ...envEfetivo,
      LG_E2E_BANCO_URL: bancoE2E,
      LG_E2E_OPERACAO_EMAIL: operacaoEmail,
      LG_E2E_SEM_ACESSO_EMAIL: semAcessoEmail,
    },
    stdio: "inherit",
  });
});

test("B4: pessoa limitada não vê menus restritos, vê kit somente leitura e é bloqueada por URL direta (403 -> Sem acesso)", async ({
  browser,
}) => {
  const ctxSemAcesso = await browser.newContext();
  const pageSemAcesso = await ctxSemAcesso.newPage();
  await entrar(pageSemAcesso, semAcessoEmail, semAcessoSenha);

  const menuSemAcesso = pageSemAcesso.getByRole("navigation", { name: "Menu principal" });
  await expect(menuSemAcesso.getByRole("link", { name: "Início" })).toBeVisible();
  await expect(menuSemAcesso.getByRole("link", { name: "Kit de marca" })).toHaveCount(0);
  await expect(menuSemAcesso.getByRole("link", { name: "Pessoas" })).toHaveCount(0);

  await pageSemAcesso.goto("/kit");
  const estadoSemAcessoKit = pageSemAcesso.getByTestId("estado-sem-acesso");
  await expect(estadoSemAcessoKit).toBeVisible();
  await expect(estadoSemAcessoKit).toContainText("Sem acesso");

  await pageSemAcesso.goto("/pessoas");
  const estadoSemAcessoPessoas = pageSemAcesso.getByTestId("estado-sem-acesso");
  await expect(estadoSemAcessoPessoas).toBeVisible();
  await expect(estadoSemAcessoPessoas).toContainText("Sem acesso");
  await ctxSemAcesso.close();

  const ctxLeitura = await browser.newContext();
  const pageLeitura = await ctxLeitura.newPage();
  await entrar(pageLeitura, operacaoEmail, operacaoSenha);

  const menuLeitura = pageLeitura.getByRole("navigation", { name: "Menu principal" });
  await expect(menuLeitura.getByRole("link", { name: "Kit de marca" })).toBeVisible();
  await expect(menuLeitura.getByRole("link", { name: "Pessoas" })).toHaveCount(0);

  await pageLeitura.goto("/kit");
  await expect(pageLeitura.getByRole("heading", { name: "Kit de marca" })).toBeVisible();
  await expect(pageLeitura.getByText("Modo somente leitura")).toBeVisible();
  await expect(pageLeitura.getByRole("button", { name: "Salvar kit de marca" })).toHaveCount(0);
  await expect(pageLeitura.getByRole("button", { name: "Adicionar pilar" })).toHaveCount(0);
  await expect(pageLeitura.getByRole("button", { name: "Adicionar cor" })).toHaveCount(0);
  await expect(pageLeitura.getByRole("button", { name: /Enviar logo|Trocar logo/ })).toHaveCount(0);
  await expect(pageLeitura.getByRole("button", { name: "Enviar arquivo" })).toHaveCount(0);

  await pageLeitura.goto("/pessoas");
  await expect(pageLeitura.getByTestId("estado-sem-acesso")).toContainText("Sem acesso");
  await ctxLeitura.close();
});

test("B3, B5, B6, B7: fluxo do dono em Pessoas — editar alçadas com conjunto pronto, efeito imediato, remoção e histórico", async ({
  browser,
}) => {
  const ctxDono = await browser.newContext();
  const pageDono = await ctxDono.newPage();
  await entrar(pageDono, donoEmail, donoSenha);

  const ctxOp = await browser.newContext();
  const pageOp = await ctxOp.newPage();
  await entrar(pageOp, operacaoEmail, operacaoSenha);

  const menuDono = pageDono.getByRole("navigation", { name: "Menu principal" });
  await menuDono.getByRole("link", { name: "Pessoas" }).click();
  await expect(pageDono).toHaveURL(/\/pessoas$/);

  const linhaDono = pageDono.getByTestId(`membro-${donoEmail}`);
  await expect(linhaDono).toBeVisible();
  await expect(linhaDono.getByText("Dono", { exact: true })).toBeVisible();
  await expect(linhaDono.getByRole("button", { name: "Editar alçadas" })).toHaveCount(0);
  await expect(linhaDono.getByRole("button", { name: "Remover" })).toHaveCount(0);

  const linhaOp = pageDono.getByTestId(`membro-${operacaoEmail}`);
  await expect(linhaOp).toBeVisible();
  await linhaOp.getByRole("button", { name: "Editar alçadas" }).click();

  const dialog = pageDono.getByRole("dialog", { name: "Editar alçadas" });
  await expect(dialog).toBeVisible();

  // Seleciona conjunto pronto (pré-preenche as caixas) e ajusta uma caixa manualmente.
  await dialog.getByLabel("Conjunto pronto").selectOption("criador_conteudo");
  await expect(dialog.getByLabel("Ver a marca e os posts")).toBeChecked();
  await expect(dialog.getByLabel("Conversar com o CMO e editar o DNA da marca")).toBeChecked();
  await expect(dialog.getByLabel("Criar e editar posts")).toBeChecked();
  await expect(dialog.getByLabel("Ver consumo em créditos")).not.toBeChecked();

  await dialog.getByLabel("Ver consumo em créditos").check();
  await dialog.getByRole("button", { name: "Salvar alçadas" }).click();
  await expect(dialog).toBeHidden();

  await expect(linhaOp).toContainText("Conversar com o CMO e editar o DNA da marca");
  await expect(linhaOp).toContainText("Ver consumo em créditos");

  // Efeito imediato (B3): operacao-e2e passa a ver botão de salvar no /kit sem refazer login.
  await pageOp.goto("/kit");
  await expect(pageOp.getByRole("button", { name: "Salvar kit de marca" })).toBeVisible();

  // Revoga cmo_dna de volta e valida perda imediata no /kit.
  await linhaOp.getByRole("button", { name: "Editar alçadas" }).click();
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("Conversar com o CMO e editar o DNA da marca").uncheck();
  await dialog.getByRole("button", { name: "Salvar alçadas" }).click();
  await expect(dialog).toBeHidden();

  await pageOp.reload();
  await expect(pageOp.getByText("Modo somente leitura")).toBeVisible();
  await expect(pageOp.getByRole("button", { name: "Salvar kit de marca" })).toHaveCount(0);

  // Remove membro semAcessoEmail com confirmação.
  const linhaSemAcesso = pageDono.getByTestId(`membro-${semAcessoEmail}`);
  await expect(linhaSemAcesso).toBeVisible();
  pageDono.once("dialog", (d) => d.accept());
  await linhaSemAcesso.getByRole("button", { name: "Remover" }).click();
  await expect(linhaSemAcesso).toHaveCount(0);

  // Histórico (B6) registra concessões, revogações e remoção com quem, alvo e o quê.
  const historico = pageDono.getByTestId("historico-acessos");
  await expect(historico).toContainText("Alçada concedida");
  await expect(historico).toContainText("Alçada revogada");
  await expect(historico).toContainText("Pessoa removida");
  await expect(historico).toContainText(operacaoEmail);
  await expect(historico).toContainText(semAcessoEmail);

  await ctxOp.close();
  await ctxDono.close();
});

test("B8: Administração → Equipe lista casa, impede remover último admin, alterna admin e audita", async ({
  page,
}) => {
  await entrar(page, donoEmail, donoSenha);

  await page.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Administração" }).click();
  await page.getByRole("navigation", { name: "Menu da Administração" }).getByRole("link", { name: "Equipe" }).click();
  await expect(page).toHaveURL(/\/admin\/equipe$/);

  const linhaDono = page.getByTestId(`equipe-${donoEmail}`);
  await expect(linhaDono).toBeVisible();
  await expect(linhaDono).toContainText("Administrador");

  // Garante que haja apenas 1 admin antes de validar o erro de último administrador.
  const outrosAdminsRevogados: string[] = [];
  const linhasEquipe = page.getByTestId("lista-equipe").locator("tbody tr");
  const totalLinhas = await linhasEquipe.count();
  const emailsOutrosAdmins: string[] = [];
  for (let i = 0; i < totalLinhas; i++) {
    const linha = linhasEquipe.nth(i);
    const emailLinha = (await linha.locator("td").nth(1).innerText()).trim();
    if (emailLinha === donoEmail) continue;
    if ((await linha.getByRole("button", { name: "Remover admin" }).count()) > 0) {
      emailsOutrosAdmins.push(emailLinha);
    }
  }
  for (const emailLinha of emailsOutrosAdmins) {
    const linha = page.getByTestId(`equipe-${emailLinha}`);
    await linha.getByRole("button", { name: "Remover admin" }).click();
    await expect(linha).toContainText("Sem acesso");
    outrosAdminsRevogados.push(emailLinha);
  }

  try {
    // Tenta remover o único admin da plataforma -> erro 409 claro.
    await linhaDono.getByRole("button", { name: "Remover admin" }).click();
    await expect(page.getByRole("alert").filter({ hasText: "último administrador" })).toBeVisible();
    await expect(linhaDono).toContainText("Administrador");

    // Concede e revoga admin para membro da casa (Laura Growth).
    const linhaOp = page.getByTestId(`equipe-${operacaoEmail}`);
    await expect(linhaOp).toBeVisible();
    await expect(linhaOp).toContainText("Laura Growth");
    await expect(linhaOp).toContainText("Sem acesso");

    await linhaOp.getByRole("button", { name: "Tornar admin" }).click();
    await expect(linhaOp).toContainText("Administrador");

    await linhaOp.getByRole("button", { name: "Remover admin" }).click();
    await expect(linhaOp).toContainText("Sem acesso");

    const auditoria = page.getByTestId("auditoria-equipe");
    await expect(auditoria).toContainText("Administração concedida");
    await expect(auditoria).toContainText("Administração revogada");
    await expect(auditoria).toContainText(operacaoEmail);
  } finally {
    for (const emailAdmin of outrosAdminsRevogados) {
      const linha = page.getByTestId(`equipe-${emailAdmin}`);
      if ((await linha.getByRole("button", { name: "Tornar admin" }).count()) > 0) {
        await linha.getByRole("button", { name: "Tornar admin" }).click();
        await expect(linha).toContainText("Administrador");
      }
    }
  }
});

test.fixme("B1, B2, B9, B10: fluxo completo de convite → aceite (conta nova e conta existente) [aguardando S2]", async ({
  browser,
}) => {
  const convidadoEmail = `convidado-${Date.now()}@exemplo.test`;
  const convidadoSenha = crypto.randomBytes(18).toString("base64url");

  const ctxDono = await browser.newContext();
  const pageDono = await ctxDono.newPage();
  await entrar(pageDono, donoEmail, donoSenha);

  await pageDono.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Pessoas" }).click();

  // Convidar pessoa existente retorna 409 claro.
  const formConvite = pageDono.getByTestId("form-convite");
  await formConvite.getByLabel("E-mail").fill(operacaoEmail);
  await formConvite.getByRole("button", { name: "Enviar convite" }).click();
  await expect(formConvite.getByRole("alert")).toBeVisible();

  // Cria convite para nova pessoa usando conjunto pronto "Somente leitura".
  await formConvite.getByLabel("E-mail").fill(convidadoEmail);
  await formConvite.getByLabel("Conjunto pronto").selectOption("somente_leitura");
  await formConvite.getByRole("button", { name: "Enviar convite" }).click();

  const caixaLink = pageDono.getByTestId("convite-link-gerado");
  await expect(caixaLink).toBeVisible();
  const linkBruto = await caixaLink.getByLabel("Link do convite").inputValue();
  const caminhoConvite = new URL(linkBruto).pathname;

  await expect(pageDono.getByTestId(`convite-${convidadoEmail}`)).toBeVisible();

  // Novo navegador anônimo abre o link do convite e cria conta.
  const ctxConvidado = await browser.newContext();
  const pageConvidado = await ctxConvidado.newPage();
  await pageConvidado.goto(caminhoConvite);

  await expect(pageConvidado.getByRole("heading", { name: /Convite para Laura Growth/ })).toBeVisible();
  await pageConvidado.getByLabel("Nome").fill("Convidado E2E");
  await pageConvidado.getByLabel("Senha").fill(convidadoSenha);
  await pageConvidado.getByRole("button", { name: "Criar conta e aceitar convite" }).click();
  await expect(pageConvidado).toHaveURL(/\/inicio$/);

  // Convidado tem somente leitura: /kit abre, /pessoas bloqueia com Sem acesso.
  await pageConvidado.goto("/kit");
  await expect(pageConvidado.getByText("Modo somente leitura")).toBeVisible();
  await pageConvidado.goto("/pessoas");
  await expect(pageConvidado.getByTestId("estado-sem-acesso")).toContainText("Sem acesso");

  // Reabrir link já usado mostra convite inválido (404).
  await pageConvidado.goto(caminhoConvite);
  await expect(pageConvidado.getByRole("heading", { name: /Convite inválido/ })).toBeVisible();

  await ctxConvidado.close();
  await ctxDono.close();
});
