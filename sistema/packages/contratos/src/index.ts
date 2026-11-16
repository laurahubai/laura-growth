// Formato dos dados trocados entre as telas e o servidor.
// Espelha os tipos de backend/api.go: ao mudar um, mude o outro.
import { z } from "zod";

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
});
export type Usuario = z.infer<typeof Usuario>;

export const Sessao = z.object({
  usuario: Usuario,
  organizacoes: z.array(Organizacao),
  organizacaoAtiva: Organizacao,
});
export type Sessao = z.infer<typeof Sessao>;

export const OrganizacaoDetalhe = Organizacao.extend({
  criadaEm: z.string(),
  papel: z.enum(["dono", "membro"]),
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
