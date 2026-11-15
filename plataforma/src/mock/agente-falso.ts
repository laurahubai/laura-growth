// Simulador do agente de IA. Gera conteúdo de EXEMPLO a partir do kit de marca.
// No backend real, isto será trocado pela chamada aos fornecedores de IA (a definir).
import type { Agente, Etapa, Organizacao, Pauta, Post, Revisao, Texto, Visual } from "./types";
import { REDES } from "./regras";

const PAUTAS_SUGERIDAS = [
  { tema: "3 sinais de que seu marketing perdeu a cadência", objetivo: "Gerar identificação e salvamentos" },
  { tema: "Bastidores: como um post nasce com IA e aprovação humana", objetivo: "Mostrar transparência" },
  { tema: "Checklist rápido para planejar o mês de conteúdo", objetivo: "Educar e gerar compartilhamentos" },
  { tema: "Mito ou verdade: IA deixa os posts sem personalidade?", objetivo: "Engajar nos comentários" },
];

let giro = 0;
export function sugerirPauta(org: Organizacao): Pauta {
  const s = PAUTAS_SUGERIDAS[giro++ % PAUTAS_SUGERIDAS.length];
  return { tema: s.tema, objetivo: s.objetivo, redes: ["instagram", "linkedin"], formato: giro % 2 ? "carrossel" : "imagem" };
}

export function gerarTexto(org: Organizacao, pauta: Pauta, comentario?: string, anterior?: Texto): Texto {
  const tom = org.kit.tomDeVoz.split(/[,.]/)[0]?.trim().toLowerCase() || "próximo";
  let legenda =
    `${pauta.tema}\n\n` +
    `Na ${org.nome}, a gente acredita em marketing ${tom}: constância, clareza e uma pessoa sempre aprovando o que vai ao ar.\n\n` +
    `👉 ${pauta.objetivo}. Salve este post e conte nos comentários como é na sua rotina.`;
  if (comentario && anterior) {
    legenda = `${anterior.legenda}\n\n[Ajuste aplicado: ${comentario}]`;
    if (/curt|menor|resum/i.test(comentario)) legenda = anterior.legenda.split("\n\n").slice(0, 2).join("\n\n");
  }
  const tags = org.kit.temas
    .split(/[,;]/)
    .map((t) => "#" + t.trim().toLowerCase().replace(/\s+/g, ""))
    .filter((t) => t.length > 1)
    .slice(0, 4);
  return { legenda, hashtags: [...tags, "#marketingdigital"].join(" ") };
}

export function gerarVisual(pauta: Pauta, anterior?: Visual): Visual {
  const n = pauta.formato === "carrossel" ? 4 : 1;
  const base = ["Capa", "O problema", "O que fazer", "Salve para depois"];
  return {
    slides: Array.from({ length: n }, (_, i) => ({
      titulo: i === 0 ? pauta.tema : base[i] + (anterior ? " (nova versão)" : ""),
    })),
  };
}

export function revisar(org: Organizacao, post: Post): Revisao {
  const avisos: string[] = [];
  const leg = post.texto?.legenda ?? "";
  for (const r of REDES) {
    if (post.pauta.redes.includes(r.id) && leg.length > r.limite) avisos.push(`Texto acima do limite do ${r.nome}.`);
  }
  const proibidas = org.kit.nuncaDizer.split(/[,;]/).map((s) => s.trim().toLowerCase()).filter(Boolean);
  for (const p of proibidas) if (leg.toLowerCase().includes(p)) avisos.push(`O texto usa "${p}", que está na lista "nunca dizer" do kit.`);
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
      return { agente: "Strategy Agent", passos: [...pre, "Lendo o kit de marca", "Olhando o calendário e a cadência", "Escolhendo tema, objetivo, rede e formato"] };
    case 2:
      return { agente: "Creative Agent", passos: [...pre, "Lendo tom de voz e público", "Escrevendo a legenda", "Escolhendo hashtags", "Conferindo limites de cada rede"] };
    case 3:
      return { agente: "Creative Agent", passos: [...pre, "Aplicando cores e fontes do kit", "Montando a arte", "Adaptando o tamanho para cada rede"] };
    default:
      return { agente: "Social Agent", passos: [...pre, "Conferindo marca", "Conferindo ortografia", "Conferindo formato de cada rede"] };
  }
}

/** Consumo fictício por tarefa (fornecedor e preços: a definir). */
export function consumoFicticio(etapa: Etapa, slides = 1) {
  if (etapa === 3)
    return { fornecedor: "Fornecedor de imagem (exemplo)", unidade: "imagem" as const, quantidade: slides, preco: 0.04 };
  const tokens = 900 + Math.round(Math.random() * 1600);
  return { fornecedor: "Fornecedor de texto (exemplo)", unidade: "tokens" as const, quantidade: tokens, preco: 0.000009 };
}
