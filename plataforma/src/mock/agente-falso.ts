// Simulador do agente de IA. Gera conteúdo de EXEMPLO a partir do kit de marca.
// No backend real, isto será trocado pela chamada aos fornecedores de IA (a definir).
import type { Agente, Etapa, KitMarca, Objetivo, Organizacao, Pauta, Post, Rede, Revisao, Texto, Visual } from "./types";
import { diasDaSemana, hoje, REDES } from "./regras";

const MODELOS_DE_TEMA = [
  (p: string) => `3 erros comuns em ${p}`,
  (p: string) => `Bastidores: ${p} na prática`,
  (p: string) => `Checklist rápido de ${p}`,
  (p: string) => `Mito ou verdade sobre ${p}`,
  (p: string) => `Por onde começar em ${p}`,
];
// Proporção de exemplo: mais posts para atrair e engajar, menos para converter.
const MIX: Objetivo[] = ["atrair", "engajar", "atrair", "converter", "engajar"];
const CTA: Record<Objetivo, string> = {
  atrair: "Salve este post para consultar depois.",
  engajar: "Conte nos comentários como é na sua rotina.",
  converter: "Fale com a gente pelo link do perfil.",
};
export const ctaPadrao = (o: Objetivo) => CTA[o];

const pilaresDe = (org: Organizacao) => (org.kit.pilares.length ? org.kit.pilares : ["a marca"]);

let giro = 0;
export function sugerirPauta(org: Organizacao): Pauta {
  const i = giro++;
  const pilares = pilaresDe(org);
  const pilar = pilares[i % pilares.length];
  return {
    tema: MODELOS_DE_TEMA[i % MODELOS_DE_TEMA.length](pilar),
    pilar,
    objetivo: MIX[i % MIX.length],
    redes: ["instagram", "linkedin"],
    formato: i % 2 ? "imagem" : "carrossel",
  };
}

export interface ItemDoPlano {
  data: string;
  pauta: Pauta;
}

/** Plano semanal em lote (CAL-04): propõe só o que falta para cumprir a cadência da semana. */
export function planejarSemana(org: Organizacao, posts: Post[], deslocamento: number): ItemDoPlano[] {
  const dias = diasDaSemana(deslocamento);
  const daSemana = posts.filter((p) => p.orgId === org.id && !p.descartado && p.data >= dias[0] && p.data <= dias[6]);
  const falta = {} as Record<Rede, number>;
  for (const r of REDES) falta[r.id] = Math.max(0, org.cadencia[r.id] - daSemana.filter((p) => p.pauta.redes.includes(r.id)).length);

  const grupos: Rede[][] = [];
  while (REDES.some((r) => falta[r.id] > 0)) {
    const redes = REDES.filter((r) => falta[r.id] > 0).map((r) => r.id);
    redes.forEach((r) => falta[r]--);
    grupos.push(redes);
  }
  // Dias úteis ainda não passados, espalhando os posts pela semana.
  const livres = dias.slice(0, 5).filter((d) => d >= hoje());
  const uteis = livres.length ? livres : dias.filter((d) => d >= hoje());
  if (!uteis.length) return [];
  const pilares = pilaresDe(org);
  const jaUsados = daSemana.length;
  return grupos.map((redes, i) => {
    const k = jaUsados + i;
    const pilar = pilares[k % pilares.length];
    return {
      data: uteis[Math.floor((i * uteis.length) / grupos.length)],
      pauta: {
        tema: MODELOS_DE_TEMA[k % MODELOS_DE_TEMA.length](pilar),
        pilar,
        objetivo: MIX[k % MIX.length],
        redes,
        formato: k % 2 ? "imagem" : "carrossel",
      },
    };
  });
}

/** Onboarding assistido (MKT-05): rascunho de EXEMPLO do kit. No produto real, o agente lê o site e os perfis. */
export function rascunharKit(nome: string, tipo: Organizacao["tipo"], site: string, perfis: string): KitMarca {
  const clinica = tipo === "clinica_estetica";
  return {
    cores: ["#f26b3a", "#1f2430", "#fff3ed"],
    fontes: "Inter",
    tomDeVoz: clinica ? "Acolhedor, claro e responsável. Sem promessas de resultado." : "Próximo, direto e confiante. Sem exageros.",
    temas: clinica ? "cuidados com a pele, bem-estar, rotina da clínica" : "novidades, bastidores, dicas práticas",
    pilares: clinica
      ? ["cuidados com a pele", "bastidores da clínica", "dúvidas frequentes", "equipe e estrutura"]
      : ["dicas práticas", "bastidores", "histórias de clientes", "novidades da marca"],
    publico: clinica ? "(exemplo) Pessoas da região que buscam cuidados estéticos com segurança" : `(exemplo) Clientes e interessados na ${nome}`,
    nuncaDizer: clinica ? "milagre, resultado garantido, sem riscos" : "garantido, o melhor do mercado",
    exemplos: "(exemplo) O agente listaria aqui posts anteriores encontrados nos perfis informados",
    site,
    perfis,
  };
}

