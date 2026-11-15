"use client";
import Link from "next/link";
import { useMemo, useState } from "react";
import { Download } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { ESTADOS, ETAPAS, estadoDoPost, fmtNum, fmtUsd, nomeFormato } from "@/mock/regras";
import type { RegistroCusto } from "@/mock/types";
import { PageHeader } from "@/components/ui";

export default function Custos() {
  const { dados } = useLoja();
  const [orgF, setOrgF] = useState("todas");
  const [dias, setDias] = useState(30);

  const base = useMemo(() => {
    if (!dados) return [];
    const corte = Date.now() - dias * 86400_000;
    return dados.custos.filter((c) => (orgF === "todas" || c.orgId === orgF) && new Date(c.em).getTime() >= corte);
  }, [dados, orgF, dias]);
  if (!dados) return null;

  const post = (id: string) => dados.posts.find((p) => p.id === id)!;
  const org = (id: string) => dados.orgs.find((o) => o.id === id)!;
  const soma = (l: RegistroCusto[]) => l.reduce((s, c) => s + c.custoUsd, 0);
  const tokens = base.filter((c) => c.unidade === "tokens").reduce((s, c) => s + c.quantidade, 0);
  const agrupar = (f: (c: RegistroCusto) => string) => {
    const m = new Map<string, RegistroCusto[]>();
    base.forEach((c) => m.set(f(c), [...(m.get(f(c)) ?? []), c]));
    return [...m.entries()].map(([k, l]) => ({ k, usd: soma(l), n: l.length })).sort((a, b) => b.usd - a.usd);
  };

  const porPost = agrupar((c) => c.postId).map((x) => {
    const p = post(x.k);
    return { ...x, p, estado: estadoDoPost(p, org(p.orgId)) };
  });
  const media = (filtro: (e: string) => boolean, formato: string) => {
    const l = porPost.filter((x) => x.p.pauta.formato === formato && filtro(x.estado));
    return l.length ? soma(l.flatMap((x) => base.filter((c) => c.postId === x.k))) / l.length : null;
  };
  const publicados = (e: string) => e === "publicado";
  const descartados = (e: string) => e === "descartado";
  const custoDescartado = soma(base.filter((c) => post(c.postId).descartado));
  const custoAjustes = soma(base.filter((c, i, arr) => arr.findIndex((x) => x.postId === c.postId && x.etapa === c.etapa) !== i));

  const exportar = () => {
    const cab = ["data", "organizacao", "post", "agente", "etapa", "formato", "redes", "fornecedor", "unidade", "quantidade", "preco_unitario_usd", "custo_usd", "situacao_post"];
    const linhas = base.map((c) => {
      const p = post(c.postId);
      return [c.em, org(c.orgId).nome, p.pauta.tema, c.agente, ETAPAS[c.etapa - 1].nome, p.pauta.formato, p.pauta.redes.join("|"), c.fornecedor, c.unidade, c.quantidade, c.precoUnitarioUsd, c.custoUsd.toFixed(6), estadoDoPost(p, org(p.orgId))];
    });
    const csv = [cab, ...linhas].map((l) => l.map((v) => `"${String(v).replace(/"/g, '""')}"`).join(";")).join("\n");
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8" }));
    a.download = `custos-laura-growth-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(a.href);
  };

  const Tabela = ({ titulo, linhas }: { titulo: string; linhas: { k: string; usd: number; n: number }[] }) => (
    <section className="card">
      <h2>{titulo}</h2>
      <table><thead><tr><th>Item</th><th className="d">Tarefas</th><th className="d">Custo</th></tr></thead>
        <tbody>{linhas.map((l) => <tr key={l.k}><td>{l.k}</td><td className="d">{l.n}</td><td className="d">{fmtUsd(l.usd)}</td></tr>)}</tbody>
      </table>
    </section>
  );

  return (
    <>
      <PageHeader
        titulo="Custos (Operação)"
        descricao="Só a equipe Laura vê esta tela. Tokens, fornecedores e preços são de exemplo."
        acoes={<button className="btn primario" onClick={exportar} disabled={!base.length}><Download size={16} aria-hidden /> Exportar planilha</button>}
      />
      <div className="row" style={{ marginBottom: 16 }}>
        <label className="row small">Organização
          <select value={orgF} onChange={(e) => setOrgF(e.target.value)}>
            <option value="todas">Todas</option>
            {dados.orgs.map((o) => <option key={o.id} value={o.id}>{o.nome}</option>)}
          </select>
        </label>
        <label className="row small">Período
          <select value={dias} onChange={(e) => setDias(Number(e.target.value))}>
            <option value={7}>Últimos 7 dias</option><option value={30}>Últimos 30 dias</option><option value={3650}>Tudo</option>
          </select>
        </label>
      </div>
      <div className="grid g4" style={{ marginBottom: 16 }}>
        {[
          ["Custo total", fmtUsd(soma(base))],
          ["Tokens de texto", fmtNum(tokens)],
          ["Custo em posts descartados", fmtUsd(custoDescartado)],
          ["Custo de ajustes (retrabalho)", fmtUsd(custoAjustes)],
        ].map(([r, v]) => <div key={r} className="card metric"><div className="small muted">{r}</div><div className="valor" style={{ fontSize: 22 }}>{v}</div></div>)}
      </div>
      <section className="card" style={{ marginBottom: 16 }}>
        <h2>Custo médio por post</h2>
        <table><thead><tr><th>Formato</th><th className="d">Publicados</th><th className="d">Descartados</th><th className="d">Todos</th></tr></thead>
          <tbody>
            {(["imagem", "carrossel"] as const).map((f) => (
              <tr key={f}><td>{nomeFormato(f)}</td>
                {[publicados, descartados, () => true].map((fl, i) => { const m = media(fl, f); return <td key={i} className="d">{m === null ? "—" : fmtUsd(m)}</td>; })}
              </tr>
            ))}
          </tbody>
        </table>
      </section>
      <div className="grid g3" style={{ marginBottom: 16 }}>
        <Tabela titulo="Por etapa" linhas={agrupar((c) => ETAPAS[c.etapa - 1].nome)} />
        <Tabela titulo="Por agente" linhas={agrupar((c) => c.agente)} />
        <Tabela titulo="Por organização" linhas={agrupar((c) => org(c.orgId).nome)} />
      </div>
      <section className="card rolagem">
        <h2>Por post</h2>
        <table><thead><tr><th>Post</th><th>Organização</th><th>Formato</th><th>Situação</th><th className="d">Custo</th></tr></thead>
          <tbody>{porPost.map((x) => (
            <tr key={x.k}><td><Link href={`/posts/${x.k}`}>{x.p.pauta.tema}</Link></td><td>{org(x.p.orgId).nome}</td><td>{nomeFormato(x.p.pauta.formato)}</td><td>{ESTADOS[x.estado].rotulo}</td><td className="d">{fmtUsd(x.usd)}</td></tr>
          ))}</tbody>
        </table>
      </section>
    </>
  );
}
