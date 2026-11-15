"use client";
import Link from "next/link";
import { use, useState } from "react";
import { ArrowLeft, Check, Download, MessageSquare, Pause, Pencil, RotateCcw, Sparkles, Trash2, Upload } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { estadoDoPost, ETAPAS, fmtData, fmtHora, fmtNum, fmtUsd, nomeFormato, nomeObjetivo, nomeRede, OBJETIVOS, REDES } from "@/mock/regras";
import type { Formato, Objetivo, Organizacao, Pauta, Post, Texto, Visual } from "@/mock/types";
import { EtiquetasPauta, PageHeader, Previas, SlideArte, StatusBadge, TagIA } from "@/components/ui";

export default function DetalhePost({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const loja = useLoja();
  const { dados, org } = loja;
  const post = dados?.posts.find((p) => p.id === id);
  const [modo, setModo] = useState<"nada" | "ajuste" | "editar">("nada");
  const [comentario, setComentario] = useState("");

  if (!dados || !org) return null;
  if (!post || post.orgId !== org.id)
    return <p className="vazio">Post não encontrado nesta organização. <Link href="/posts">Voltar aos posts</Link></p>;

  const estado = estadoDoPost(post, org);
  const custos = dados.custos.filter((c) => c.postId === post.id);
  const total = custos.reduce((s, c) => s + c.custoUsd, 0);
  const bloqueado = post.trabalhando || post.descartado;
  const fechar = () => { setModo("nada"); setComentario(""); };

  return (
    <>
      <Link href="/posts" className="row small" style={{ marginBottom: 8 }}><ArrowLeft size={14} /> Posts</Link>
      <PageHeader
        titulo={post.pauta.tema}
        descricao={`${nomeFormato(post.pauta.formato)} · ${post.pauta.redes.map(nomeRede).join(", ")} · previsto para ${fmtData(post.data)}`}
        acoes={<><EtiquetasPauta pauta={post.pauta} /><StatusBadge estado={estado} /></>}
      />
      <ol className="stepper" aria-label="Etapas do post" style={{ listStyle: "none", padding: 0 }}>
        {ETAPAS.map((e) => (
          <li key={e.n} className={`passo ${e.n < post.etapa ? "feito" : ""} ${e.n === post.etapa ? "atual" : ""} ${e.n === 5 ? "obrig" : ""}`} aria-current={e.n === post.etapa ? "step" : undefined}>
            <b>{e.n}. {e.nome}</b>
            {e.n < post.etapa ? "Concluída" : e.n === post.etapa ? "Agora" : e.n === 5 ? "Obrigatória" : "A seguir"}
          </li>
        ))}
      </ol>

      <div className="grid g2">
        <div className="stack">
          <section className="card stack">
            {post.descartado && <div className="alerta t-neutral">Este post foi descartado. O custo dele continua no painel de custos.</div>}
            {estado === "atrasado" && <div className="alerta t-warning">A data prevista passou sem aprovação. Ele não será publicado até ser aprovado e reagendado.</div>}

            {post.etapa === 1 && <VerPauta post={post} />}
            {post.etapa === 2 && <VerTexto post={post} />}
            {post.etapa === 3 && <VerVisual post={post} org={org} />}
            {post.etapa === 4 && <VerRevisao post={post} />}
            {post.etapa === 5 && (
              <div className="stack">
                <h2 style={{ margin: 0 }}>Como vai ficar em cada rede</h2>
                <p className="muted">Confira e aprove. Nada vai ao ar sem esta aprovação.</p>
                <Previas post={post} org={org} />
                {post.revisao && post.revisao.avisos.some((a) => !a.startsWith("Nenhum")) && post.revisao.avisos.map((a, i) => <div key={i} className="alerta t-warning">{a}</div>)}
              </div>
            )}
            {post.etapa === 6 && <Publicacao post={post} org={org} />}

            {post.trabalhando && (
              <div className="row" style={{ justifyContent: "space-between" }}>
                <span className="row muted"><span className="pulsar" /> {post.piloto ? "Piloto automático: o agente segue sozinho até a aprovação final." : "O agente está trabalhando. Acompanhe ao lado."}</span>
                {post.piloto && (
                  <button className="btn peq" onClick={() => loja.pausarPiloto(post.id)} title="O agente termina o passo atual e espera você">
                    <Pause size={14} aria-hidden /> Pausar para eu revisar
                  </button>
                )}
              </div>
            )}
            {!post.trabalhando && !post.piloto && post.etapa <= 4 && !post.descartado && org.pilotoAutomatico && (
              <div className="alerta t-neutral">Piloto automático pausado neste post. O agente espera você em cada etapa.</div>
            )}

            {post.etapa <= 5 && !post.descartado && (
              <div className="row" style={{ borderTop: "1px solid var(--laura-border)", paddingTop: 12 }}>
                <button className="btn primario" disabled={bloqueado || (post.etapa === 1 && post.pauta.tema.startsWith("("))} onClick={() => { fechar(); loja.aprovarEtapa(post.id); }}>
                  <Check size={16} aria-hidden /> {post.etapa === 5 ? "Aprovar post" : post.etapa === 4 ? "Seguir para aprovação final" : "Aprovar e seguir"}
                </button>
                <button className="btn" disabled={bloqueado} onClick={() => setModo(modo === "ajuste" ? "nada" : "ajuste")}>
                  <MessageSquare size={16} aria-hidden /> {post.etapa === 5 ? "Devolver com comentário" : "Pedir ajuste"}
                </button>
                {post.etapa <= 4 && !post.piloto && org.pilotoAutomatico && (
                  <button className="btn" disabled={bloqueado || (post.etapa === 1 && post.pauta.tema.startsWith("("))} onClick={() => { fechar(); loja.aprovarEtapa(post.id, true); }}>
                    <Sparkles size={16} aria-hidden /> Retomar piloto automático
                  </button>
                )}
                {(post.etapa <= 3 || post.etapa === 5) && (
                  <button className="btn" disabled={bloqueado} onClick={() => setModo(modo === "editar" ? "nada" : "editar")}>
                    <Pencil size={16} aria-hidden /> {post.etapa === 5 ? "Editar texto à mão" : "Editar à mão"}
                  </button>
                )}
                <button className="btn perigo" disabled={bloqueado} onClick={() => { if (confirm("Descartar este post? Ele sai do calendário, mas o histórico e o custo ficam guardados.")) loja.descartar(post.id); }}>
                  <Trash2 size={16} aria-hidden /> Descartar
                </button>
              </div>
            )}

            {modo === "ajuste" && (
              <div className="stack">
                <label className="campo">
                  O que você quer mudar?
                  <textarea value={comentario} onChange={(e) => setComentario(e.target.value)} placeholder="Ex.: deixe mais curto e mais leve" autoFocus />
                </label>
                <div className="row">
                  <button className="btn primario" disabled={!comentario.trim()} onClick={() => { post.etapa === 5 ? loja.devolver(post.id, comentario) : loja.pedirAjuste(post.id, comentario); fechar(); }}>
                    Enviar ao agente
                  </button>
                  <button className="btn" onClick={fechar}>Cancelar</button>
                </div>
              </div>
            )}
            {modo === "editar" && <Editor post={post} org={org} aoSalvar={fechar} />}
          </section>

          <section className="card">
            <h2>Versões anteriores</h2>
            {post.versoes.length === 0 ? (
              <p className="muted">Ainda não há versões anteriores.</p>
            ) : (
              <ul className="stack" style={{ listStyle: "none", padding: 0, margin: 0 }}>
                {[...post.versoes].reverse().map((v) => (
                  <li key={v.id} className="row" style={{ justifyContent: "space-between" }}>
                    <span>
                      <b>{ETAPAS[v.etapa - 1].nome}</b> · <span className="muted num">{fmtHora(v.em)}</span>
                      {v.comentario && <span className="muted"> · antes do ajuste “{v.comentario}”</span>}
                      <div className="small muted" style={{ maxWidth: 520, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{resumo(v.conteudo)}</div>
                    </span>
                    {!bloqueado && v.etapa === post.etapa && (
                      <button className="btn peq" onClick={() => loja.restaurarVersao(post.id, v.id)}><RotateCcw size={14} aria-hidden /> Restaurar</button>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>

        <aside className="stack">
          <section className="card" aria-live="polite">
            <h2 className="row">Atividade do agente {post.trabalhando && <span className="pulsar" aria-label="trabalhando" />}</h2>
            {post.atividade.length === 0 ? (
              <p className="muted">Quando o agente trabalhar neste post, cada passo aparece aqui ao vivo.</p>
            ) : (
              <ul className="atividade">
                {post.atividade.map((a, i) => (
                  <li key={i}><span className="hora">{new Date(a.em).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit", second: "2-digit" })}</span>{a.texto}</li>
                ))}
              </ul>
            )}
          </section>
          <section className="card">
            <h2>Custo deste post</h2>
            <p className="small muted">Visão Operação (a organização não vê). Valores de exemplo.</p>
            <table>
              <tbody>
                {custos.map((c) => (
                  <tr key={c.id}>
                    <td className="small">{ETAPAS[c.etapa - 1].nome}<div className="muted">{c.agente}</div></td>
                    <td className="d small">{fmtNum(c.quantidade)} {c.unidade === "tokens" ? "tokens" : "img"}</td>
                    <td className="d small">{fmtUsd(c.custoUsd)}</td>
                  </tr>
                ))}
                <tr><td><b>Total</b></td><td /><td className="d"><b>{fmtUsd(total)}</b></td></tr>
              </tbody>
            </table>
          </section>
          <section className="card">
            <h2>Histórico</h2>
            <ul className="atividade">
              {[...post.historico].reverse().map((h, i) => (
                <li key={i} style={{ flexDirection: "column", gap: 0 }}>
                  <span><b>{h.quem}</b> {h.quem.includes("Agent") && <TagIA texto="IA" />}</span>
                  <span>{h.acao}</span>
                  <span className="small muted num">{fmtHora(h.em)}</span>
                </li>
              ))}
            </ul>
          </section>
        </aside>
      </div>
    </>
  );
}

function resumo(c: unknown): string {
  const o = c as Record<string, unknown>;
  if ("legenda" in o) return String(o.legenda);
  if ("tema" in o) return String(o.tema);
  if ("slides" in o) return (o.slides as { titulo: string }[]).map((s) => s.titulo).join(" · ");
  if ("avisos" in o) return (o.avisos as string[]).join(" ");
  return "";
}

function VerPauta({ post }: { post: Post }) {
  if (post.pauta.tema.startsWith("(")) return <p className="muted">O Strategy Agent está preparando a sugestão de pauta…</p>;
  return (
    <div className="stack">
      <div className="row"><h2 style={{ margin: 0 }}>Pauta</h2>{post.origem === "sugestao" ? <TagIA texto="Sugestão da Laura" /> : <span className="badge t-neutral">Pedido seu</span>}</div>
      <dl className="grid g2" style={{ margin: 0 }}>
        <div><dt className="small muted">Tema</dt><dd style={{ margin: 0 }}>{post.pauta.tema}</dd></div>
        <div><dt className="small muted">Pilar de conteúdo</dt><dd style={{ margin: 0 }}>{post.pauta.pilar || "—"}</dd></div>
        <div><dt className="small muted">Objetivo</dt><dd style={{ margin: 0 }}>{nomeObjetivo(post.pauta.objetivo)} <span className="small muted">· {OBJETIVOS.find((o) => o.id === post.pauta.objetivo)?.explica}</span></dd></div>
        <div><dt className="small muted">Redes</dt><dd style={{ margin: 0 }}>{post.pauta.redes.map(nomeRede).join(", ")}</dd></div>
        <div><dt className="small muted">Formato</dt><dd style={{ margin: 0 }}>{nomeFormato(post.pauta.formato)}</dd></div>
      </dl>
      <p className="small muted">Ao aprovar, o Creative Agent escreve o texto.</p>
    </div>
  );
}

function VerTexto({ post }: { post: Post }) {
  if (!post.texto) return <p className="muted">O Creative Agent está escrevendo o texto…</p>;
  return (
    <div className="stack">
      <div className="row"><h2 style={{ margin: 0 }}>Texto</h2><TagIA /></div>
      <div className="legenda">{post.texto.legenda}</div>
      <div><span className="small muted">Chamada para ação (CTA)</span><div><b>{post.texto.cta || "Sem CTA"}</b></div></div>
      <p style={{ color: "var(--laura-brand-strong)" }}>{post.texto.hashtags}</p>
      <p className="small muted num">{post.texto.legenda.length} caracteres</p>
    </div>
  );
}

function VerVisual({ post, org }: { post: Post; org: Organizacao }) {
  const loja = useLoja();
  if (!post.visual) return <p className="muted">O Creative Agent está montando a arte…</p>;
  const enviar = (i: number, f?: File) => {
    if (!f) return;
    const r = new FileReader();
    r.onload = () => {
      const slides = post.visual!.slides.map((s, j) => (j === i ? { ...s, imagem: String(r.result) } : s));
      loja.editarManual(post.id, "visual", { slides });
    };
    r.readAsDataURL(f);
  };
  return (
    <div className="stack">
      <div className="row"><h2 style={{ margin: 0 }}>Visual</h2><TagIA texto="Arte de exemplo" /></div>
      <div className="slides">
        {post.visual.slides.map((s, i) => (
          <div key={i} className="stack" style={{ gap: 6 }}>
            <SlideArte slide={s} org={org} i={i} total={post.visual!.slides.length} />
            <label className="btn peq" style={{ justifyContent: "center" }}>
              <Upload size={14} aria-hidden /> Enviar imagem
              <input type="file" accept="image/*" hidden onChange={(e) => enviar(i, e.target.files?.[0])} disabled={post.trabalhando} />
            </label>
          </div>
        ))}
      </div>
      <p className="small muted">Tamanhos por rede: {post.pauta.redes.map((r) => `${nomeRede(r)} ${REDES.find((x) => x.id === r)!.tamanho}`).join(" · ")}. Imagem enviada fica só no seu navegador.</p>
    </div>
  );
}

function VerRevisao({ post }: { post: Post }) {
  if (!post.revisao) return <p className="muted">O Social Agent está conferindo o post…</p>;
  return (
    <div className="stack">
      <div className="row"><h2 style={{ margin: 0 }}>Revisão automática</h2><TagIA texto="Conferido por IA" /></div>
      {post.revisao.avisos.map((a, i) => (
        <div key={i} className={`alerta ${a.startsWith("Nenhum") ? "t-success" : "t-warning"}`}>{a}</div>
      ))}
      <p className="small muted">Os avisos são apoio; a decisão é sua.</p>
    </div>
  );
}

function Publicacao({ post, org }: { post: Post; org: Organizacao }) {
  const loja = useLoja();
  const conectada = post.pauta.redes.some((r) => org.conexoes[r] === "ativa");
  const baixar = () => {
    const conteudo = `${post.pauta.tema}\nRedes: ${post.pauta.redes.map(nomeRede).join(", ")}\nData: ${fmtData(post.data)}\nPilar: ${post.pauta.pilar}\nObjetivo: ${nomeObjetivo(post.pauta.objetivo)}\n\n${post.texto?.legenda}\n\n${post.texto?.cta}\n\n${post.texto?.hashtags}\n`;
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([conteudo], { type: "text/plain;charset=utf-8" }));
    a.download = `post-${post.data}.txt`;
    a.click();
    URL.revokeObjectURL(a.href);
    post.visual?.slides.forEach((s, i) => {
      if (!s.imagem) return;
      const img = document.createElement("a");
      img.href = s.imagem;
      img.download = `post-${post.data}-imagem-${i + 1}`;
      img.click();
    });
  };
  return (
    <div className="stack">
      <h2 style={{ margin: 0 }}>Publicação</h2>
      <p className="muted">Aprovado por {post.aprovadoPor ?? "—"}.</p>
      <label className="campo" style={{ maxWidth: 220 }}>
        Data de publicação
        <input type="date" value={post.data} onChange={(e) => loja.reagendar(post.id, e.target.value)} disabled={post.publicacao === "publicado"} />
      </label>
      {post.publicacao === "publicado" ? (
        <div className="alerta t-success">Publicado. 🎉</div>
      ) : post.publicacao === "falhou" ? (
        <div className="stack">
          <div className="alerta t-danger">A publicação falhou (simulado). O post não foi ao ar. Tente de novo ou baixe para publicar à mão.</div>
          <div className="row"><button className="btn primario" onClick={() => loja.tentarDeNovo(post.id)}>Tentar de novo</button><button className="btn" onClick={baixar}><Download size={16} /> Baixar post</button></div>
        </div>
      ) : conectada ? (
        <div className="stack">
          <div className="alerta t-brand">Agendado para {fmtData(post.data)} nas redes conectadas. Para homologar, simule o resultado:</div>
          <div className="row">
            <button className="btn primario" onClick={() => loja.simularPublicacao(post.id, true)}>Simular publicação com sucesso</button>
            <button className="btn" onClick={() => loja.simularPublicacao(post.id, false)}>Simular falha</button>
          </div>
        </div>
      ) : (
        <div className="stack">
          <div className="alerta t-success">Nenhuma rede conectada: o post está pronto para você publicar à mão.</div>
          <div className="row">
            <button className="btn primario" onClick={baixar}><Download size={16} aria-hidden /> Baixar texto e imagens</button>
            <button className="btn" onClick={() => loja.marcarPublicado(post.id)}>Marcar como publicado</button>
          </div>
        </div>
      )}
    </div>
  );
}

function Editor({ post, org, aoSalvar }: { post: Post; org: Organizacao; aoSalvar: () => void }) {
  const alvo = post.etapa === 5 ? 2 : post.etapa; // na aprovação final, edita-se o texto
  const loja = useLoja();
  const [pauta, setPauta] = useState<Pauta>(post.pauta);
  const [texto, setTexto] = useState<Texto>(post.texto ?? { legenda: "", cta: "", hashtags: "" });
  const [visual, setVisual] = useState<Visual>(post.visual ?? { slides: [] });
  const salvar = () => {
    if (alvo === 1) loja.editarManual(post.id, "pauta", pauta);
    if (alvo === 2) loja.editarManual(post.id, "texto", texto);
    if (alvo === 3) loja.editarManual(post.id, "visual", visual);
    aoSalvar();
  };
  return (
    <div className="stack">
      {alvo === 1 && (
        <>
          <label className="campo">Tema<input value={pauta.tema} onChange={(e) => setPauta({ ...pauta, tema: e.target.value })} /></label>
          <label className="campo">Pilar de conteúdo
            <select value={pauta.pilar} onChange={(e) => setPauta({ ...pauta, pilar: e.target.value })}>
              {[...new Set([pauta.pilar, ...org.kit.pilares])].filter(Boolean).map((p) => <option key={p} value={p}>{p}</option>)}
            </select>
          </label>
          <label className="campo">Objetivo
            <select value={pauta.objetivo} onChange={(e) => setPauta({ ...pauta, objetivo: e.target.value as Objetivo })}>
              {OBJETIVOS.map((o) => <option key={o.id} value={o.id}>{o.nome} — {o.explica}</option>)}
            </select>
          </label>
          <div className="row">
            {REDES.map((r) => (
              <label key={r.id} className="row"><input type="checkbox" checked={pauta.redes.includes(r.id)} onChange={(e) => setPauta({ ...pauta, redes: e.target.checked ? [...pauta.redes, r.id] : pauta.redes.filter((x) => x !== r.id) })} />{r.nome}</label>
            ))}
          </div>
          <label className="campo">Formato
            <select value={pauta.formato} onChange={(e) => setPauta({ ...pauta, formato: e.target.value as Formato })}>
              <option value="imagem">Imagem com legenda</option><option value="carrossel">Carrossel</option>
            </select>
          </label>
        </>
      )}
      {alvo === 2 && (
        <>
          <label className="campo">Legenda<textarea style={{ minHeight: 180 }} value={texto.legenda} onChange={(e) => setTexto({ ...texto, legenda: e.target.value })} /></label>
          <label className="campo">Chamada para ação (CTA)<input value={texto.cta} onChange={(e) => setTexto({ ...texto, cta: e.target.value })} placeholder="Ex.: Salve este post para consultar depois." /></label>
          <label className="campo">Hashtags<input value={texto.hashtags} onChange={(e) => setTexto({ ...texto, hashtags: e.target.value })} /></label>
        </>
      )}
      {alvo === 3 && visual.slides.map((s, i) => (
        <label key={i} className="campo">Texto da arte {i + 1}
          <input value={s.titulo} onChange={(e) => setVisual({ slides: visual.slides.map((x, j) => (j === i ? { ...x, titulo: e.target.value } : x)) })} />
        </label>
      ))}
      <div className="row">
        <button className="btn primario" disabled={alvo === 1 && (!pauta.tema.trim() || !pauta.redes.length)} onClick={salvar}>Salvar edição</button>
        <button className="btn" onClick={aoSalvar}>Cancelar</button>
      </div>
    </div>
  );
}
