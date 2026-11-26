// Etapa 2B: acessos por pessoa e por alçada (PRM-01 a PRM-05).
// Espelha backend/alcadas.go, backend/pessoas.go e backend/admin_equipe.go: ao mudar um, mude o outro.
//
// Rotas (S1):
// - GET    /api/pessoas                       -> RespostaPessoas (exige gerenciar_pessoas)
// - PUT    /api/pessoas/{usuarioId}/alcadas   -> PessoaMembro (exige gerenciar_pessoas; dono e autoedição -> 403)
// - DELETE /api/pessoas/{usuarioId}           -> 204 (exige gerenciar_pessoas; dono e autorremoção -> 403)
// - GET    /api/pessoas/registro?limite=100   -> RespostaRegistroAcessos (exige gerenciar_pessoas)
// - GET    /api/admin/equipe                  -> RespostaEquipeAdmin (exige usuario.adminPlataforma = true)
// - PUT    /api/admin/equipe/{usuarioId}      -> MembroEquipe (exige usuario.adminPlataforma = true; só membros da casa; último admin -> 409)
//
// Conjuntos prontos (B7):
// São retornados pelo servidor no campo `conjuntos` de GET /api/pessoas (e espelhados aqui
// em CONJUNTOS_ALCADAS). Ao aplicar um conjunto, a tela apenas pré-preenche as caixas;
// o salvamento envia apenas `alcadas` individuais para PUT /api/pessoas/{usuarioId}/alcadas.
import { z } from "zod";

export const ALCADAS = [
  "ver_marca_posts",
  "cmo_dna",
  "aprovar_plano",
  "editar_posts",
  "aprovar_posts",
  "conectar_redes",
  "ver_creditos",
  "gerenciar_pessoas",
] as const;

export const Alcada = z.enum(ALCADAS);
export type Alcada = z.infer<typeof Alcada>;

export const rotuloAlcada: Record<Alcada, string> = {
  ver_marca_posts: "Ver a marca e os posts",
  cmo_dna: "Conversar com o CMO e editar o DNA da marca",
  aprovar_plano: "Aprovar o plano de marketing",
  editar_posts: "Criar e editar posts",
  aprovar_posts: "Aprovar posts para publicação",
  conectar_redes: "Conectar e desconectar redes",
  ver_creditos: "Ver consumo em créditos",
  gerenciar_pessoas: "Convidar pessoas e dar acessos",
};

export const CodigoConjuntoAlcadas = z.enum([
  "somente_leitura",
  "criador_conteudo",
  "aprovador",
  "gestor_marketing",
]);
export type CodigoConjuntoAlcadas = z.infer<typeof CodigoConjuntoAlcadas>;

export const ConjuntoAlcadas = z.object({
  codigo: CodigoConjuntoAlcadas,
  nome: z.string(),
  alcadas: z.array(Alcada),
});
export type ConjuntoAlcadas = z.infer<typeof ConjuntoAlcadas>;

export const CONJUNTOS_ALCADAS: readonly ConjuntoAlcadas[] = [
  {
    codigo: "somente_leitura",
    nome: "Somente leitura",
    alcadas: ["ver_marca_posts", "ver_creditos"],
  },
  {
    codigo: "criador_conteudo",
    nome: "Criador de conteúdo",
    alcadas: ["ver_marca_posts", "cmo_dna", "editar_posts"],
  },
  {
    codigo: "aprovador",
    nome: "Aprovador",
    alcadas: ["ver_marca_posts", "aprovar_plano", "aprovar_posts"],
  },
  {
    codigo: "gestor_marketing",
    nome: "Gestor de marketing",
    alcadas: [
      "ver_marca_posts",
      "cmo_dna",
      "aprovar_plano",
      "editar_posts",
      "aprovar_posts",
      "conectar_redes",
      "ver_creditos",
    ],
  },
];

export const PapelOrganizacao = z.enum(["dono", "membro"]);
export type PapelOrganizacao = z.infer<typeof PapelOrganizacao>;

export const PessoaMembro = z.object({
  usuarioId: z.string(),
  nome: z.string(),
  email: z.string(),
  papel: PapelOrganizacao,
  alcadas: z.array(Alcada),
  criadoEm: z.string(),
});
export type PessoaMembro = z.infer<typeof PessoaMembro>;

export const EstadoEnvioConvite = z.enum(["enviado", "falhou", "pendente"]);
export type EstadoEnvioConvite = z.infer<typeof EstadoEnvioConvite>;

export const ConvitePendente = z.object({
  id: z.string(),
  email: z.string(),
  alcadas: z.array(Alcada),
  envioEstado: EstadoEnvioConvite,
  criadoPorNome: z.string().nullable(),
  criadoEm: z.string(),
  expiraEm: z.string(),
});
export type ConvitePendente = z.infer<typeof ConvitePendente>;

// GET /api/pessoas
export const RespostaPessoas = z.object({
  membros: z.array(PessoaMembro),
  convites: z.array(ConvitePendente),
  conjuntos: z.array(ConjuntoAlcadas),
});
export type RespostaPessoas = z.infer<typeof RespostaPessoas>;

// PUT /api/pessoas/{usuarioId}/alcadas -> 200 PessoaMembro
export const PedidoSalvarAlcadas = z.object({
  alcadas: z.array(Alcada),
});
export type PedidoSalvarAlcadas = z.infer<typeof PedidoSalvarAlcadas>;

// Ações gravadas em registro_acessos:
// - Org: "alcada_concedida", "alcada_revogada", "membro_removido",
//        (e na S2: "convite_criado", "convite_reenviado", "convite_revogado", "convite_aceito")
// - Plataforma: "admin_plataforma_concedido", "admin_plataforma_revogado"
export const RegistroAcesso = z.object({
  id: z.string(),
  acao: z.string(),
  atorId: z.string().nullable(),
  atorNome: z.string().nullable(),
  atorEmail: z.string().nullable(),
  alvoUsuarioId: z.string().nullable(),
  alvoNome: z.string().nullable(),
  alvoEmail: z.string().nullable(),
  detalhe: z.record(z.string(), z.unknown()),
  em: z.string(),
});
export type RegistroAcesso = z.infer<typeof RegistroAcesso>;

// GET /api/pessoas/registro
export const RespostaRegistroAcessos = z.object({
  registros: z.array(RegistroAcesso),
});
export type RespostaRegistroAcessos = z.infer<typeof RespostaRegistroAcessos>;

// ---- Administração → Equipe (B8) ----

export const MembroEquipe = z.object({
  usuarioId: z.string(),
  nome: z.string(),
  email: z.string(),
  adminPlataforma: z.boolean(),
  organizacoes: z.array(z.string()),
});
export type MembroEquipe = z.infer<typeof MembroEquipe>;

// GET /api/admin/equipe
export const RespostaEquipeAdmin = z.object({
  equipe: z.array(MembroEquipe),
  registros: z.array(RegistroAcesso),
});
export type RespostaEquipeAdmin = z.infer<typeof RespostaEquipeAdmin>;

// PUT /api/admin/equipe/{usuarioId} -> 200 MembroEquipe
export const PedidoAtualizarEquipeAdmin = z.object({
  adminPlataforma: z.boolean(),
});
export type PedidoAtualizarEquipeAdmin = z.infer<typeof PedidoAtualizarEquipeAdmin>;
