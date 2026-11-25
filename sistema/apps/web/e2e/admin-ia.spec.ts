import { execFileSync } from "node:child_process";
import crypto from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test, type Locator, type Page } from "@playwright/test";
import { carregarEnvEfetivo, validarBancoE2E } from "./env";

const dirAtual = path.dirname(fileURLToPath(import.meta.url));

const envEfetivo = carregarEnvEfetivo();
const bancoE2E = validarBancoE2E(envEfetivo.LG_E2E_BANCO_URL);
const adminEmail = envEfetivo.LG_SEMENTE_EMAIL ?? "admin-e2e@exemplo.test";
const adminSenha = envEfetivo.LG_SEMENTE_SENHA ?? "";
const operacaoEmail = "operacao-e2e@exemplo.test";
const operacaoSenha = envEfetivo.LG_E2E_OPERACAO_SENHA || crypto.randomBytes(18).toString("base64url");

async function entrar(page: Page, email: string, senha: string) {
  await page.goto("/entrar");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel("Senha").fill(senha);
  await page.getByRole("button", { name: "Entrar na plataforma" }).click();
  await expect(page).toHaveURL(/\/inicio$/);
}

async function adicionarPreco(
  page: Page,
  linhaModelo: Locator,
  unidade: "token_entrada" | "token_saida" | "imagem",
  valorUsd: string,
) {
  await linhaModelo.getByRole("button", { name: "Preços" }).click();
  const dialog = page.getByRole("dialog", { name: "Preços do modelo" });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("Unidade").selectOption(unidade);
  await dialog.getByLabel("Valor (US$)").fill(valorUsd);
  await dialog.getByRole("button", { name: "Adicionar preço" }).click();
  const rotuloUnidade =
    unidade === "token_entrada"
      ? "Entrada (US$ por 1 milhão de tokens)"
      : unidade === "token_saida"
        ? "Saída (US$ por 1 milhão de tokens)"
        : "Imagem (US$ por imagem)";
  await expect(dialog.locator("tbody tr", { hasText: rotuloUnidade }).first()).toBeVisible();
  await dialog.getByRole("button", { name: "Fechar" }).click();
  await expect(dialog).toBeHidden();
}

test.beforeAll(() => {
  if (!adminSenha) {
    throw new Error("LG_SEMENTE_SENHA não definida em sistema/.env");
  }
  const backendDir = path.resolve(dirAtual, "../../../backend");
  execFileSync("go", ["run", "./e2e_apoio.go", "preparar"], {
    cwd: backendDir,
    env: {
      ...process.env,
      ...envEfetivo,
      LG_E2E_BANCO_URL: bancoE2E,
      LG_E2E_OPERACAO_EMAIL: operacaoEmail,
      LG_E2E_OPERACAO_SENHA: operacaoSenha,
    },
    stdio: "inherit",
  });
});

test.afterAll(() => {
  const backendDir = path.resolve(dirAtual, "../../../backend");
  execFileSync("go", ["run", "./e2e_apoio.go", "limpar"], {
    cwd: backendDir,
    env: {
      ...process.env,
      ...envEfetivo,
      LG_E2E_BANCO_URL: bancoE2E,
      LG_E2E_OPERACAO_EMAIL: operacaoEmail,
    },
    stdio: "inherit",
  });
});

