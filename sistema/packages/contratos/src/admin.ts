// Etapa 2A-1: Administração da plataforma (cadastro de IA por chave de API).
// Contrato entre as telas (/admin) e o servidor (rotas /api/admin/*).
// Espelha backend/admin_*.go: ao mudar um, mude o outro.
//
// Regras gerais:
// - Todas as rotas /api/admin/* exigem sessão e usuario.adminPlataforma = true;
//   caso contrário o servidor responde 403 { codigo: "sem_permissao" }.
// - Valores em US$ são números JSON (ex.: 12.5). Tokens e imagens são inteiros.
// - A chave de API nunca volta em resposta nenhuma: só chaveFinal (4 últimos caracteres).
// - Datas: ISO 8601 (criadoEm) e "AAAA-MM-DD" (vigenciaInicio).
// - Erros: { codigo, mensagem } (ErroApi), com mensagem pronta para mostrar na tela.
//   Códigos transversais comuns: 400 dados_invalidos | formato_invalido,
//   403 sem_permissao, 404 nao_encontrado, 500 falha_interna (mensagem genérica, sem detalhes internos).
import { z } from "zod";

// ---- Enumerações ----

export const TipoFornecedor = z.enum(["openai", "compativel_openai", "anthropic", "simulado"]);
export type TipoFornecedor = z.infer<typeof TipoFornecedor>;

export const nomeTipoFornecedor: Record<TipoFornecedor, string> = {
  openai: "OpenAI",
  compativel_openai: "Compatível com OpenAI",
  anthropic: "Anthropic",
  simulado: "Simulado (para testes)",
};

// Endereço preenchido sozinho pelo tipo (editável). Compatível: obrigatório e livre. Simulado: sem endereço.
export const enderecoPadraoFornecedor: Record<TipoFornecedor, string> = {
  openai: "https://api.openai.com/v1",
  compativel_openai: "",
  anthropic: "https://api.anthropic.com/v1",
  simulado: "",
};

export const Servico = z.enum(["texto", "imagem"]);
export type Servico = z.infer<typeof Servico>;

export const UsoPermitido = z.enum(["todas", "casa"]);
export type UsoPermitido = z.infer<typeof UsoPermitido>;

export const SituacaoLimite = z.enum(["normal", "aviso", "limite"]);
export type SituacaoLimite = z.infer<typeof SituacaoLimite>;

export const UnidadePreco = z.enum(["token_entrada", "token_saida", "imagem"]);
export type UnidadePreco = z.infer<typeof UnidadePreco>;

// Tokens: US$ por 1 milhão de tokens. Imagem: US$ por imagem.
export const nomeUnidadePreco: Record<UnidadePreco, string> = {
  token_entrada: "Entrada (US$ por 1 milhão de tokens)",
  token_saida: "Saída (US$ por 1 milhão de tokens)",
  imagem: "Imagem (US$ por imagem)",
};

export const Atividade = z.enum(["chat_cmo", "plano", "pauta", "texto", "imagem", "revisao"]);
export type Atividade = z.infer<typeof Atividade>;

export const nomeAtividade: Record<Atividade, string> = {
  chat_cmo: "Chat do CMO",
  plano: "Plano de marketing",
  pauta: "Pauta",
  texto: "Texto",
  imagem: "Imagem",
  revisao: "Revisão (Guardian)",
};

// Serviço exigido por atividade: "imagem" usa modelos de imagem; as demais, de texto.
export const servicoAtividade: Record<Atividade, Servico> = {
  chat_cmo: "texto",
  plano: "texto",
  pauta: "texto",
  texto: "texto",
  imagem: "imagem",
  revisao: "texto",
};

// ---- Fornecedores ----

export const Fornecedor = z.object({
  id: z.string(),
  nome: z.string(),
  tipo: TipoFornecedor,
  cadastro: z.enum(["chave_api", "assinatura"]),
  endereco: z.string(),
  chaveFinal: z.string().nullable(), // 4 últimos caracteres; null se não há chave (simulado)
  usoPermitido: UsoPermitido,
  limiteMensalUsd: z.number(),
  avisoPercentual: z.number().int(),
  ativo: z.boolean(),
  gastoMesUsd: z.number(),
  situacao: SituacaoLimite,
  simulado: z.boolean(),
  criadoEm: z.string(),
});
export type Fornecedor = z.infer<typeof Fornecedor>;

// GET /api/admin/fornecedores
export const RespostaFornecedores = z.object({ fornecedores: z.array(Fornecedor) });

