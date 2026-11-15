"use client";
// Plano semanal em lote (CAL-04): o agente propõe os posts que faltam para a cadência e a pessoa aprova o plano.
import { useEffect, useState } from "react";
import { Sparkles, Trash2 } from "lucide-react";
import { PASSOS_PLANO, planejarSemana, type ItemDoPlano } from "@/mock/agente-falso";
import { diasDaSemana, fmtData, nomeFormato, nomeRede, OBJETIVOS } from "@/mock/regras";
import type { Objetivo, Organizacao, Post } from "@/mock/types";
import { TagIA } from "@/components/ui";

const espera = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** Editor do plano proposto. Usado no calendário e no fim do onboarding. */
export function PlanoSemanal({ org, posts, rotuloAprovar, aoAprovar }: { org: Organizacao; posts: Post[]; rotuloAprovar?: string; aoAprovar: (itens: ItemDoPlano[]) => void }) {
  const [semana, setSemana] = useState(1);
  const [passos, setPassos] = useState<string[]>([]);
  const [itens, setItens] = useState<ItemDoPlano[] | null>(null);
  const dias = diasDaSemana(semana);

  useEffect(() => {
    let vivo = true;
    (async () => {
      setItens(null);
      setPassos([]);
      for (const p of PASSOS_PLANO) {
        await espera(450);
        if (!vivo) return;
        setPassos((x) => [...x, p]);
      }
      await espera(350);
      if (vivo) setItens(planejarSemana(org, posts, semana));
    })();
    return () => { vivo = false; };
    // Recalcula só ao trocar de semana ou de organização.
  }, [semana, org.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const mudar = (i: number, f: (x: ItemDoPlano) => ItemDoPlano) => setItens((l) => l!.map((x, j) => (j === i ? f(x) : x)));
  const pilares = org.kit.pilares.length ? org.kit.pilares : ["a marca"];

  return (
    <div className="stack">
      <label className="campo" style={{ maxWidth: 320 }}>
        Semana a planejar
        <select value={semana} onChange={(e) => setSemana(Number(e.target.value))}>
          {[0, 1, 2].map((n) => {
            const d = diasDaSemana(n);
            return <option key={n} value={n}>{n === 0 ? "Esta semana" : n === 1 ? "Próxima semana" : "Daqui a duas semanas"} ({fmtData(d[0]).slice(0, 5)} a {fmtData(d[6]).slice(0, 5)})</option>;
          })}
        </select>
      </label>

      {!itens ? (
        <div className="card" style={{ background: "var(--laura-ai-soft)", borderColor: "transparent" }} aria-live="polite">
          <p className="row" style={{ margin: 0 }}><span className="pulsar" /> <b>Strategy Agent montando o plano…</b></p>
          <ul className="atividade">{passos.map((p) => <li key={p}>{p}</li>)}</ul>
        </div>
      ) : itens.length === 0 ? (
        <div className="alerta t-success">A cadência desta semana já está coberta pelos posts do calendário. Nada a propor.</div>
      ) : (
        <>
          <div className="row"><TagIA texto="Plano proposto pelo agente (exemplo)" /><span className="small muted">Ajuste o que quiser antes de aprovar.</span></div>
          {itens.map((it, i) => (
            <div key={i} className="card stack" style={{ gap: 8 }}>
              <div className="row">
                <input style={{ flex: 1 }} value={it.pauta.tema} aria-label={`Tema do post ${i + 1}`} onChange={(e) => mudar(i, (x) => ({ ...x, pauta: { ...x.pauta, tema: e.target.value } }))} />
                <button className="btn peq" aria-label={`Tirar o post ${i + 1} do plano`} onClick={() => setItens(itens.filter((_, j) => j !== i))}><Trash2 size={14} aria-hidden /></button>
              </div>
              <div className="row">
                <select value={it.data} aria-label="Dia" onChange={(e) => mudar(i, (x) => ({ ...x, data: e.target.value }))}>
                  {dias.map((d) => <option key={d} value={d}>{new Date(d + "T12:00").toLocaleDateString("pt-BR", { weekday: "short", day: "2-digit", month: "2-digit" })}</option>)}
                </select>
                <select value={it.pauta.pilar} aria-label="Pilar" onChange={(e) => mudar(i, (x) => ({ ...x, pauta: { ...x.pauta, pilar: e.target.value } }))}>
                  {pilares.map((p) => <option key={p} value={p}>{p}</option>)}
                </select>
                <select value={it.pauta.objetivo} aria-label="Objetivo" onChange={(e) => mudar(i, (x) => ({ ...x, pauta: { ...x.pauta, objetivo: e.target.value as Objetivo } }))}>
                  {OBJETIVOS.map((o) => <option key={o.id} value={o.id}>{o.nome}</option>)}
                </select>
              </div>
              <span className="small muted">{nomeFormato(it.pauta.formato)} · {it.pauta.redes.map(nomeRede).join(", ")}</span>
            </div>
          ))}
          <div className="alerta t-info">
            Aprovar o plano cria os posts{org.pilotoAutomatico ? " e o agente começa a produzi-los sozinho" : ""}. Nenhum deles é publicado sem a sua aprovação final, post a post.
          </div>
          <button className="btn primario" style={{ alignSelf: "flex-start" }} disabled={itens.some((x) => !x.pauta.tema.trim())} onClick={() => aoAprovar(itens)}>
            <Sparkles size={16} aria-hidden /> {rotuloAprovar ?? "Aprovar plano e criar"} {itens.length} {itens.length === 1 ? "post" : "posts"}
          </button>
        </>
      )}
    </div>
  );
}