export const PASSOS_ONBOARDING = ["Lendo o site informado", "Lendo os perfis nas redes", "Identificando tom de voz e público", "Propondo pilares de conteúdo", "Montando o rascunho do kit"];
export const PASSOS_PLANO = ["Lendo o kit de marca e os pilares", "Olhando o calendário da semana", "Calculando o que falta para a cadência", "Distribuindo pilares, objetivos e datas"];

export function gerarTexto(org: Organizacao, pauta: Pauta, comentario?: string, anterior?: Texto): Texto {
  const tom = org.kit.tomDeVoz.split(/[,.]/)[0]?.trim().toLowerCase() || "próximo";
  let legenda =
    `${pauta.tema}\n\n` +
    `Na ${org.nome}, a gente acredita em comunicação ${tom}: constância, clareza e uma pessoa sempre aprovando o que vai ao ar.`;
  if (comentario && anterior) {
    legenda = `${anterior.legenda}\n\n[Ajuste aplicado: ${comentario}]`;
    if (/curt|menor|resum/i.test(comentario)) legenda = anterior.legenda.split("\n\n").slice(0, 2).join("\n\n");
  }
  const tags = [pauta.pilar, ...org.kit.temas.split(/[,;]/)]
    .map((t) => "#" + t.trim().toLowerCase().replace(/\s+/g, ""))
    .filter((t, i, a) => t.length > 1 && a.indexOf(t) === i)
    .slice(0, 4);
  return { legenda, cta: anterior?.cta || CTA[pauta.objetivo], hashtags: tags.join(" ") };
}

export function gerarVisual(pauta: Pauta, anterior?: Visual): Visual {
  const n = pauta.formato === "carrossel" ? 4 : 1;
  const base = ["Capa", "O problema", "O que fazer", "Salve para depois"];
  return {
    slides: Array.from({ length: n }, (_, i) => ({
      titulo: i === 0 ? pauta.tema : base[i] + (anterior ? " (nova versão)" : ""),
      imagem: anterior?.slides[i]?.imagem, // imagem enviada pela pessoa é preservada
    })),
  };
}

export function revisar(org: Organizacao, post: Post): Revisao {
  const avisos: string[] = [];
  const leg = `${post.texto?.legenda ?? ""}\n${post.texto?.cta ?? ""}`;
  for (const r of REDES) {
    if (post.pauta.redes.includes(r.id) && leg.length > r.limite) avisos.push(`Texto acima do limite do ${r.nome}.`);
  }
  const proibidas = org.kit.nuncaDizer.split(/[,;]/).map((s) => s.trim().toLowerCase()).filter(Boolean);
  for (const p of proibidas) if (leg.toLowerCase().includes(p)) avisos.push(`O texto usa "${p}", que está na lista "nunca dizer" do kit.`);
  if (!post.texto?.cta.trim()) avisos.push("O post está sem chamada para ação (CTA).");
  if (post.pauta.redes.includes("linkedin") && (post.texto?.hashtags.split(" ").length ?? 0) > 3)
    avisos.push("LinkedIn: recomendado usar até 3 hashtags (exemplo de regra).");
  if (!avisos.length) avisos.push("Nenhum problema encontrado em marca, ortografia e formato.");
  return { avisos };
}

/** Passos mostrados ao vivo enquanto o agente "trabalha". */
export function passosAoVivo(etapa: Etapa, ajuste: boolean): { agente: Agente; passos: string[] } {
  const pre = ajuste ? ["Lendo seu comentário de ajuste"] : [];
  switch (etapa) {
    case 1:
      return { agente: "Strategy Agent", passos: [...pre, "Lendo o kit de marca e os pilares", "Olhando o calendário e a cadência", "Escolhendo tema, pilar, objetivo, rede e formato"] };
    case 2:
      return { agente: "Creative Agent", passos: [...pre, "Lendo tom de voz e público", "Escrevendo a legenda", "Escrevendo a chamada para ação", "Escolhendo hashtags"] };
    case 3:
      return { agente: "Creative Agent", passos: [...pre, "Aplicando cores e fontes do kit", "Montando a arte", "Adaptando o tamanho para cada rede"] };
    default:
      return { agente: "Social Agent", passos: [...pre, "Conferindo marca", "Conferindo ortografia e CTA", "Conferindo formato de cada rede"] };
  }
}

/** Consumo fictício por tarefa (fornecedor e preços: a definir). */
export function consumoFicticio(etapa: Etapa, slides = 1, fornecedor?: string) {
  if (etapa === 3)
    return { fornecedor: fornecedor ?? "Fornecedor de imagem (exemplo)", unidade: "imagem" as const, quantidade: slides, preco: 0.04 };
  const tokens = 900 + Math.round(Math.random() * 1600);
  return { fornecedor: fornecedor ?? "Fornecedor de texto (exemplo)", unidade: "tokens" as const, quantidade: tokens, preco: 0.000009 };
}