// POST /api/admin/fornecedores -> 201 Fornecedor
// chave: obrigatória, exceto no simulado; mínimo de 12 caracteres ASCII visíveis (máx. 500, sem espaços/quebras).
// endereco: se vazio, usa o padrão do tipo.
// Erros: 400 dados_invalidos | formato_invalido (chave fora do padrão) | endereco_invalido (sem https ou rede interna) | chave_obrigatoria; 409 nome_repetido.
export const PedidoNovoFornecedor = z.object({
  nome: z.string().trim().min(1, "Informe o nome.").max(100),
  tipo: TipoFornecedor,
  endereco: z.string().trim().max(500).optional(),
  chave: z.string().max(500).optional(),
  usoPermitido: UsoPermitido.optional(), // padrão "todas"
  limiteMensalUsd: z.number().positive("Informe um limite maior que zero."),
  avisoPercentual: z.number().int().min(1).max(100).optional(), // padrão 80
});
export type PedidoNovoFornecedor = z.infer<typeof PedidoNovoFornecedor>;

// PATCH /api/admin/fornecedores/{id} -> 200 Fornecedor. Campos ausentes não mudam.
// chave: troca a chave (mínimo de 12 caracteres ASCII visíveis, máx. 500). ativo=false desativa (nada é apagado). O tipo não muda.
// Erros: 400 dados_invalidos | formato_invalido | endereco_invalido; 404 nao_encontrado; 409 nome_repetido.
export const PedidoEditarFornecedor = z.object({
  nome: z.string().trim().min(1).max(100).optional(),
  endereco: z.string().trim().max(500).optional(),
  chave: z.string().min(1).max(500).optional(),
  usoPermitido: UsoPermitido.optional(),
  limiteMensalUsd: z.number().positive().optional(),
  avisoPercentual: z.number().int().min(1).max(100).optional(),
  ativo: z.boolean().optional(),
});
export type PedidoEditarFornecedor = z.infer<typeof PedidoEditarFornecedor>;

// POST /api/admin/fornecedores/{id}/ler-modelos (sem corpo) -> 200
// Erros com mensagem clara e sem a chave: 502 fornecedor_recusou_chave | fornecedor_indisponivel |
// fornecedor_erro; 400 endereco_invalido; 409 fornecedor_inativo.
export const RespostaLerModelos = z.object({
  modelos: z.array(z.object({ identificador: z.string(), jaCadastrado: z.boolean() })),
});

// ---- Modelos ----

export const PrecosVigentes = z.object({
  tokenEntrada: z.number().nullable(),
  tokenSaida: z.number().nullable(),
  imagem: z.number().nullable(),
});

export const Modelo = z.object({
  id: z.string(),
  fornecedorId: z.string(),
  fornecedorNome: z.string(),
  identificador: z.string(),
  servico: Servico,
  origem: z.enum(["lido", "manual"]),
  limiteMensalUsd: z.number().nullable(), // null = usa o do fornecedor
  avisoPercentual: z.number().int().nullable(),
  ativo: z.boolean(),
  gastoMesUsd: z.number(),
  situacao: SituacaoLimite, // considera o limite próprio, ou o do fornecedor se não houver
  simulado: z.boolean(),
  precosVigentes: PrecosVigentes,
});
export type Modelo = z.infer<typeof Modelo>;

// GET /api/admin/fornecedores/{id}/modelos
export const RespostaModelos = z.object({ modelos: z.array(Modelo) });

// POST /api/admin/fornecedores/{id}/modelos -> 201 Modelo
// Erros: 409 modelo_repetido; 400 servico_invalido (ex.: imagem no Anthropic).
export const PedidoNovoModelo = z.object({
  identificador: z.string().trim().min(1, "Informe o modelo.").max(200),
  servico: Servico,
  origem: z.enum(["lido", "manual"]),
  limiteMensalUsd: z.number().positive().nullable().optional(),
  avisoPercentual: z.number().int().min(1).max(100).nullable().optional(),
});
export type PedidoNovoModelo = z.infer<typeof PedidoNovoModelo>;

// PATCH /api/admin/modelos/{id} -> 200 Modelo. null em limite/aviso volta a usar o do fornecedor.
export const PedidoEditarModelo = z.object({
  servico: Servico.optional(),
  limiteMensalUsd: z.number().positive().nullable().optional(),
  avisoPercentual: z.number().int().min(1).max(100).nullable().optional(),
  ativo: z.boolean().optional(),
});
export type PedidoEditarModelo = z.infer<typeof PedidoEditarModelo>;

// ---- Preços (só acrescenta; histórico preservado) ----

export const Preco = z.object({
  id: z.string(),
  modeloId: z.string(),
  unidade: UnidadePreco,
  precoUsd: z.number(),
  vigenciaInicio: z.string(), // AAAA-MM-DD
  criadoEm: z.string(),
});
export type Preco = z.infer<typeof Preco>;

// GET /api/admin/modelos/{id}/precos (mais recentes primeiro)
export const RespostaPrecos = z.object({ precos: z.array(Preco) });

// POST /api/admin/modelos/{id}/precos -> 201 Preco. Erros: 409 preco_repetido (mesma unidade e vigência).
export const PedidoNovoPreco = z.object({
  unidade: UnidadePreco,
  precoUsd: z.number().min(0),
  vigenciaInicio: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Use AAAA-MM-DD."),
});
export type PedidoNovoPreco = z.infer<typeof PedidoNovoPreco>;

