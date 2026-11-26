"use client";
import Link from "next/link";
import { Users } from "lucide-react";
import { nomeTipoOrganizacao } from "@laura-growth/contratos";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import { useOrganizacoesAdmin } from "@/lib/api";

export default function PaginaOrganizacoesAdmin() {
  const organizacoes = useOrganizacoesAdmin();

  const bloqueado = useRedirecionarSemPermissao(organizacoes.error);
  if (bloqueado) return <div className="vazio" role="alert">{bloqueado}</div>;

  return (
    <div className="stack" style={{ gap: 20 }}>
      <div className="page-header" style={{ marginBottom: 0 }}>
        <div>
          <h1>Organizações</h1>
          <p>Gerencie pessoas, alçadas e convites de qualquer organização da plataforma.</p>
        </div>
      </div>

      <section className="card stack" data-testid="lista-organizacoes" aria-busy={organizacoes.isPending}>
        {organizacoes.isPending && <p className="vazio">Carregando organizações…</p>}
        {organizacoes.isError && (
          <div className="alerta t-danger" role="alert">
            {organizacoes.error.message}
          </div>
        )}
        {organizacoes.data && organizacoes.data.organizacoes.length === 0 && (
          <p className="vazio">Nenhuma organização cadastrada ainda.</p>
        )}
        {organizacoes.data && organizacoes.data.organizacoes.length > 0 && (
          <div className="rolagem">
            <table>
              <thead>
                <tr>
                  <th>Organização</th>
                  <th>Tipo</th>
                  <th className="d">Pessoas</th>
                  <th>Criada em</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {organizacoes.data.organizacoes.map((org) => (
                  <tr key={org.id} data-testid={`org-${org.id}`}>
                    <td>
                      <div className="row" style={{ gap: 6 }}>
                        <Link href={`/admin/organizacoes/${org.id}`} style={{ fontWeight: 600 }}>
                          {org.nome}
                        </Link>
                        {org.daCasa && <span className="badge t-brand">Da casa</span>}
                      </div>
                    </td>
                    <td>{nomeTipoOrganizacao[org.tipo]}</td>
                    <td className="d">{org.membros}</td>
                    <td className="num small">{new Date(org.criadaEm).toLocaleDateString("pt-BR")}</td>
                    <td>
                      <Link href={`/admin/organizacoes/${org.id}`} className="btn peq">
                        <Users size={14} aria-hidden /> Gerenciar pessoas
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
