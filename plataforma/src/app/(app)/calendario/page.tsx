"use client";
import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AlertTriangle, ChevronLeft, ChevronRight, Sparkles, X } from "lucide-react";
import { PlanoSemanal } from "@/components/PlanoSemanal";
import { useLoja, usePostsDaOrg } from "@/mock/loja";
import { ESTADOS, estadoDoPost, faltasDeCadencia, hoje, nomeRede } from "@/mock/regras";
import { PageHeader } from "@/components/ui";

const DIAS = ["Seg", "Ter", "Qua", "Qui", "Sex", "Sáb", "Dom"];

function Mes() {
  const { org, criarPlano } = useLoja();
  const params = useSearchParams();
  const [planejando, setPlanejando] = useState(false);
  const [criados, setCriados] = useState(0);
  useEffect(() => { if (params.get("planejar")) setPlanejando(true); }, [params]);
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
            <button className="btn primario" onClick={() => { setCriados(0); setPlanejando(true); }}><Sparkles size={16} aria-hidden /> Planejar semana com o agente</button>
            <Link href="/posts?novo=1" className="btn">Novo post</Link>
          </>
        }
      />
      {criados > 0 && (
        <div className="alerta t-success" style={{ marginBottom: 12 }} role="status">
          <span>Plano aprovado: {criados} {criados === 1 ? "post criado" : "posts criados"}. {org.pilotoAutomatico ? "O agente está produzindo um por vez; eles chegam em " : "Eles esperam você em "}<Link href={org.pilotoAutomatico ? "/aprovacoes" : "/posts"}>{org.pilotoAutomatico ? "Aprovações" : "Posts"}</Link>.</span>
        </div>
      )}
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
                    <Link key={p.id} href={`/posts/${p.id}`} className={`cal-post t-${e.tom}`} title={`${p.pauta.tema} — ${e.rotulo} — pilar: ${p.pauta.pilar || "sem pilar"}`}>
                      {e.rotulo}: {p.pauta.tema}
                    </Link>
                  );
                })}
              </div>
            );
          })}
        </div>
      </div>

      {planejando && (
        <div className="drawer-fundo" onClick={() => setPlanejando(false)}>
          <aside className="drawer stack" role="dialog" aria-label="Plano semanal" onClick={(e) => e.stopPropagation()}>
            <div className="row" style={{ justifyContent: "space-between" }}>
              <h2 style={{ margin: 0 }}>Plano semanal</h2>
              <button className="btn peq" onClick={() => setPlanejando(false)} aria-label="Fechar"><X size={16} /></button>
            </div>
            <p className="muted" style={{ margin: 0 }}>O agente propõe os posts que faltam para cumprir a cadência, distribuindo os pilares da {org.nome}.</p>
            <PlanoSemanal org={org} posts={posts} aoAprovar={(itens) => { setCriados(criarPlano(itens)); setPlanejando(false); }} />
          </aside>
        </div>
      )}
    </>
  );
}

export default function Calendario() {
  return (
    <Suspense>
      <Mes />
    </Suspense>
  );
}
