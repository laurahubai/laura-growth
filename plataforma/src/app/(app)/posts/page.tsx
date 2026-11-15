"use client";
import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Sparkles, X } from "lucide-react";
import { useLoja, usePostsDaOrg } from "@/mock/loja";
import { ESTADOS, estadoDoPost, ETAPAS, fmtData, hoje, nomeFormato, nomeRede, REDES } from "@/mock/regras";
import type { EstadoPost, Formato, Rede } from "@/mock/types";
import { PageHeader, StatusBadge } from "@/components/ui";

function Lista() {
  const { org, novoPost } = useLoja();
  const posts = usePostsDaOrg();
  const router = useRouter();
  const params = useSearchParams();
  const [aberto, setAberto] = useState(false);
  const [filtro, setFiltro] = useState<EstadoPost | "todos">("todos");
  const [tema, setTema] = useState("");
  const [objetivo, setObjetivo] = useState("");
  const [redes, setRedes] = useState<Rede[]>(["instagram"]);
  const [formato, setFormato] = useState<Formato>("imagem");
  const [data, setData] = useState(hoje());

  useEffect(() => {
    if (params.get("novo")) setAberto(true);
  }, [params]);
  if (!org) return null;

  const lista = posts
    .map((p) => ({ p, e: estadoDoPost(p, org) }))
    .filter((x) => filtro === "todos" || x.e === filtro)
    .sort((a, b) => b.p.data.localeCompare(a.p.data));

  const criar = (origem: "sugestao" | "pedido") => {
    const id = novoPost(origem, origem === "pedido" ? { tema, objetivo, redes, formato } : { tema: "(o agente vai sugerir)", objetivo: "", redes: ["instagram"], formato: "imagem" }, data);
    router.push(`/posts/${id}`);
  };

  return (
    <>
      <PageHeader
        titulo="Posts"
        descricao={`Todos os posts da ${org.nome}.`}
        acoes={<button className="btn primario" onClick={() => setAberto(true)}>Novo post</button>}
      />
      <div className="row" style={{ marginBottom: 12 }}>
        <label className="row small">
          Situação
          <select value={filtro} onChange={(e) => setFiltro(e.target.value as EstadoPost | "todos")}>
            <option value="todos">Todas</option>
            {Object.entries(ESTADOS).map(([k, v]) => (
              <option key={k} value={k}>{v.rotulo}</option>
            ))}
          </select>
        </label>
        <span className="small muted num">{lista.length} posts</span>
      </div>
      <div className="card rolagem" style={{ padding: 0 }}>
        {lista.length === 0 ? (
          <p className="vazio">Nenhum post nesta situação.</p>
        ) : (
          <table>
            <thead>
              <tr><th>Tema</th><th>Etapa</th><th>Redes</th><th>Formato</th><th>Data</th><th>Situação</th></tr>
            </thead>
            <tbody>
              {lista.map(({ p, e }) => (
                <tr key={p.id} className="clicavel" onClick={() => router.push(`/posts/${p.id}`)}>
                  <td><a href={`/posts/${p.id}`} onClick={(ev) => ev.preventDefault()}>{p.pauta.tema}</a></td>
                  <td className="muted">{p.etapa}. {ETAPAS[p.etapa - 1].nome}</td>
                  <td className="muted">{p.pauta.redes.map(nomeRede).join(", ")}</td>
                  <td className="muted">{nomeFormato(p.pauta.formato)}</td>
                  <td className="num">{fmtData(p.data)}</td>
                  <td><StatusBadge estado={e} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {aberto && (
        <div className="drawer-fundo" onClick={() => setAberto(false)}>
          <aside className="drawer stack" role="dialog" aria-label="Novo post" onClick={(e) => e.stopPropagation()}>
            <div className="row" style={{ justifyContent: "space-between" }}>
              <h2 style={{ margin: 0 }}>Novo post</h2>
              <button className="btn peq" onClick={() => setAberto(false)} aria-label="Fechar"><X size={16} /></button>
            </div>
            <label className="campo">
              Data prevista
              <input type="date" value={data} onChange={(e) => setData(e.target.value)} />
            </label>
            <div className="card stack" style={{ background: "var(--laura-ai-soft)", borderColor: "transparent" }}>
              <h3 style={{ margin: 0 }}>Deixar o agente sugerir</h3>
              <p className="small muted">O Strategy Agent lê o kit de marca e o calendário e propõe tema, objetivo, rede e formato.</p>
              <button className="btn" onClick={() => criar("sugestao")}><Sparkles size={16} aria-hidden /> Pedir sugestão de pauta</button>
            </div>
            <h3 style={{ margin: "8px 0 0" }}>Ou descreva o que você quer</h3>
            <label className="campo">Tema<input value={tema} onChange={(e) => setTema(e.target.value)} placeholder="Ex.: Dicas para manter a cadência" /></label>
            <label className="campo">Objetivo<input value={objetivo} onChange={(e) => setObjetivo(e.target.value)} placeholder="Ex.: Gerar salvamentos" /></label>
            <fieldset className="stack" style={{ border: 0, padding: 0, margin: 0, gap: 6 }}>
              <legend className="small" style={{ fontWeight: 500, marginBottom: 6 }}>Redes</legend>
              {REDES.map((r) => (
                <label key={r.id} className="row">
                  <input type="checkbox" checked={redes.includes(r.id)} onChange={(e) => setRedes(e.target.checked ? [...redes, r.id] : redes.filter((x) => x !== r.id))} />
                  {r.nome}
                </label>
              ))}
            </fieldset>
            <label className="campo">
              Formato
              <select value={formato} onChange={(e) => setFormato(e.target.value as Formato)}>
                <option value="imagem">Imagem com legenda</option>
                <option value="carrossel">Carrossel</option>
              </select>
            </label>
            <button className="btn primario" disabled={!tema.trim() || redes.length === 0} onClick={() => criar("pedido")}>Criar post com meu pedido</button>
          </aside>
        </div>
      )}
    </>
  );
}

export default function Posts() {
  return (
    <Suspense>
      <Lista />
    </Suspense>
  );
}
