"use client";
import { nomeTipoOrganizacao } from "@laura-growth/contratos";
import { useOrganizacaoAtiva, useSessao } from "@/lib/api";

export default function Inicio() {
  const { data: sessao } = useSessao();
  const org = useOrganizacaoAtiva(sessao?.organizacaoAtiva.id ?? "");
  if (!sessao) return null;

  return (
    <>
      <div className="page-header">
        <div>
          <h1>Olá, {sessao.usuario.nome}</h1>
          <p>Você está trabalhando na organização <b>{sessao.organizacaoAtiva.nome}</b>.</p>
        </div>
      </div>
      <section className="card stack" aria-busy={org.isPending}>
        <h2 style={{ margin: 0 }}>Organização ativa</h2>
        {org.isPending && <p className="muted">Carregando…</p>}
        {org.isError && <p role="alert" style={{ color: "var(--laura-danger)" }}>{org.error.message}</p>}
        {org.data && (
          <dl className="grid g3" style={{ margin: 0 }}>
            <div><dt className="small muted">Nome</dt><dd style={{ margin: 0 }}><b>{org.data.nome}</b></dd></div>
            <div><dt className="small muted">Tipo</dt><dd style={{ margin: 0 }}>{nomeTipoOrganizacao[org.data.tipo]}</dd></div>
            <div><dt className="small muted">Seu papel</dt><dd style={{ margin: 0 }}>{org.data.papel === "dono" ? "Dono" : "Membro"}</dd></div>
            <div><dt className="small muted">Pessoas com acesso</dt><dd style={{ margin: 0 }} className="num">{org.data.membros}</dd></div>
            <div><dt className="small muted">Criada em</dt><dd style={{ margin: 0 }}>{new Date(org.data.criadaEm).toLocaleDateString("pt-BR")}</dd></div>
          </dl>
        )}
      </section>
      <p className="small muted" style={{ marginTop: 16 }}>
        Em construção: acesso, organizações e kit de marca prontos. Criação de posts chega na etapa 3.
      </p>
    </>
  );
}
