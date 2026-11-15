"use client";
// Fila de aprovação (APR-04): aprovar ou devolver vários posts em sequência. A aprovação continua post a post.
import Link from "next/link";
import { useState } from "react";
import { Check, MessageSquare, SkipForward } from "lucide-react";
import { useLoja, usePostsDaOrg } from "@/mock/loja";
import { fmtData, hoje, nomeFormato, nomeRede } from "@/mock/regras";
import { EtiquetasPauta, PageHeader, Previas } from "@/components/ui";

export default function Aprovacoes() {
  const loja = useLoja();
  const { org } = loja;
  const posts = usePostsDaOrg();
  const [pulados, setPulados] = useState<string[]>([]);
  const [devolvendo, setDevolvendo] = useState(false);
  const [comentario, setComentario] = useState("");
  const [feitos, setFeitos] = useState({ aprovados: 0, devolvidos: 0 });
  if (!org) return null;

  const fila = posts.filter((p) => p.etapa === 5 && !p.descartado).sort((a, b) => a.data.localeCompare(b.data));
  const restantes = fila.filter((p) => !pulados.includes(p.id));
  const post = restantes[0];
  const emCriacao = posts.filter((p) => p.etapa < 5 && !p.descartado && p.trabalhando).length;
  const limpar = () => { setDevolvendo(false); setComentario(""); };

  return (
    <>
      <PageHeader
        titulo="Aprovações"
        descricao={`Posts da ${org.nome} aguardando a sua aprovação final, um após o outro.`}
        acoes={<span className="badge t-ai num">{restantes.length} na fila</span>}
      />
      {(feitos.aprovados > 0 || feitos.devolvidos > 0) && (
        <p className="small muted" role="status">Nesta sessão: <b className="num">{feitos.aprovados}</b> aprovados · <b className="num">{feitos.devolvidos}</b> devolvidos.</p>
      )}

      {!post ? (
        <section className="card vazio stack" style={{ alignItems: "center" }}>
          <p style={{ margin: 0 }}>{fila.length > 0 ? "Você pulou os posts que restam na fila." : "Nenhum post aguardando aprovação."}</p>
          {emCriacao > 0 && <p className="small" style={{ margin: 0 }}>O agente está criando {emCriacao} post(s). Eles aparecem aqui quando ficarem prontos.</p>}
          <div className="row">
            {fila.length > 0 && <button className="btn primario" onClick={() => setPulados([])}>Rever os pulados</button>}
            <Link href="/calendario" className="btn">Ver calendário</Link>
          </div>
        </section>
      ) : (
        <section className="card stack">
          <div className="row" style={{ justifyContent: "space-between" }}>
            <div>
              <h2 style={{ margin: 0 }}>{post.pauta.tema}</h2>
              <p className="small muted" style={{ margin: "4px 0 0" }}>
                {nomeFormato(post.pauta.formato)} · {post.pauta.redes.map(nomeRede).join(", ")} · previsto para {fmtData(post.data)}
              </p>
            </div>
            <EtiquetasPauta pauta={post.pauta} />
          </div>
          {post.data < hoje() && <div className="alerta t-warning">A data prevista já passou. Depois de aprovar, reagende na tela do post.</div>}
          {post.revisao?.avisos.filter((a) => !a.startsWith("Nenhum")).map((a, i) => <div key={i} className="alerta t-warning">{a}</div>)}

          <Previas post={post} org={org} />
          <div><span className="small muted">Chamada para ação (CTA)</span><div><b>{post.texto?.cta || "Sem CTA"}</b></div></div>

          <div className="row" style={{ borderTop: "1px solid var(--laura-border)", paddingTop: 12 }}>
            <button className="btn primario" onClick={() => { loja.aprovarEtapa(post.id); setFeitos((f) => ({ ...f, aprovados: f.aprovados + 1 })); limpar(); }}>
              <Check size={16} aria-hidden /> Aprovar e ver o próximo
            </button>
            <button className="btn" onClick={() => setDevolvendo(!devolvendo)}>
              <MessageSquare size={16} aria-hidden /> Devolver com comentário
            </button>
            <button className="btn" onClick={() => { setPulados([...pulados, post.id]); limpar(); }}>
              <SkipForward size={16} aria-hidden /> Pular por agora
            </button>
            <Link href={`/posts/${post.id}`} className="small" style={{ marginLeft: "auto" }}>Abrir o post completo</Link>
          </div>
          {devolvendo && (
            <div className="stack">
              <label className="campo">
                O que você quer mudar?
                <textarea value={comentario} onChange={(e) => setComentario(e.target.value)} placeholder="Ex.: deixe mais curto e mais leve" autoFocus />
              </label>
              <div className="row">
                <button className="btn primario" disabled={!comentario.trim()} onClick={() => { loja.devolver(post.id, comentario); setFeitos((f) => ({ ...f, devolvidos: f.devolvidos + 1 })); limpar(); }}>
                  Devolver ao agente e ver o próximo
                </button>
                <button className="btn" onClick={limpar}>Cancelar</button>
              </div>
            </div>
          )}
          <p className="small muted">Nada vai ao ar sem esta aprovação. Cada post é aprovado individualmente.</p>
        </section>
      )}
    </>
  );
}
