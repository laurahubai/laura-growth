// Formato dos dados trocados entre as telas e o servidor.
// Espelha os tipos de backend/api.go: ao mudar um, mude o outro.
import { z } from "zod";
import { Alcada, PapelOrganizacao } from "./acessos";

export const TipoOrganizacao = z.enum(["empresa", "clinica_estetica"]);
export type TipoOrganizacao = z.infer<typeof TipoOrganizacao>;

export const Organizacao = z.object({
  id: z.string(),
  nome: z.string(),
  tipo: TipoOrganizacao,
});
export type Organizacao = z.infer<typeof Organizacao>;

export const Usuario = z.object({
  id: z.string(),
  nome: z.string(),
  email: z.string(),
  // Etapa 2A-1: só o administrador da plataforma vê a Administração (PRD 2.0 §3).
  adminPlataforma: z.boolean(),
});
export type Usuario = z.infer<typeof Usuario>;

export const Sessao = z.object({
  usuario: Usuario,
  organizacoes: z.array(Organizacao),
  organizacaoAtiva: Organizacao,
  alcadas: z.array(Alcada),
});
export type Sessao = z.infer<typeof Sessao>;

export const OrganizacaoDetalhe = Organizacao.extend({
  criadaEm: z.string(),
  papel: PapelOrganizacao,
  membros: z.number(),
});
export type OrganizacaoDetalhe = z.infer<typeof OrganizacaoDetalhe>;

export const ErroApi = z.object({ codigo: z.string(), mensagem: z.string() });
export type ErroApi = z.infer<typeof ErroApi>;

export const PedidoEntrar = z.object({
  email: z.string().trim().email("Confira o e-mail."),
  senha: z.string().min(1, "Informe a senha."),
});
export type PedidoEntrar = z.infer<typeof PedidoEntrar>;

export const nomeTipoOrganizacao: Record<TipoOrganizacao, string> = {
  empresa: "Empresa",
  clinica_estetica: "Clínica de estética",
};

// ---- Etapa 2: kit de marca ----

export const LIMITES_KIT = { pilaresMin: 3, pilaresMax: 5, coresMax: 5, textoMax: 5000, pilarMax: 500, arquivoMaxMB: 8, produtosMax: 20, produtoNomeMax: 500 } as const;

// Etapa 2C: códigos estáveis; rótulos em PT-BR nas telas.
export const ModeloNegocio = z.enum(["", "b2b", "b2c", "b2b2c", "outro"]);
export type ModeloNegocio = z.infer<typeof ModeloNegocio>;
export const nomeModeloNegocio: Record<Exclude<ModeloNegocio, "">, string> = {
  b2b: "B2B (vende para empresas)",
  b2c: "B2C (vende para consumidores)",
  b2b2c: "B2B2C",
  outro: "Outro",
};

export const CanalKit = z.enum([
  "instagram", "facebook", "linkedin", "tiktok", "youtube",
  "google_meu_negocio", "site_blog", "whatsapp", "email",
]);
export type CanalKit = z.infer<typeof CanalKit>;
export const nomeCanalKit: Record<CanalKit, string> = {
  instagram: "Instagram",
  facebook: "Facebook",
  linkedin: "LinkedIn",
  tiktok: "TikTok",
  youtube: "YouTube",
  google_meu_negocio: "Google Meu Negócio",
  site_blog: "Site/blog",
  whatsapp: "WhatsApp",
  email: "E-mail",
};

export const ProdutoKit = z.object({ nome: z.string(), prioritario: z.boolean() });
export type ProdutoKit = z.infer<typeof ProdutoKit>;

// Campos do kit que têm origem (RF-10) e aparecem em "campos_alterados" das versões.
// Arquivos aparecem nas versões como "logo:<nome>" / "arquivos:<nome>" (sem origem).
export const CampoKit = z.enum([
  "site", "perfis", "tomDeVoz", "publico", "temas", "nuncaDizer", "fontes", "exemplos",
  "pilares", "cores", "produtos", "diferencial", "modeloNegocio", "regioes",
  "comoCompram", "maiorDesafio", "objetivo90Dias", "canais", "restricoesSetor",
]);
export type CampoKit = z.infer<typeof CampoKit>;
// modeloOutro segue a origem de modeloNegocio; canaisOutro segue a de canais.