test("caminho principal da Administração de IA: fornecedor simulado, modelos, preços, testes, atividade com reserva e registros", async ({
  page,
}) => {
  const nomeFornecedor = `Simulado E2E ${Date.now()}`;

  await entrar(page, adminEmail, adminSenha);

  const menuPrincipal = page.getByRole("navigation", { name: "Menu principal" });
  const linkAdmin = menuPrincipal.getByRole("link", { name: "Administração" });
  await expect(linkAdmin).toBeVisible();
  await linkAdmin.click();
  await expect(page).toHaveURL(/\/admin\/fornecedores$/);

  // Verifica que o campo de chave no cadastro começa vazio, usa autoComplete="new-password" e data-bwignore.
  await page.getByRole("button", { name: "Novo fornecedor" }).click();
  const dialogNovo = page.getByRole("dialog", { name: "Novo fornecedor" });
  await expect(dialogNovo).toBeVisible();
  const campoChaveNovo = dialogNovo.getByLabel("Chave de API");
  await expect(campoChaveNovo).toHaveValue("");
  await expect(campoChaveNovo).toHaveAttribute("autocomplete", "new-password");
  await expect(campoChaveNovo).toHaveAttribute("data-bwignore", /^(true)?$/);

  // Cria o fornecedor simulado.
  await dialogNovo.getByLabel("Tipo").selectOption("simulado");
  await dialogNovo.getByLabel("Nome").fill(nomeFornecedor);
  await dialogNovo.getByRole("button", { name: "Salvar fornecedor" }).click();
  await expect(dialogNovo).toBeHidden();

  const linhaFornecedor = page.getByTestId("lista-fornecedores").locator("tr", { hasText: nomeFornecedor });
  await expect(linhaFornecedor).toBeVisible();
  await expect(linhaFornecedor.getByText("Simulado", { exact: true })).toBeVisible();

  // Verifica que o campo "Trocar chave" ao editar fornecedor com chave começa vazio, usa autoComplete="new-password" e data-bwignore.
  const linhaReferencia = page
    .getByTestId("lista-fornecedores")
    .locator("tr", { hasText: "OpenAI (referência E2E)" });
  await expect(linhaReferencia).toBeVisible();
  await linhaReferencia.getByRole("button", { name: "Editar" }).click();
  const dialogEditar = page.getByRole("dialog", { name: "Editar fornecedor" });
  await expect(dialogEditar).toBeVisible();
  const campoTrocarChave = dialogEditar.getByLabel("Trocar chave");
  await expect(campoTrocarChave).toHaveValue("");
  await expect(campoTrocarChave).toHaveAttribute("autocomplete", "new-password");
  await expect(campoTrocarChave).toHaveAttribute("data-bwignore", /^(true)?$/);
  await dialogEditar.getByRole("button", { name: "Fechar" }).click();
  await expect(dialogEditar).toBeHidden();

  // Acessa os modelos do fornecedor simulado, lê os disponíveis e inclui os três modelos.
  await linhaFornecedor.getByRole("link", { name: "Modelos" }).click();
  await expect(page).toHaveURL(/\/admin\/fornecedores\/[^/]+$/);

  await page.getByRole("button", { name: "Ler modelos disponíveis" }).click();
  const secaoLidos = page.locator("section", { hasText: "Modelos disponíveis no fornecedor" });
  await expect(secaoLidos).toBeVisible();

  const linhaLidoTexto = secaoLidos.locator("tr", { hasText: "simulado-texto" });
  await expect(linhaLidoTexto).toBeVisible();
  await linhaLidoTexto.getByRole("button", { name: "Incluir" }).click();
  await expect(linhaLidoTexto.getByText("Já cadastrado")).toBeVisible();

  const linhaLidoImagem = secaoLidos.locator("tr", { hasText: "simulado-imagem" });
  await expect(linhaLidoImagem).toBeVisible();
  await linhaLidoImagem.getByRole("button", { name: "Incluir" }).click();
  await expect(linhaLidoImagem.getByText("Já cadastrado")).toBeVisible();

  // Inclui manualmente o modelo simulado-falha (texto).
  const secaoManual = page.locator("section", { hasText: "Incluir modelo à mão" });
  await secaoManual.getByLabel("Identificador do modelo").fill("simulado-falha");
  await secaoManual.getByLabel("Serviço").selectOption("texto");
  await secaoManual.getByRole("button", { name: "Incluir modelo" }).click();

  const secaoCadastrados = page.locator("section", { hasText: "Modelos cadastrados" });
  const linhaTexto = secaoCadastrados.locator("tr", { hasText: "simulado-texto" });
  const linhaFalha = secaoCadastrados.locator("tr", { hasText: "simulado-falha" });
  const linhaImagem = secaoCadastrados.locator("tr", { hasText: "simulado-imagem" });
  await expect(linhaTexto).toBeVisible();
  await expect(linhaFalha).toBeVisible();
  await expect(linhaImagem).toBeVisible();

  // Cadastra preços vigentes para os três modelos.
  await adicionarPreco(page, linhaTexto, "token_entrada", "1.5");
  await adicionarPreco(page, linhaTexto, "token_saida", "2.5");
  await adicionarPreco(page, linhaFalha, "token_entrada", "1");
  await adicionarPreco(page, linhaFalha, "token_saida", "2");
  await adicionarPreco(page, linhaImagem, "imagem", "0.04");

  // Executa "Testar" em simulado-texto e valida resultado, tokens, US$ e selo Simulado.
  await linhaTexto.getByRole("button", { name: "Testar" }).click();
  const resultadoTeste = page.getByTestId("resultado-teste");
  await expect(resultadoTeste).toBeVisible();
  await expect(resultadoTeste.getByText("Simulado", { exact: true })).toBeVisible();
  await expect(resultadoTeste).toContainText("Resposta simulada (simulado-texto): Oi, qual modelo é você?");
  await expect(resultadoTeste.locator("dd").nth(0)).toHaveText("12");
  await expect(resultadoTeste.locator("dd").nth(1)).toHaveText("18");
  await expect(resultadoTeste.locator("dd").nth(3)).toHaveText("US$ 0,000063");

  // Executa "Testar" em simulado-imagem e valida imagem, quantidade, US$ e selo Simulado.
  await linhaImagem.getByRole("button", { name: "Testar" }).click();
  await expect(resultadoTeste.getByAltText("Imagem gerada no teste")).toBeVisible();
  await expect(resultadoTeste.getByText("Simulado", { exact: true })).toBeVisible();
  await expect(resultadoTeste.locator("dd").nth(2)).toHaveText("1");
  await expect(resultadoTeste.locator("dd").nth(3)).toContainText("US$ 0,04");

  // Configura atividade "texto" com principal simulado-falha e reserva simulado-texto.
  const menuAdmin = page.getByRole("navigation", { name: "Menu da Administração" });
  await menuAdmin.getByRole("link", { name: "Atividades" }).click();
  await expect(page).toHaveURL(/\/admin\/atividades$/);

  const linhaAtividadeTexto = page.getByTestId("atividade-texto");
  const selectPrincipal = linhaAtividadeTexto.getByLabel("Principal de Texto");
  const selectReserva = linhaAtividadeTexto.getByLabel("Reserva de Texto");
  await expect(selectPrincipal).toBeEnabled();
  await selectPrincipal.selectOption({ label: `${nomeFornecedor} — simulado-falha (Simulado)` });
  await selectReserva.selectOption({ label: `${nomeFornecedor} — simulado-texto (Simulado)` });
  await linhaAtividadeTexto.getByRole("button", { name: "Salvar" }).click();
  await expect(linhaAtividadeTexto.getByText("Pronta", { exact: true })).toBeVisible();

  // Testa a atividade e confirma que a reserva assumiu.
  await linhaAtividadeTexto.getByRole("button", { name: "Testar atividade" }).click();
  const resultadoAtiv = page.getByTestId("resultado-atividade");
  await expect(resultadoAtiv).toBeVisible();
  await expect(resultadoAtiv.getByText("Atendida pela reserva")).toBeVisible();
  await expect(resultadoAtiv.getByText("Simulado", { exact: true })).toBeVisible();
  await expect(resultadoAtiv).toContainText(`${nomeFornecedor} — simulado-texto`);
  await expect(resultadoAtiv).toContainText("falhou:");
  await expect(resultadoAtiv).toContainText("Resposta simulada (simulado-texto):");

  // Confirma que a tela Registros mostra as tentativas (falha do principal e sucesso da reserva).
  await menuAdmin.getByRole("link", { name: "Registros" }).click();
  await expect(page).toHaveURL(/\/admin\/registros$/);

  const linhaRegFalha = page
    .locator("tbody tr")
    .filter({ hasText: nomeFornecedor })
    .filter({ hasText: "simulado-falha" })
    .first();
  await expect(linhaRegFalha).toBeVisible();
  await expect(linhaRegFalha.locator("td").nth(5)).toHaveText("1");
  await expect(linhaRegFalha.locator("td").nth(6)).toHaveText("Não");
  await expect(linhaRegFalha.locator("td").nth(7)).toHaveText("Falha");
  await expect(linhaRegFalha).toContainText("Simulado");

  const linhaRegReserva = page
    .locator("tbody tr")
    .filter({ hasText: nomeFornecedor })
    .filter({ hasText: "simulado-texto" })
    .filter({ hasText: "Texto" })
    .first();
  await expect(linhaRegReserva).toBeVisible();
  await expect(linhaRegReserva.locator("td").nth(5)).toHaveText("2");
  await expect(linhaRegReserva.locator("td").nth(6)).toHaveText("Sim");
  await expect(linhaRegReserva.locator("td").nth(7)).toHaveText("Ok");
  await expect(linhaRegReserva).toContainText("Simulado");
  await expect(linhaRegReserva.locator("td").nth(9)).toHaveText("12 / 18");
  await expect(linhaRegReserva.locator("td").nth(11)).toHaveText("US$ 0,000063");
});

