import type { EstadoPost, Etapa, Formato, Objetivo, Organizacao, Post, Rede } from "./types";

export const ETAPAS: { n: Etapa; nome: string }[] = [
  { n: 1, nome: "Pauta" },
  { n: 2, nome: "Texto" },
  { n: 3, nome: "Visual" },
  { n: 4, nome: "Revisão" },
  { n: 5, nome: "Aprovação final" },
  { n: 6, nome: "Publicação" },
];

export const REDES: { id: Rede; nome: string; limite: number; tamanho: string }[] = [
  { id: "instagram", nome: "Instagram", limite: 2200, tamanho: "1080×1350" },
  { id: "facebook", nome: "Facebook", limite: 63206, tamanho: "1200×1200" },
  { id: "linkedin", nome: "LinkedIn", limite: 3000, tamanho: "1200×1200" },
];
export const nomeRede = (r: Rede) => REDES.find((x) => x.id === r)!.nome;
export const OBJETIVOS: { id: Objetivo; nome: string; explica: string }[] = [
  { id: "atrair", nome: "Atrair", explica: "Alcançar quem ainda não conhece a marca" },
  { id: "engajar", nome: "Engajar", explica: "Criar conversa com quem já acompanha" },
  { id: "converter", nome: "Converter", explica: "Levar a pessoa a dar um passo (contato, visita)" },
];
export const nomeObjetivo = (o: Objetivo) => OBJETIVOS.find((x) => x.id === o)?.nome ?? "—";
export const nomeFormato = (f: Formato) => (f === "imagem" ? "Imagem com legenda" : "Carrossel");

export const ESTADOS: Record<EstadoPost, { rotulo: string; tom: string }> = {
  em_criacao: { rotulo: "Em criação", tom: "info" },
  aguardando_aprovacao: { rotulo: "Aguardando aprovação", tom: "ai" },
  agendado: { rotulo: "Agendado", tom: "brand" },
  pronto_manual: { rotulo: "Pronto para publicação manual", tom: "success" },
  publicado: { rotulo: "Publicado", tom: "success" },
  falhou: { rotulo: "Falhou", tom: "danger" },
  atrasado: { rotulo: "Atrasado", tom: "warning" },
  descartado: { rotulo: "Descartado", tom: "neutral" },
};

export const hoje = () => new Date().toISOString().slice(0, 10);

export function estadoDoPost(p: Post, org?: Organizacao): EstadoPost {
  if (p.descartado) return "descartado";
  if (p.publicacao === "publicado") return "publicado";
  if (p.publicacao === "falhou") return "falhou";
  if (p.etapa < 6) {
    if (p.data < hoje()) return "atrasado"; // APR-03
    return p.etapa === 5 ? "aguardando_aprovacao" : "em_criacao";
  }
  const conectada = org && p.pauta.redes.some((r) => org.conexoes[r] === "ativa");
  return conectada ? "agendado" : "pronto_manual";
}

export const fmtData = (iso: string) => {
  const [a, m, d] = iso.slice(0, 10).split("-");
  return `${d}/${m}/${a}`;
};
export const fmtHora = (iso: string) =>
  new Date(iso).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
export const fmtUsd = (n: number) => `US$ ${n.toFixed(4).replace(".", ",")}`;
export const fmtNum = (n: number) => n.toLocaleString("pt-BR");

const isoLocal = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

/** Os 7 dias (segunda a domingo) da semana atual (0) ou das seguintes (1, 2...). */
export function diasDaSemana(deslocamento = 0): string[] {
  const d = new Date();
  const ini = new Date(d.getFullYear(), d.getMonth(), d.getDate() - ((d.getDay() + 6) % 7) + deslocamento * 7);
  return Array.from({ length: 7 }, (_, i) => isoLocal(new Date(ini.getFullYear(), ini.getMonth(), ini.getDate() + i)));
}

export function semanaAtual(): [string, string] {
  const d = new Date();
  const dia = (d.getDay() + 6) % 7; // segunda = 0
  const ini = new Date(d);
  ini.setDate(d.getDate() - dia);
  const fim = new Date(ini);
  fim.setDate(ini.getDate() + 6);
  return [ini.toISOString().slice(0, 10), fim.toISOString().slice(0, 10)];
}

/** Redes abaixo da cadência na semana atual (CAL-03). */
export function faltasDeCadencia(org: Organizacao, posts: Post[]) {
  const [ini, fim] = semanaAtual();
  const daSemana = posts.filter((p) => p.orgId === org.id && !p.descartado && p.data >= ini && p.data <= fim);
  return REDES.map((r) => {
    const tem = daSemana.filter((p) => p.pauta.redes.includes(r.id)).length;
    return { rede: r.id, tem, meta: org.cadencia[r.id] };
  }).filter((x) => x.tem < x.meta);
}