export const OrigemCampo = z.object({
  origem: z.enum(["confirmado", "sugerido_ia"]),
  definidoPor: z.string().nullable(), // nome da pessoa; null quando sugerido pela IA
  definidoEm: z.string(),
});
export type OrigemCampo = z.infer<typeof OrigemCampo>;

export const KitMarca = z.object({
  organizacaoId: z.string(),
  site: z.string(),
  perfis: z.string(),
  tomDeVoz: z.string(),
  publico: z.string(),
  temas: z.string(),
  nuncaDizer: z.string(),
  fontes: z.string(),
  exemplos: z.string(),
  pilares: z.array(z.string()),
  cores: z.array(z.string()),
  // ---- Etapa 2C: DNA da marca (campos novos) ----
  produtos: z.array(ProdutoKit),
  diferencial: z.string(),
  modeloNegocio: ModeloNegocio,
  modeloOutro: z.string(),
  regioes: z.string(),
  comoCompram: z.string(),
  maiorDesafio: z.string(),
  objetivo90Dias: z.string(),
  canais: z.array(CanalKit),
  canaisOutro: z.string(),
  restricoesSetor: z.string(),
  atualizadoEm: z.string().nullable(),
});
export type KitMarca = z.infer<typeof KitMarca>;

export const ArquivoMarca = z.object({
  id: z.string(),
  tipo: z.enum(["logo", "referencia"]),
  nome: z.string(),
  mime: z.string(),
  tamanho: z.number(),
  criadoEm: z.string(),
});
export type ArquivoMarca = z.infer<typeof ArquivoMarca>;

export const RespostaKit = z.object({
  kit: KitMarca,
  arquivos: z.array(ArquivoMarca),
  // Etapa 2C: origem por campo (só campos com valor ou já marcados) e número da versão atual (0 = kit nunca salvo).
  origens: z.partialRecord(CampoKit, OrigemCampo),
  versao: z.number(),
});
export type RespostaKit = z.infer<typeof RespostaKit>;

// ---- Etapa 2C: gravação, confirmação e versões ----
// PUT /api/kit: corpo = campos editáveis do KitMarca + organizacaoId (trava da etapa 2: 409 "organizacao_trocada"
// se não for a organização ativa) + versaoBase; atualizadoEm é ignorado.
// Resposta 200 = RespostaKit. 409 codigo "kit_desatualizado" quando versaoBase != versão atual,
// mensagem "O kit foi alterado por outra pessoa. Recarregue para ver a versão atual."
// 400 codigo "dados_invalidos" para modelo/canal inválido ou tamanho excedido.
// POST /api/kit/confirmar: PedidoConfirmarKit -> 200 RespostaKit (mesmo 409).
export const PedidoConfirmarKit = z.object({ campos: z.array(CampoKit).min(1), versaoBase: z.number() });
export type PedidoConfirmarKit = z.infer<typeof PedidoConfirmarKit>;

export const ResumoVersaoKit = z.object({
  numero: z.number(),
  autorTipo: z.enum(["pessoa", "ia"]),
  autorNome: z.string().nullable(),
  camposAlterados: z.array(z.string()),
  criadoEm: z.string(),
});
export type ResumoVersaoKit = z.infer<typeof ResumoVersaoKit>;

// GET /api/kit/versoes?pagina=1 (20 por página, mais recente primeiro)
export const RespostaVersoesKit = z.object({
  versoes: z.array(ResumoVersaoKit),
  pagina: z.number(),
  total: z.number(),
});
export type RespostaVersoesKit = z.infer<typeof RespostaVersoesKit>;

// GET /api/kit/versoes/{numero}: versão antiga, somente leitura (404 se não existe na org ativa).
export const VersaoKit = ResumoVersaoKit.extend({
  dados: KitMarca.omit({ organizacaoId: true }),
  origens: z.partialRecord(CampoKit, OrigemCampo),
});
export type VersaoKit = z.infer<typeof VersaoKit>;

export * from "./admin";
export * from "./acessos";
