// Dados de EXEMPLO (fictícios) para homologar a jornada.
import type { Dados, Etapa, Organizacao, Post, RegistroCusto } from "./types";
import { gerarTexto, gerarVisual } from "./agente-falso";

export const VERSAO_DADOS = 2;

const dia = (delta: number) => {
  const d = new Date();
  d.setDate(d.getDate() + delta);
  return d.toISOString().slice(0, 10);
};
const agoraMenos = (h: number) => new Date(Date.now() - h * 3600_000).toISOString();

const orgs: Organizacao[] = [
  {
    id: "laura-growth",
    nome: "Laura Growth",
    tipo: "empresa",
    kit: {
      cores: ["#f26b3a", "#1f2430", "#fff3ed"],
      fontes: "Inter",
      tomDeVoz: "Próximo, direto e confiante. Sem exageros.",
      temas: "marketing com IA, constância, bastidores, produtividade",
      pilares: ["marketing com IA", "constância de publicação", "bastidores da Laura", "rotina do marketing de clínicas"],
      site: "(exemplo) site da Laura Growth",
      perfis: "(exemplo) @lauragrowth",
      publico: "Donos e responsáveis de marketing de clínicas de estética de porte médio",
      nuncaDizer: "mágica, 100% automático, garantido",
      exemplos: "(exemplo) Post sobre os bastidores da criação de conteúdo com IA",
    },
    cadencia: { instagram: 3, facebook: 2, linkedin: 2 },
    pilotoAutomatico: true,
    conexoes: { instagram: "nao_conectada", facebook: "nao_conectada", linkedin: "nao_conectada" },
  },
  {
    id: "laura-hub",
    nome: "Laura Hub",
    tipo: "empresa",
    kit: {
      cores: ["#1570ef", "#182230", "#eff8ff"],
      fontes: "Inter",
      tomDeVoz: "Acolhedor, profissional e organizado.",
      temas: "gestão de clínica, agenda, atendimento, estética",
      pilares: ["gestão da clínica", "agenda e atendimento", "histórias de clínicas", "novidades do produto"],
      site: "(exemplo) site da Laura Hub",
      perfis: "(exemplo) @laurahub",
      publico: "Gestores de clínicas de estética",
      nuncaDizer: "milagre, resultado garantido",
      exemplos: "(exemplo) Dica de organização da agenda da clínica",
    },
    cadencia: { instagram: 3, facebook: 1, linkedin: 1 },
    pilotoAutomatico: true,
    conexoes: { instagram: "nao_conectada", facebook: "nao_conectada", linkedin: "nao_conectada" },
  },
];

let seq = 0;
const id = (p: string) => `${p}-${++seq}`;

function montar(org: Organizacao, tema: string, etapa: Etapa, data: number, extra: Partial<Post> = {}): Post {
  const pauta = { tema, pilar: org.kit.pilares[seq % org.kit.pilares.length], objetivo: (["atrair", "engajar", "converter"] as const)[seq % 3], redes: ["instagram", "linkedin"] as Post["pauta"]["redes"], formato: (seq % 2 ? "carrossel" : "imagem") as "carrossel" | "imagem" };
  const texto = etapa >= 2 ? gerarTexto(org, pauta) : undefined;
  const visual = etapa >= 3 ? gerarVisual(pauta) : undefined;
  return {
    id: id("post"),
    orgId: org.id,
    origem: "sugestao",
    etapa,
    data: dia(data),
    descartado: false,
    piloto: true,
    pauta,
    texto,
    visual,
    revisao: etapa >= 4 ? { avisos: ["Nenhum problema encontrado em marca, ortografia e formato."] } : undefined,
    versoes: [],
    historico: [{ em: agoraMenos(30), quem: "Strategy Agent", acao: "Sugeriu a pauta" }],
    atividade: [],
    trabalhando: false,
    ...extra,
  };
}

function custosDe(post: Post): RegistroCusto[] {
  const out: RegistroCusto[] = [];
  const ate = Math.min(post.etapa, 4);
  for (let e = 1; e <= ate; e++) {
    const img = e === 3;
    const q = img ? (post.pauta.formato === "carrossel" ? 4 : 1) : 1200 + ((seq * 337 + e * 211) % 1500);
    const preco = img ? 0.04 : 0.000009;
    out.push({
      id: id("custo"),
      orgId: post.orgId,
      postId: post.id,
      agente: e === 1 ? "Strategy Agent" : e === 4 ? "Social Agent" : "Creative Agent",
      etapa: e as Etapa,
      fornecedor: img ? "Fornecedor de imagem (exemplo)" : "Fornecedor de texto (exemplo)",
      unidade: img ? "imagem" : "tokens",
      quantidade: q,
      precoUnitarioUsd: preco,
      custoUsd: q * preco,
      em: agoraMenos(24 - e),
    });
  }
  return out;
}

export function dadosDeExemplo(): Dados {
  seq = 0;
  const [lg, lh] = orgs.map((o) => structuredClone(o));
  const posts: Post[] = [
    montar(lg, "3 sinais de que seu marketing perdeu a cadência", 6, -3, { publicacao: "publicado", aprovadoPor: "Você" }),
    montar(lg, "Como planejar um mês de posts em 1 hora", 5, 1),
    montar(lg, "O que muda quando a marca publica toda semana", 5, 2),
    montar(lg, "Bastidores: como a Laura revisa um post", 5, 4),
    montar(lg, "Bastidores: um post do pedido à aprovação", 2, 2, { origem: "pedido" }),
    montar(lg, "Erros comuns ao terceirizar o marketing", 3, -1),
    montar(lg, "Ideia descartada: tendências do ano", 2, 0, { descartado: true }),
    montar(lh, "Organize a agenda da clínica em 3 passos", 6, 2, { aprovadoPor: "Você" }),
    montar(lh, "Como reduzir faltas nos atendimentos", 4, 3),
  ];
  return {
    versao: VERSAO_DADOS,
    orgs: [lg, lh],
    posts,
    custos: posts.flatMap(custosDe),
  };
}
