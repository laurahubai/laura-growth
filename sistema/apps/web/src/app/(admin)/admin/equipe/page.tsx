"use client";
import { useState } from "react";
import { ShieldCheck, ShieldOff } from "lucide-react";
import type { MembroEquipe } from "@laura-growth/contratos";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import { TabelaRegistroAcessos } from "@/components/TabelaRegistroAcessos";
import { FalhaApi, useAtualizarEquipeAdmin, useEquipeAdmin, useSessao } from "@/lib/api";

export default function PaginaEquipeAdmin() {
  const { data: sessao } = useSessao();
  const equipe = useEquipeAdmin();
  const atualizar = useAtualizarEquipeAdmin();
  const [erroAcao, setErroAcao] = useState("");

  const bloqueado = useRedirecionarSemPermissao(equipe.error || atualizar.error);
  if (bloqueado) return <div className="vazio" role="alert">{bloqueado}</div>;

  const alternarAdmin = (m: MembroEquipe) => {
    if (m.adminPlataforma) {
      const ehVoce = m.usuarioId === sessao?.usuario.id;
      const textoConfirmacao = ehVoce
        ? "Retirar seu próprio acesso de administrador da plataforma? Você perderá acesso imediato à Administração."
        : `Retirar o acesso de administrador de "${m.nome}" (${m.email})?`;
      if (!confirm(textoConfirmacao)) return;
    }
    setErroAcao("");
    atualizar.mutate(
      { usuarioId: m.usuarioId, dados: { adminPlataforma: !m.adminPlataforma } },
      {
        onError: (e) => {
          setErroAcao(
            e instanceof FalhaApi
              ? e.message
              : "Não foi possível atualizar a permissão administrativa.",
          );
        },
      },
    );
  };

  return (
    <div className="stack" style={{ gap: 20 }}>
      <div className="page-header" style={{ marginBottom: 0 }}>
        <div>
          <h1>Equipe da plataforma</h1>
          <p>
            Apenas pessoas das organizações da casa (Laura Growth e Laura Hub) podem administrar a plataforma.
          </p>
        </div>
      </div>

      {erroAcao && (
        <div className="alerta t-danger" role="alert">
          {erroAcao}
        </div>
      )}

      <section className="card stack" data-testid="lista-equipe" aria-busy={equipe.isPending}>
        <h2 style={{ margin: 0 }}>Membros e administradores</h2>
        {equipe.isPending && <p className="vazio">Carregando equipe…</p>}
        {equipe.isError && (
          <div className="alerta t-danger" role="alert">
            {equipe.error.message}
          </div>
        )}
        {equipe.data && equipe.data.equipe.length === 0 && (
          <p className="vazio">Nenhuma pessoa elegível encontrada.</p>
        )}
        {equipe.data && equipe.data.equipe.length > 0 && (
          <div className="rolagem">
            <table>
              <thead>
                <tr>
                  <th>Pessoa</th>
                  <th>E-mail</th>
                  <th>Organizações da casa</th>
                  <th>Administração</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {equipe.data.equipe.map((m) => {
                  const ehVoce = m.usuarioId === sessao?.usuario.id;
                  return (
                    <tr key={m.usuarioId} data-testid={`equipe-${m.email}`}>
                      <td>
                        <b>{m.nome}</b> {ehVoce && <span className="small muted">(você)</span>}
                      </td>
                      <td className="small">{m.email}</td>
                      <td>
                        {m.organizacoes.length === 0 ? (
                          <span className="small muted">Nenhuma</span>
                        ) : (
                          <div className="row" style={{ gap: 4 }}>
                            {m.organizacoes.map((org) => (
                              <span key={org} className="badge t-neutral">{org}</span>
                            ))}
                          </div>
                        )}
                      </td>
                      <td>
                        <span className={`badge ${m.adminPlataforma ? "t-brand" : "t-neutral"}`}>
                          {m.adminPlataforma ? "Administrador" : "Não é administrador"}
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={`btn peq ${m.adminPlataforma ? "perigo" : ""}`}
                          disabled={atualizar.isPending}
                          onClick={() => alternarAdmin(m)}
                        >
                          {m.adminPlataforma ? (
                            <>
                              <ShieldOff size={14} aria-hidden /> Retirar acesso de administrador
                            </>
                          ) : (
                            <>
                              <ShieldCheck size={14} aria-hidden /> Dar acesso de administrador
                            </>
                          )}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="card stack" data-testid="auditoria-equipe">
        <h2 style={{ margin: 0 }}>Registro de alterações</h2>
        {equipe.isPending && <p className="muted" style={{ margin: 0 }}>Carregando histórico…</p>}
        {equipe.data && (
          <TabelaRegistroAcessos
            registros={equipe.data.registros}
            vazio="Nenhuma alteração administrativa registrada ainda."
          />
        )}
      </section>
    </div>
  );
}
