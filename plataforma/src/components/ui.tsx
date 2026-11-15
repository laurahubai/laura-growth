"use client";
import type { EstadoPost, Organizacao, Slide } from "@/mock/types";
import { ESTADOS } from "@/mock/regras";
import { Sparkles } from "lucide-react";

export function StatusBadge({ estado }: { estado: EstadoPost }) {
  const e = ESTADOS[estado];
  return <span className={`badge t-${e.tom}`}>{e.rotulo}</span>;
}

export function PageHeader({ titulo, descricao, acoes }: { titulo: string; descricao?: string; acoes?: React.ReactNode }) {
  return (
    <div className="page-header">
      <div>
        <h1>{titulo}</h1>
        {descricao && <p>{descricao}</p>}
      </div>
      {acoes && <div className="row">{acoes}</div>}
    </div>
  );
}

export const TagIA = ({ texto = "Gerado por IA" }: { texto?: string }) => (
  <span className="ai-tag">
    <Sparkles size={12} aria-hidden /> {texto}
  </span>
);

/** Arte de EXEMPLO desenhada com as cores do kit (no lugar da imagem gerada por IA). */
export function SlideArte({ slide, org, i, total }: { slide: Slide; org: Organizacao; i: number; total: number }) {
  const [c1, c2] = org.kit.cores;
  const fundo = slide.imagem
    ? { backgroundImage: `url(${slide.imagem})` }
    : { background: i % 2 ? c2 : `linear-gradient(160deg, ${c1}, ${c2})` };
  return (
    <div className="slide" style={fundo} role="img" aria-label={`Arte ${i + 1} de ${total}: ${slide.titulo}`}>
      <small>{org.nome}</small>
      <span style={{ textShadow: "0 1px 2px rgb(0 0 0 / .3)" }}>{slide.imagem ? "" : slide.titulo}</span>
      <small>
        {total > 1 ? `${i + 1}/${total} · ` : ""}
        {slide.imagem ? "Imagem enviada por você" : "Arte de exemplo"}
      </small>
    </div>
  );
}
