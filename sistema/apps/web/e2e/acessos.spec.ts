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

function caminhoDoLinkConvite(linkBruto: string): string {
  return linkBruto.startsWith("http") ? new URL(linkBruto).pathname : linkBruto;
}

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
});

test("B4: pessoa limitada não vê menus restritos, vê kit somente leitura e é bloqueada por URL direta (403 -> Sem acesso)", async ({
  browser,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  const operacaoEmail = `op-b4-${tag}@exemplo.test`;
  const operacaoSenha = senhaAleatoria();
  const semAcessoEmail = `sem-b4-${tag}@exemplo.test`;
  const semAcessoSenha = senhaAleatoria();

  const apoioEnv = {
    LG_E2E_OPERACAO_EMAIL: operacaoEmail,
    LG_E2E_OPERACAO_SENHA: operacaoSenha,
    LG_E2E_SEM_ACESSO_EMAIL: semAcessoEmail,
    LG_E2E_SEM_ACESSO_SENHA: semAcessoSenha,
  };
  rodarApoio(["preparar"], apoioEnv);

  try {
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
  } finally {
    rodarApoio(["limpar"], apoioEnv);
  }
});

test("B3, B5, B6, B7: fluxo em Pessoas — dono e própria conta protegidos, conjunto pronto, foco na gaveta, efeito imediato, remoção e histórico", async ({
  browser,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  const orgNome = `Org Pessoas E2E ${tag}`;
  const orgDonoEmail = `dono-p-${tag}@exemplo.test`;
  const orgDonoSenha = senhaAleatoria();
  const operacaoEmail = `op-p-${tag}@exemplo.test`;
  const operacaoSenha = senhaAleatoria();
  const removerEmail = `rem-p-${tag}@exemplo.test`;
  const removerSenha = senhaAleatoria();
  const gerenciadorEmail = `ger-p-${tag}@exemplo.test`;
  const gerenciadorSenha = senhaAleatoria();

  const apoioEnv = {
    LG_E2E_ORG_NOME: orgNome,
    LG_E2E_ORG_DONO_EMAIL: orgDonoEmail,
    LG_E2E_ORG_DONO_SENHA: orgDonoSenha,
    LG_E2E_OPERACAO_EMAIL: operacaoEmail,
    LG_E2E_OPERACAO_SENHA: operacaoSenha,
    LG_E2E_REMOVER_EMAIL: removerEmail,
    LG_E2E_REMOVER_SENHA: removerSenha,
    LG_E2E_GERENCIADOR_EMAIL: gerenciadorEmail,
    LG_E2E_GERENCIADOR_SENHA: gerenciadorSenha,
  };
  rodarApoio(["preparar"], apoioEnv);

  try {
    // Membro com gerenciar_pessoas (não dono) vê "Sua conta" na própria linha e "Dono não editável" no dono (B5).
    const ctxGer = await browser.newContext();
    const pageGer = await ctxGer.newPage();
    await entrar(pageGer, gerenciadorEmail, gerenciadorSenha);
    await pageGer.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Pessoas" }).click();
    await expect(pageGer).toHaveURL(/\/pessoas$/);

    const linhaPropria = pageGer.getByTestId(`membro-${gerenciadorEmail}`);
    await expect(linhaPropria).toContainText("Sua conta");
    await expect(linhaPropria.getByRole("button", { name: "Editar alçadas" })).toHaveCount(0);
    await expect(linhaPropria.getByRole("button", { name: "Remover" })).toHaveCount(0);
    await ctxGer.close();

    const ctxDono = await browser.newContext();
    const pageDono = await ctxDono.newPage();
    await entrar(pageDono, orgDonoEmail, orgDonoSenha);

    const ctxOp = await browser.newContext();
    const pageOp = await ctxOp.newPage();
    await entrar(pageOp, operacaoEmail, operacaoSenha);

    await pageDono.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Pessoas" }).click();
    await expect(pageDono).toHaveURL(/\/pessoas$/);

    const linhaDono = pageDono.getByTestId(`membro-${orgDonoEmail}`);
    await expect(linhaDono).toBeVisible();
    await expect(linhaDono.getByText("Dono", { exact: true })).toBeVisible();
    await expect(linhaDono.getByRole("button", { name: "Editar alçadas" })).toHaveCount(0);
    await expect(linhaDono.getByRole("button", { name: "Remover" })).toHaveCount(0);

    // G-5 / G-6: foco entra na primeira caixa da gaveta ao abrir e volta ao botão "Editar alçadas" ao fechar.
    const linhaOp = pageDono.getByTestId(`membro-${operacaoEmail}`);
    const botaoEditarOp = linhaOp.getByRole("button", { name: "Editar alçadas" });
    await botaoEditarOp.click();

    const dialog = pageDono.getByRole("dialog", { name: "Editar alçadas" });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByLabel("Ver a marca e os posts")).toBeFocused();

    await pageDono.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(botaoEditarOp).toBeFocused();

    // Seleciona conjunto pronto (pré-preenche as caixas) e ajusta manualmente (B7).
    await botaoEditarOp.click();
    await expect(dialog).toBeVisible();
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

    // Efeito imediato (B3): operacao passa a ver botão de salvar no /kit sem refazer login.
    await pageOp.goto("/kit");
    await expect(pageOp.getByRole("button", { name: "Salvar kit de marca" })).toBeVisible();

    // Revoga cmo_dna de volta e valida perda imediata no /kit.
    await botaoEditarOp.click();
    await expect(dialog).toBeVisible();
    await dialog.getByLabel("Conversar com o CMO e editar o DNA da marca").uncheck();
    await dialog.getByRole("button", { name: "Salvar alçadas" }).click();
    await expect(dialog).toBeHidden();

    await pageOp.reload();
    await expect(pageOp.getByText("Modo somente leitura")).toBeVisible();
    await expect(pageOp.getByRole("button", { name: "Salvar kit de marca" })).toHaveCount(0);

    // Remove membro dedicado com confirmação.
    const linhaRemover = pageDono.getByTestId(`membro-${removerEmail}`);
    await expect(linhaRemover).toBeVisible();
    pageDono.once("dialog", (d) => d.accept());
    await linhaRemover.getByRole("button", { name: "Remover" }).click();
    await expect(linhaRemover).toHaveCount(0);

    // Histórico (B6) registra concessões, revogações e remoção.
    const historico = pageDono.getByTestId("historico-acessos");
    await expect(historico).toContainText("Alçada concedida");
    await expect(historico).toContainText("Alçada revogada");
    await expect(historico).toContainText("Pessoa removida");
    await expect(historico).toContainText(operacaoEmail);
    await expect(historico).toContainText(removerEmail);

    await ctxOp.close();
    await ctxDono.close();
  } finally {
    rodarApoio(["limpar"], apoioEnv);
  }
});

test("B8: Administração → Equipe lista casa, pede confirmação, impede remover último admin, alterna admin e audita", async ({
  page,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  const membroCasaEmail = `casa-b8-${tag}@exemplo.test`;
  const membroCasaSenha = senhaAleatoria();
  const apoioEnv = {
    LG_E2E_OPERACAO_EMAIL: membroCasaEmail,
    LG_E2E_OPERACAO_SENHA: membroCasaSenha,
  };
  rodarApoio(["preparar"], apoioEnv);

  const outrosAdminsRevogados: string[] = [];

  try {
    await entrar(page, donoEmail, donoSenha);

    await page.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Administração" }).click();
    await page.getByRole("navigation", { name: "Menu da Administração" }).getByRole("link", { name: "Equipe" }).click();
    await expect(page).toHaveURL(/\/admin\/equipe$/);

    const linhaDono = page.getByTestId(`equipe-${donoEmail}`);
    await expect(linhaDono).toBeVisible();
    await expect(linhaDono).toContainText("Administrador");

    // Se houver outros admins pré-existentes no banco E2E, revoga temporariamente (e restaura no finally).
    const linhasEquipe = page.getByTestId("lista-equipe").locator("tbody tr");
    const totalLinhas = await linhasEquipe.count();
    const emailsOutrosAdmins: string[] = [];
    for (let i = 0; i < totalLinhas; i++) {
      const linha = linhasEquipe.nth(i);
      const emailLinha = (await linha.locator("td").nth(1).innerText()).trim();
      if (emailLinha === donoEmail) continue;
      if ((await linha.getByRole("button", { name: "Retirar acesso de administrador" }).count()) > 0) {
        emailsOutrosAdmins.push(emailLinha);
      }
    }
    for (const emailLinha of emailsOutrosAdmins) {
      const linha = page.getByTestId(`equipe-${emailLinha}`);
      page.once("dialog", (d) => d.accept());
      await linha.getByRole("button", { name: "Retirar acesso de administrador" }).click();
      await expect(linha).toContainText("Não é administrador");
      outrosAdminsRevogados.push(emailLinha);
    }

    // Tenta remover o único admin da plataforma (confirma diálogo -> erro 409 claro).
    page.once("dialog", (d) => d.accept());
    await linhaDono.getByRole("button", { name: "Retirar acesso de administrador" }).click();
    await expect(page.getByRole("alert").filter({ hasText: "último administrador" })).toBeVisible();
    await expect(linhaDono).toContainText("Administrador");

    // Concede e revoga admin (com confirmação) para membro da casa (Laura Growth).
    const linhaCasa = page.getByTestId(`equipe-${membroCasaEmail}`);
    await expect(linhaCasa).toBeVisible();
    await expect(linhaCasa).toContainText("Laura Growth");
    await expect(linhaCasa).toContainText("Não é administrador");

    await linhaCasa.getByRole("button", { name: "Dar acesso de administrador" }).click();
    await expect(linhaCasa).toContainText("Administrador");

    page.once("dialog", (d) => d.accept());
    await linhaCasa.getByRole("button", { name: "Retirar acesso de administrador" }).click();
    await expect(linhaCasa).toContainText("Não é administrador");

    const auditoria = page.getByTestId("auditoria-equipe");
    await expect(auditoria).toContainText("Administração concedida");
    await expect(auditoria).toContainText("Administração revogada");
    await expect(auditoria).toContainText(membroCasaEmail);
  } finally {
    for (const emailAdmin of outrosAdminsRevogados) {
      const linha = page.getByTestId(`equipe-${emailAdmin}`);
      if ((await linha.getByRole("button", { name: "Dar acesso de administrador" }).count()) > 0) {
        await linha.getByRole("button", { name: "Dar acesso de administrador" }).click();
        await expect(linha).toContainText("Administrador");
      }
    }
    rodarApoio(["limpar"], apoioEnv);
  }
});

test("B1, B2, B4, B6, B9: convite → zero alçada bloqueado → reenvio → aceite (conta nova e existente) → expirado → cancelamento", async ({
  browser,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  const orgNome = `Org Convite E2E ${tag}`;
  const orgDonoEmail = `dono-conv-${tag}@exemplo.test`;
  const orgDonoSenha = senhaAleatoria();
  const membroEmail = `membro-conv-${tag}@exemplo.test`;
  const membroSenha = senhaAleatoria();
  const existenteEmail = `existente-conv-${tag}@exemplo.test`;
  const existenteSenha = senhaAleatoria();

  const convidadoEmail = `convidado-e2e-${tag}@exemplo.test`;
  const convidadoSenha = senhaAleatoria();
  const cancelarEmail = `convidado-e2e-cancelar-${tag}@exemplo.test`;
  const expiradoEmail = `convidado-e2e-expirado-${tag}@exemplo.test`;

  const apoioEnv = {
    LG_E2E_ORG_NOME: orgNome,
    LG_E2E_ORG_DONO_EMAIL: orgDonoEmail,
    LG_E2E_ORG_DONO_SENHA: orgDonoSenha,
    LG_E2E_OPERACAO_EMAIL: membroEmail,
    LG_E2E_OPERACAO_SENHA: membroSenha,
    LG_E2E_EXISTENTE_EMAIL: existenteEmail,
    LG_E2E_EXISTENTE_SENHA: existenteSenha,
  };
  rodarApoio(["preparar"], apoioEnv);

  try {
    const ctxDono = await browser.newContext();
    const pageDono = await ctxDono.newPage();
    await entrar(pageDono, orgDonoEmail, orgDonoSenha);

    await pageDono.getByRole("navigation", { name: "Menu principal" }).getByRole("link", { name: "Pessoas" }).click();
    await expect(pageDono).toHaveURL(/\/pessoas$/);

    const formConvite = pageDono.getByTestId("form-convite");

    // G-4: convite com zero alçadas mostra aviso e bloqueia o envio.
    await formConvite.getByLabel("E-mail").fill(convidadoEmail);
    await formConvite.getByRole("button", { name: "Enviar convite" }).click();
    await expect(formConvite.getByRole("alert")).toContainText("Selecione pelo menos uma alçada");

    // Convidar membro existente retorna 409 claro.
    await formConvite.getByLabel("E-mail").fill(membroEmail);
    await formConvite.getByLabel("Ver a marca e os posts").check();
    await formConvite.getByRole("button", { name: "Enviar convite" }).click();
    await expect(formConvite.getByRole("alert")).toContainText("já faz parte da organização");

    // Cria convite para nova pessoa usando conjunto pronto "Somente leitura".
    await formConvite.getByLabel("E-mail").fill(convidadoEmail);
    await formConvite.getByLabel("Conjunto pronto").selectOption("somente_leitura");
    await formConvite.getByRole("button", { name: "Enviar convite" }).click();

    const caixaLink = pageDono.getByTestId("convite-link-gerado");
    await expect(caixaLink).toBeVisible();
    await expect(caixaLink.getByRole("button", { name: "Copiar link" })).toBeVisible();
    await expect(caixaLink.getByRole("alert")).toBeVisible();
    const campoLink = caixaLink.getByLabel("Link do convite");
    const linkAntigo = await campoLink.inputValue();
    const caminhoAntigo = caminhoDoLinkConvite(linkAntigo);

    const linhaConvite = pageDono.getByTestId(`convite-${convidadoEmail}`);
    await expect(linhaConvite).toBeVisible();
    await expect(linhaConvite).toContainText("Não enviado");

    // Reenviar gera novo token (invalida o anterior) e mostra novo Copiar link.
    await linhaConvite.getByRole("button", { name: "Reenviar" }).click();
    await expect(campoLink).not.toHaveValue(linkAntigo);
    const caminhoNovo = caminhoDoLinkConvite(await campoLink.inputValue());
    expect(caminhoNovo).not.toBe(caminhoAntigo);

    // Recarregar a página não mostra mais "Copiar link".
    await pageDono.reload();
    await expect(pageDono.getByTestId("convite-link-gerado")).toHaveCount(0);

    // Novo navegador anônimo valida token antigo invalidado (404), S-1 Referrer-Policy e aceite (conta nova).
    const ctxConvidado = await browser.newContext();
    const pageConvidado = await ctxConvidado.newPage();

    await pageConvidado.goto(caminhoAntigo);
    await expect(pageConvidado.getByRole("heading", { name: /Convite inválido/ })).toBeVisible();

    const respostaConvite = await pageConvidado.goto(caminhoNovo);
    expect(respostaConvite?.headers()["referrer-policy"]).toBe("no-referrer");
    await expect(pageConvidado.getByRole("heading", { name: new RegExp(`Convite para ${orgNome}`) })).toBeVisible();
    await expect(pageConvidado.getByText("co***@exemplo.test")).toBeVisible();
    await pageConvidado.getByLabel("Nome").fill("Convidado E2E");
    await pageConvidado.getByLabel("Senha").fill(convidadoSenha);
    await pageConvidado.getByRole("button", { name: "Criar conta e aceitar convite" }).click();
    await expect(pageConvidado).toHaveURL(/\/inicio$/);

    // Convidado tem somente leitura: vê apenas os menus permitidos, /kit abre sem botões de edição e /pessoas bloqueia com Sem acesso.
    const menuConvidado = pageConvidado.getByRole("navigation", { name: "Menu principal" });
    await expect(menuConvidado.getByRole("link", { name: "Início" })).toBeVisible();
    await expect(menuConvidado.getByRole("link", { name: "Kit de marca" })).toBeVisible();
    await expect(menuConvidado.getByRole("link", { name: "Pessoas" })).toHaveCount(0);

    await pageConvidado.goto("/kit");
    await expect(pageConvidado.getByText("Modo somente leitura")).toBeVisible();
    await expect(pageConvidado.getByRole("button", { name: "Salvar kit de marca" })).toHaveCount(0);
    await pageConvidado.goto("/pessoas");
    await expect(pageConvidado.getByTestId("estado-sem-acesso")).toContainText("Sem acesso");

    // Uso único: reabrir link já aceito mostra convite inválido (404).
    await pageConvidado.goto(caminhoNovo);
    await expect(pageConvidado.getByRole("heading", { name: /Convite inválido/ })).toBeVisible();
    await ctxConvidado.close();

    // Histórico e lista de membros refletem o aceite do convite.
    await pageDono.reload();
    await expect(pageDono.getByTestId(`membro-${convidadoEmail}`)).toBeVisible();
    await expect(pageDono.getByTestId(`convite-${convidadoEmail}`)).toHaveCount(0);
    const historico = pageDono.getByTestId("historico-acessos");
    await expect(historico).toContainText("Convite criado");
    await expect(historico).toContainText("Convite reenviado");
    await expect(historico).toContainText("Convite aceito");

    // Conta existente: 403 email_diferente ao aceitar logado com outra conta, depois login e aceite.
    await formConvite.getByLabel("E-mail").fill(existenteEmail);
    await formConvite.getByLabel("Ver a marca e os posts").check();
    await formConvite.getByRole("button", { name: "Enviar convite" }).click();
    await expect(caixaLink).toBeVisible();
    const caminhoExistente = caminhoDoLinkConvite(await caixaLink.getByLabel("Link do convite").inputValue());

    const ctxExistente = await browser.newContext();
    const pageExistente = await ctxExistente.newPage();
    await entrar(pageExistente, membroEmail, membroSenha);
    await pageExistente.goto(caminhoExistente);
    await expect(pageExistente.getByText(`conectado como ${membroEmail}`)).toBeVisible();
    await pageExistente.getByRole("button", { name: "Aceitar convite" }).click();
    await expect(pageExistente.getByRole("alert").filter({ hasText: "outro e-mail" })).toBeVisible();

    await pageExistente.getByRole("button", { name: /Sair e entrar com o e-mail convidado/ }).click();
    await pageExistente.getByLabel("E-mail").fill(existenteEmail);
    await pageExistente.getByLabel("Senha").fill(existenteSenha);
    await pageExistente.getByRole("button", { name: "Entrar para aceitar" }).click();
    await expect(pageExistente.getByText(`conectado como ${existenteEmail}`)).toBeVisible();
    await pageExistente.getByRole("button", { name: "Aceitar convite" }).click();
    await expect(pageExistente).toHaveURL(/\/inicio$/);
    await ctxExistente.close();

    // Convite expirado (B2): expira via e2e_apoio e valida mensagem de convite inválido na tela.
    await formConvite.getByLabel("E-mail").fill(expiradoEmail);
    await formConvite.getByLabel("Ver a marca e os posts").check();
    await formConvite.getByRole("button", { name: "Enviar convite" }).click();
    await expect(caixaLink).toBeVisible();
    const caminhoExpirado = caminhoDoLinkConvite(await caixaLink.getByLabel("Link do convite").inputValue());
    rodarApoio(["expirar-convite", expiradoEmail]);

    const ctxExp = await browser.newContext();
    const pageExp = await ctxExp.newPage();
    await pageExp.goto(caminhoExpirado);
    await expect(pageExp.getByRole("heading", { name: /Convite inválido/ })).toBeVisible();
    await ctxExp.close();

    // Cancelar convite pendente remove da lista e invalida o link.
    await formConvite.getByLabel("E-mail").fill(cancelarEmail);
    await formConvite.getByLabel("Ver a marca e os posts").check();
    await formConvite.getByRole("button", { name: "Enviar convite" }).click();
    await expect(caixaLink).toBeVisible();
    const caminhoCancelado = caminhoDoLinkConvite(await caixaLink.getByLabel("Link do convite").inputValue());

    const linhaCancelar = pageDono.getByTestId(`convite-${cancelarEmail}`);
    await expect(linhaCancelar).toBeVisible();
    pageDono.once("dialog", (d) => d.accept());
    await linhaCancelar.getByRole("button", { name: "Cancelar" }).click();
    await expect(linhaCancelar).toHaveCount(0);

    const ctxAnon = await browser.newContext();
    const pageAnon = await ctxAnon.newPage();
    await pageAnon.goto(caminhoCancelado);
    await expect(pageAnon.getByRole("heading", { name: /Convite inválido/ })).toBeVisible();
    await ctxAnon.close();

    await ctxDono.close();
  } finally {
    rodarApoio(["limpar"], apoioEnv);
  }
});

test("B10: limites de 20 envios/hora e 30 convites pendentes retornam 429 com mensagem clara na tela", async ({
  browser,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;

  // 1) Limite de 20 envios por hora (via page.request até o limite, depois tentativa pela tela).
  const orgHora = `Org B10 Hora ${tag}`;
  const donoHoraEmail = `b10-hora-${tag}@exemplo.test`;
  const donoHoraSenha = senhaAleatoria();
  const envHora = {
    LG_E2E_ORG_NOME: orgHora,
    LG_E2E_ORG_DONO_EMAIL: donoHoraEmail,
    LG_E2E_ORG_DONO_SENHA: donoHoraSenha,
  };
  rodarApoio(["preparar"], envHora);

  try {
    const ctxHora = await browser.newContext();
    const pageHora = await ctxHora.newPage();
    await entrar(pageHora, donoHoraEmail, donoHoraSenha);

    for (let i = 0; i < 20; i++) {
      const res = await pageHora.request.post("/api/convites", {
        data: {
          email: `convidado-e2e-h-${tag}-${i}@exemplo.test`,
          alcadas: ["ver_marca_posts"],
        },
      });
      expect(res.status()).toBe(201);
    }

    await pageHora.goto("/pessoas");
    const formHora = pageHora.getByTestId("form-convite");
    await formHora.getByLabel("E-mail").fill(`convidado-e2e-h-${tag}-20@exemplo.test`);
    await formHora.getByLabel("Ver a marca e os posts").check();
    await formHora.getByRole("button", { name: "Enviar convite" }).click();
    await expect(formHora.getByRole("alert")).toContainText("20 envios de convite por hora");
    await ctxHora.close();
  } finally {
    rodarApoio(["limpar"], envHora);
  }

  // 2) Limite de 30 convites pendentes por organização.
  const orgPend = `Org B10 Pendentes ${tag}`;
  const donoPendEmail = `b10-pend-${tag}@exemplo.test`;
  const donoPendSenha = senhaAleatoria();
  const envPend = {
    LG_E2E_ORG_NOME: orgPend,
    LG_E2E_ORG_DONO_EMAIL: donoPendEmail,
    LG_E2E_ORG_DONO_SENHA: donoPendSenha,
  };
  rodarApoio(["preparar"], envPend);

  try {
    rodarApoio(["semear-pendentes", orgPend, "30"]);

    const ctxPend = await browser.newContext();
    const pagePend = await ctxPend.newPage();
    await entrar(pagePend, donoPendEmail, donoPendSenha);
    await pagePend.goto("/pessoas");

    const formPend = pagePend.getByTestId("form-convite");
    await formPend.getByLabel("E-mail").fill(`convidado-e2e-p-${tag}-31@exemplo.test`);
    await formPend.getByLabel("Ver a marca e os posts").check();
    await formPend.getByRole("button", { name: "Enviar convite" }).click();
    await expect(formPend.getByRole("alert")).toContainText("30 convites pendentes");
    await ctxPend.close();
  } finally {
    rodarApoio(["limpar"], envPend);
  }
});

test("S-2: parâmetro ?proximo= no login aceita apenas /convite/[token] e ignora desvios", async ({
  page,
}) => {
  const tag = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
  const email = `op-s2-${tag}@exemplo.test`;
  const senha = senhaAleatoria();
  const apoioEnv = {
    LG_E2E_OPERACAO_EMAIL: email,
    LG_E2E_OPERACAO_SENHA: senha,
  };
  rodarApoio(["preparar"], apoioEnv);

  try {
    await page.goto("/entrar?proximo=/convite/../kit");
    await page.getByLabel("E-mail").fill(email);
    await page.getByLabel("Senha").fill(senha);
    await page.getByRole("button", { name: "Entrar na plataforma" }).click();
    await expect(page).toHaveURL(/\/inicio$/);
  } finally {
    rodarApoio(["limpar"], apoioEnv);
  }
});