// ---- Testar e registros ----

export const ModeloResumo = z.object({
  id: z.string(),
  identificador: z.string(),
  fornecedorNome: z.string(),
  simulado: z.boolean(),
  ativo: z.boolean(),
});
export type ModeloResumo = z.infer<typeof ModeloResumo>;

// POST /api/admin/modelos/{id}/testar (sem corpo) -> 200
// Texto envia "Oi, qual modelo é você?"; imagem pede "um círculo azul sobre fundo branco".
// Erros: 409 sem_preco | limite_atingido | modelo_inativo; 502 fornecedor_* (mensagem clara, sem a chave).
export const ResultadoTeste = z.object({
  execucaoId: z.string(),
  servico: Servico,
  modelo: ModeloResumo,
  texto: z.string().nullable(),
  imagemDataUrl: z.string().nullable(), // data:image/png;base64,... (não é guardada)
  tokensEntrada: z.number().int(),
  tokensSaida: z.number().int(),
  imagens: z.number().int(),
  custoUsd: z.number(),
  simulado: z.boolean(),
});
export type ResultadoTeste = z.infer<typeof ResultadoTeste>;

export const Execucao = z.object({
  id: z.string(),
  correlacaoId: z.string(),
  criadoEm: z.string(),
  atividade: z.string(), // Atividade ou "teste"
  organizacaoNome: z.string().nullable(),
  fornecedorNome: z.string(),
  modeloIdentificador: z.string(),
  tentativa: z.number().int(),
  atendidoPelaReserva: z.boolean(),
  status: z.enum(["ok", "falha", "pulado_limite"]),
  erro: z.string().nullable(),
  tokensEntrada: z.number().int(),
  tokensSaida: z.number().int(),
  imagens: z.number().int(),
  custoUsd: z.number(),
  custoEstimado: z.boolean(),
  simulado: z.boolean(),
});
export type Execucao = z.infer<typeof Execucao>;

// GET /api/admin/execucoes?limite=50 (mais recentes primeiro; limite máx. 200)
export const RespostaExecucoes = z.object({ execucoes: z.array(Execucao) });

// ---- Atividades (principal e reserva) ----

export const RotaAtividade = z.object({
  atividade: Atividade,
  nome: z.string(),
  servico: Servico,
  principal: ModeloResumo.nullable(),
  reserva: ModeloResumo.nullable(),
  pronta: z.boolean(),
  falta: z.array(z.string()), // ex.: ["modelo principal", "modelo reserva", "preço do modelo reserva"]
});
export type RotaAtividade = z.infer<typeof RotaAtividade>;

// GET /api/admin/atividades
export const RespostaAtividades = z.object({ atividades: z.array(RotaAtividade) });

// PUT /api/admin/atividades/{atividade} -> 200 RotaAtividade
// Erros: 400 servico_incompativel (modelo de imagem em atividade de texto e vice-versa) | modelos_iguais.
export const PedidoRotaAtividade = z.object({
  modeloPrincipalId: z.string().nullable(),
  modeloReservaId: z.string().nullable(),
});
export type PedidoRotaAtividade = z.infer<typeof PedidoRotaAtividade>;

// POST /api/admin/atividades/{atividade}/testar (sem corpo) -> 200
// Roda a atividade como uma atividade de verdade: principal, e reserva se o principal falhar ou estiver no limite.
// Erros: 409 atividade_incompleta (mensagem diz o que falta, AC36);
//        409 limite_atingido (principal e reserva no limite, AC34);
//        502 atividade_falhou (principal e reserva falharam; registros gravados).
export const Tentativa = z.object({
  modelo: ModeloResumo,
  status: z.enum(["ok", "falha", "pulado_limite"]),
  erro: z.string().nullable(),
});

export const ResultadoAtividade = z.object({
  correlacaoId: z.string(),
  atividade: Atividade,
  atendidoPor: ModeloResumo,
  atendidoPelaReserva: z.boolean(),
  tentativas: z.array(Tentativa),
  texto: z.string().nullable(),
  imagemDataUrl: z.string().nullable(),
  tokensEntrada: z.number().int(),
  tokensSaida: z.number().int(),
  imagens: z.number().int(),
  custoUsd: z.number(),
  simulado: z.boolean(),
});
export type ResultadoAtividade = z.infer<typeof ResultadoAtividade>;

// ---- Avisos de limite (AC34) ----

export const AvisoLimite = z.object({
  alvo: z.enum(["fornecedor", "modelo"]),
  id: z.string(),
  nome: z.string(),
  situacao: z.enum(["aviso", "limite"]),
  gastoMesUsd: z.number(),
  limiteMensalUsd: z.number(),
  avisoPercentual: z.number().int(),
});
export type AvisoLimite = z.infer<typeof AvisoLimite>;

// GET /api/admin/avisos
export const RespostaAvisos = z.object({ avisos: z.array(AvisoLimite) });