test("AC37 e AC30: usuário comum não vê o menu Administração, /admin redireciona para /inicio e Operação não mostra fornecedores", async ({ page }) => {
  await entrar(page, operacaoEmail, operacaoSenha);

  const menuPrincipal = page.getByRole("navigation", { name: "Menu principal" });
  await expect(menuPrincipal).toBeVisible();
  await expect(menuPrincipal.getByRole("link", { name: "Administração" })).toHaveCount(0);

  // AC30: a tela de Operação (/inicio e /kit) não exibe fornecedores nem opções da Administração.
  await expect(page.getByRole("heading", { name: "Organização ativa" })).toBeVisible();
  await expect(page.locator("main")).not.toContainText("OpenAI (referência E2E)");
  await expect(page.locator("main")).not.toContainText("Simulado E2E");
  await expect(page.locator("main")).not.toContainText("Fornecedores");

  await page.goto("/kit");
  await expect(page.getByRole("heading", { name: "Kit de marca" })).toBeVisible();
  await expect(page.getByLabel("Site da marca")).toBeVisible();
  await expect(page.locator("main")).not.toContainText("OpenAI (referência E2E)");
  await expect(page.locator("main")).not.toContainText("Simulado E2E");
  await expect(page.locator("main")).not.toContainText("Fornecedores");

  await page.goto("/admin");
  await expect(page).toHaveURL(/\/inicio$/);
  await expect(page.getByRole("heading", { name: /Olá,/ })).toBeVisible();
  await expect(menuPrincipal.getByRole("link", { name: "Administração" })).toHaveCount(0);

  await page.goto("/admin/fornecedores");
  await expect(page).toHaveURL(/\/inicio$/);
  await expect(menuPrincipal.getByRole("link", { name: "Administração" })).toHaveCount(0);
});
