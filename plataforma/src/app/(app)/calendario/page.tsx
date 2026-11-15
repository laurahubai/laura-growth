"use client";
import Link from "next/link";
import { useState } from "react";
import { AlertTriangle, ChevronLeft, ChevronRight } from "lucide-react";
import { useLoja, usePostsDaOrg } from "@/mock/loja";
import { ESTADOS, estadoDoPost, faltasDeCadencia, hoje, nomeRede } from "@/mock/regras";
import { PageHeader } from "@/components/ui";

const DIAS = ["Seg", "Ter", "Qua", "Qui", "Sex", "Sáb", "Dom"];

export default function Calendario() {
  const { org } = useLoja();
  const posts = usePostsDaOrg();
  const [ref, setRef] = useState(() => { const d = new Date(); return new Date(d.getFullYear(), d.getMonth(), 1); });
  if (!org) return null;

  const inicio = new Date(ref);
  inicio.setDate(1 - ((ref.getDay() + 6) % 7));
  const dias = Array.from({ length: 42 }, (_, i) => { const d = new Date(inicio); d.setDate(inicio.getDate() + i); return d; });
  const iso = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const mover = (n: number) => setRef(new Date(ref.getFullYear(), ref.getMonth() + n, 1));
  const faltas = faltasDeCadencia(org, posts);

  return (
    <>
      <PageHeader
        titulo="Calendário"
        descricao={`Posts da ${org.nome} e suas situações.`}
        acoes={
          <>
            <button className="btn peq" onClick={() => mover(-1)} aria-label="Mês anterior"><ChevronLeft size={16} /></button>
            <b style={{ minWidth: 140, textAlign: "center" }}>{ref.toLocaleDateString("pt-BR", { month: "long", year: "numeric" })}</b>
            <button className="btn peq" onClick={() => mover(1)} aria-label="Próximo mês"><ChevronRight size={16} /></button>
            <Link href="/posts?novo=1" className="btn primario">Novo post</Link>
          </>
        }
      />
      {faltas.length > 0 && (
        <div className="alerta t-warning" style={{ marginBottom: 12 }}>
          <AlertTriangle size={16} aria-hidden />
          <span>Abaixo da cadência nesta semana: {faltas.map((f) => `${nomeRede(f.rede)} (${f.tem} de ${f.meta})`).join(" · ")}.</span>
        </div>
      )}
      <div className="rolagem">
        <div className="cal" style={{ minWidth: 720 }}>
          {DIAS.map((d) => <div key={d} className="cal-cab">{d}</div>)}
          {dias.map((d) => {
            const k = iso(d);
            const doDia = posts.filter((p) => p.data === k && !p.descartado);
            return (
              <div key={k} className={`cal-dia ${d.getMonth() !== ref.getMonth() ? "fora" : ""} ${k === hoje() ? "hoje" : ""}`}>
                <span className="cal-n">{d.getDate()}</span>
                {doDia.map((p) => {
                  const e = ESTADOS[estadoDoPost(p, org)];
                  return (
                    <Link key={p.id} href={`/posts/${p.id}`} className={`cal-post t-${e.tom}`} title={`${p.pauta.tema} — ${e.rotulo}`}>
                      {e.rotulo}: {p.pauta.tema}
                    </Link>
                  );
                })}
              </div>
            );
          })}
        </div>
      </div>
    </>
  );
}
