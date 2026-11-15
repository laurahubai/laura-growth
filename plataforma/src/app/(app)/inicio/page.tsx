"use client";
import Link from "next/link";
import { AlertTriangle, Clock } from "lucide-react";
import { useLoja, usePostsDaOrg } from "@/mock/loja";
import { estadoDoPost, faltasDeCadencia, fmtData, nomeRede } from "@/mock/regras";
import { PageHeader, StatusBadge } from "@/components/ui";

export default function Inicio() {
  const { org, dados } = useLoja();
  const posts = usePostsDaOrg();
  if (!org || !dados) return null;
  const comEstado = posts.map((p) => ({ p, e: estadoDoPost(p, org) }));
  const conta = (e: string) => comEstado.filter((x) => x.e === e).length;
  const atencao = comEstado.filter((x) => ["aguardando_aprovacao", "atrasado", "falhou", "pronto_manual"].includes(x.e));
  const faltas = faltasDeCadencia(org, posts);

  return (
    <>
      <PageHeader
        titulo={`Início — ${org.nome}`}
        descricao="O que precisa da sua atenção hoje."
        acoes={<Link href="/posts?novo=1" className="btn primario">Novo post</Link>}
      />
      <div className="grid g4" style={{ marginBottom: 16 }}>
        {[
          ["Aguardando aprovação", conta("aguardando_aprovacao")],
          ["Em criação", conta("em_criacao")],
          ["Atrasados", conta("atrasado")],
          ["Publicados", conta("publicado")],
        ].map(([r, v]) => (
          <div key={r} className="card metric">
            <div className="small muted">{r}</div>
            <div className="valor">{v}</div>
          </div>
        ))}
      </div>
      <div className="grid g2">
        <section className="card">
          <h2>Precisa de você</h2>
          {atencao.length === 0 ? (
            <p className="vazio">Nada pendente. Que tal criar um novo post?</p>
          ) : (
            <table>
              <tbody>
                {atencao.map(({ p, e }) => (
                  <tr key={p.id}>
                    <td><Link href={`/posts/${p.id}`}>{p.pauta.tema}</Link></td>
                    <td className="num muted">{fmtData(p.data)}</td>
                    <td><StatusBadge estado={e} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
        <section className="card stack">
          <h2 style={{ margin: 0 }}>Cadência desta semana</h2>
          {faltas.length === 0 ? (
            <p className="muted">Cadência em dia em todas as redes.</p>
          ) : (
            faltas.map((f) => (
              <div key={f.rede} className="alerta t-warning">
                <AlertTriangle size={16} aria-hidden />
                <span>
                  {nomeRede(f.rede)}: <b className="num">{f.tem}</b> de <b className="num">{f.meta}</b> posts na semana. Faltam {f.meta - f.tem}.
                </span>
              </div>
            ))
          )}
          <p className="small muted row"><Clock size={14} aria-hidden /> Aviso de aprovação pendente aparece no sino da barra superior (canal externo: a definir).</p>
        </section>
      </div>
    </>
  );
}
