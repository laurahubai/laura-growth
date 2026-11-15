"use client";
// Cadastro das APIs de IA (AGT-07) e escolha da API por etapa (AGT-08). Vale para a plataforma inteira.
import { useState } from "react";
import { KeyRound, Pencil, Plus, Trash2 } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { ETAPAS_IA } from "@/mock/regras";
import type { ApiIA } from "@/mock/types";

const VAZIA = { id: "", nome: "", tipo: "texto" as ApiIA["tipo"], endpoint: "", modelo: "", chave: "" };

export function ApisDeIA() {
  const { dados, salvarApi, removerApi, definirApiDaEtapa } = useLoja();
  const [form, setForm] = useState<typeof VAZIA | null>(null);
  if (!dados) return null;
  const { apis, apiPorEtapa } = dados;

  const existente = form ? apis.find((a) => a.id === form.id) : undefined;
  const invalido = !form || !form.nome.trim() || !form.endpoint.trim() || (!existente && form.chave.trim().length < 4);
  const salvar = () => {
    if (!form || invalido) return;
    const chave = form.chave.trim();
    salvarApi({
      id: form.id || `api-${Date.now().toString(36)}`,
      nome: form.nome.trim(),
      tipo: form.tipo,
      endpoint: form.endpoint.trim(),
      modelo: form.modelo.trim(),
      chaveFinal: chave ? chave.slice(-4) : existente!.chaveFinal, // a chave inteira é descartada aqui
    });
    setForm(null);
  };
  const semApi = ETAPAS_IA.filter((e) => !apis.some((a) => a.id === apiPorEtapa[e.id]));

  return (
    <>
      <section className="card stack">
        <div className="row" style={{ justifyContent: "space-between" }}>
          <h2 style={{ margin: 0 }}>APIs de IA cadastradas</h2>
          <span className="badge t-neutral">Operação Laura · vale para todas as organizações</span>
        </div>
        <p className="muted" style={{ margin: 0 }}>Cada cadastro é uma conexão com um modelo de IA: o endereço da API (endpoint), o modelo e a chave de acesso.</p>
        <div className="alerta t-warning">
          <KeyRound size={16} aria-hidden />
          <span>Protótipo: <b>não use chaves reais</b>. Os dados ficam só neste navegador e a chave é descartada ao salvar; guardamos apenas os 4 últimos caracteres. No produto, a chave ficará cifrada no servidor.</span>
        </div>

        {apis.length === 0 && !form && <p className="vazio" style={{ padding: 12 }}>Nenhuma API cadastrada. Fornecedores de IA: a definir.</p>}
        {apis.length > 0 && (
          <div className="rolagem">
            <table>
              <thead><tr><th>Nome</th><th>Tipo</th><th>Endpoint</th><th>Modelo</th><th>Chave</th><th /></tr></thead>
              <tbody>
                {apis.map((a) => (
                  <tr key={a.id}>
                    <td><b>{a.nome}</b></td>
                    <td><span className={`badge ${a.tipo === "imagem" ? "t-ai" : "t-info"}`}>{a.tipo === "imagem" ? "Imagem" : "Texto"}</span></td>
                    <td className="small muted" style={{ wordBreak: "break-all" }}>{a.endpoint}</td>
                    <td className="small">{a.modelo || "—"}</td>
                    <td className="num small">••••{a.chaveFinal}</td>
                    <td>
                      <span className="row" style={{ gap: 6, flexWrap: "nowrap" }}>
                        <button className="btn peq" aria-label={`Editar ${a.nome}`} onClick={() => setForm({ ...a, chave: "" })}><Pencil size={14} aria-hidden /></button>
                        <button className="btn peq" aria-label={`Remover ${a.nome}`} onClick={() => { if (confirm(`Remover a API "${a.nome}"? As etapas que usam esta API ficam sem API definida.`)) removerApi(a.id); }}><Trash2 size={14} aria-hidden /></button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {form ? (
          <form className="stack" onSubmit={(e) => { e.preventDefault(); salvar(); }} style={{ borderTop: "1px solid var(--laura-border)", paddingTop: 12 }}>
            <h3 style={{ margin: 0 }}>{existente ? `Editar ${existente.nome}` : "Nova API de IA"}</h3>
            <div className="grid g2">
              <label className="campo">Nome<input value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} placeholder="Ex.: Texto principal" autoFocus /></label>
              <label className="campo">
                Tipo
                <select value={form.tipo} onChange={(e) => setForm({ ...form, tipo: e.target.value as ApiIA["tipo"] })}>
                  <option value="texto">Texto (pauta, legenda, revisão)</option>
                  <option value="imagem">Imagem (arte e carrossel)</option>
                </select>
              </label>
            </div>
            <label className="campo">Endpoint (endereço da API)<input value={form.endpoint} onChange={(e) => setForm({ ...form, endpoint: e.target.value })} placeholder="Endereço informado pelo fornecedor" /></label>
            <label className="campo">Modelo<input value={form.modelo} onChange={(e) => setForm({ ...form, modelo: e.target.value })} placeholder="Nome do modelo, como o fornecedor informa" /></label>
            <label className="campo">
              Chave de API
              <input type="password" autoComplete="off" value={form.chave} onChange={(e) => setForm({ ...form, chave: e.target.value })} placeholder={existente ? `Cadastrada (••••${existente.chaveFinal}). Deixe em branco para manter.` : "Chave fictícia, só para testar a tela"} />
            </label>
            <div className="row">
              <button className="btn primario" type="submit" disabled={invalido}>Salvar API</button>
              <button className="btn" type="button" onClick={() => setForm(null)}>Cancelar</button>
            </div>
          </form>
        ) : (
          <button className="btn" style={{ alignSelf: "flex-start" }} onClick={() => setForm({ ...VAZIA })}><Plus size={16} aria-hidden /> Adicionar API</button>
        )}
      </section>

      <section className="card stack">
        <h2 style={{ margin: 0 }}>API usada em cada etapa</h2>
        <p className="muted" style={{ margin: 0 }}>Cada etapa do processo pode usar uma API diferente. A mesma API pode servir a várias etapas.</p>
        {ETAPAS_IA.map((e) => {
          const opcoes = apis.filter((a) => a.tipo === e.tipo);
          const atual = apis.some((a) => a.id === apiPorEtapa[e.id]) ? apiPorEtapa[e.id]! : "";
          return (
            <label key={e.id} className="campo">
              <span>{e.nome} <span className="small muted" style={{ fontWeight: 400 }}>· {e.agente} · usa API de {e.tipo}</span></span>
              <select value={atual} onChange={(ev) => definirApiDaEtapa(e.id, ev.target.value)}>
                <option value="">{opcoes.length ? "Não definida" : `Cadastre uma API de ${e.tipo}`}</option>
                {opcoes.map((a) => <option key={a.id} value={a.id}>{a.nome}{a.modelo ? ` · ${a.modelo}` : ""}</option>)}
              </select>
            </label>
          );
        })}
        {semApi.length > 0 ? (
          <div className="alerta t-warning">Sem API definida: {semApi.map((e) => e.nome).join(" · ")}. No protótipo, essas etapas seguem com o fornecedor de exemplo.</div>
        ) : (
          <div className="alerta t-success">Todas as etapas têm uma API definida.</div>
        )}
        <p className="small muted">A escolha vale para as próximas tarefas dos agentes e aparece no painel de Custos.</p>
      </section>
    </>
  );
}
