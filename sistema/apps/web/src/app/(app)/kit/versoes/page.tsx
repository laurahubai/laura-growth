"use client";
// Etapa 2C: histórico de versões do kit de marca (RF-10, C4).
import Link from "next/link";
import { useState } from "react";
import { ArrowLeft, Bot, User } from "lucide-react";
import { rotuloCampoKit } from "@/components/KitCompartilhado";
import { SemAcesso } from "@/components/SemAcesso";
import { erroSemPermissao, useSessao, useVersoesKit } from "@/lib/api";
import { formatarDataHora } from "@/lib/formato";

const POR_PAGINA = 20;

export default function VersoesKit() {
  const { data: sessao } = useSessao();
  const orgId = sessao?.organizacaoAtiva.id ?? "";
  const [pagina, setPagina] = useState(1);
  const versoes = useVersoesKit(orgId, pagina);

  if (!sessao) return null;
  if (versoes.isPending) return <p className="muted">Carregando…</p>;
  if (erroSemPermissao(versoes.error)) {
    return (
      <SemAcesso
        mensagem={
          versoes.error instanceof Error
            ? versoes.error.message
            : "Você não tem permissão para ver as versões do kit de marca."
        }
      />
    );
  }
  if (versoes.isError) {
    return (
      <p role="alert" style={{ color: "var(--laura-danger)" }}>
        {versoes.error.message}
      </p>
    );
  }

  const { versoes: lista, total } = versoes.data;
  const totalPaginas = Math.max(1, Math.ceil(total / POR_PAGINA));

  return (
    <div className="stack" style={{ gap: 16 }}>
      <div className="page-header" style={{ marginBottom: 0 }}>
        <div>
          <h1>Versões do kit de marca</h1>
          <p>
            Histórico de alterações de <b>{sessao.organizacaoAtiva.nome}</b>.
          </p>
        </div>
        <Link href="/kit" className="btn">
          <ArrowLeft size={16} aria-hidden /> Voltar para o kit
        </Link>
      </div>

      <section className="card stack">
        {lista.length === 0 ? (
          <p className="muted" style={{ margin: 0 }}>
            Nenhuma versão registrada ainda.
          </p>
        ) : (
          <div className="rolagem">
            <table data-testid="tabela-versoes-kit">
              <thead>
                <tr>
                  <th>Versão</th>
                  <th>Quem</th>
                  <th>Quando</th>
                  <th>Campos alterados</th>
                  <th className="d">Ação</th>
                </tr>
              </thead>
              <tbody>
                {lista.map((v) => (
                  <tr key={v.numero} data-testid={`versao-${v.numero}`}>
                    <td className="num">
                      <b>Versão {v.numero}</b>
                    </td>
                    <td>
                      {v.autorTipo === "ia" ? (
                        <span className="row" style={{ gap: 6 }}>
                          <span className="badge t-ai">
                            <Bot size={12} aria-hidden /> IA
                          </span>
                          {v.autorNome && <span className="small">{v.autorNome}</span>}
                        </span>
                      ) : (
                        <span className="row" style={{ gap: 6 }}>
                          <User size={14} aria-hidden className="muted" />
                          <span>{v.autorNome ?? "Pessoa"}</span>
                        </span>
                      )}
                    </td>
                    <td className="num small">{formatarDataHora(v.criadoEm)}</td>
                    <td>
                      {v.camposAlterados.length === 0 ? (
                        <span className="small muted">—</span>
                      ) : (
                        <div className="row" style={{ gap: 4 }}>
                          {v.camposAlterados.map((c) => (
                            <span key={c} className="badge t-neutral">
                              {rotuloCampoKit(c)}
                            </span>
                          ))}
                        </div>
                      )}
                    </td>
                    <td className="d">
                      <Link href={`/kit/versoes/${v.numero}`} className="btn peq">
                        Ver versão
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {total > POR_PAGINA && (
          <div className="row" style={{ justifyContent: "space-between", paddingTop: 8 }}>
            <span className="small muted num">
              Página {pagina} de {totalPaginas} ({total} versões)
            </span>
            <div className="row">
              <button
                type="button"
                className="btn peq"
                disabled={pagina <= 1}
                onClick={() => setPagina((p) => Math.max(1, p - 1))}
              >
                Anterior
              </button>
              <button
                type="button"
                className="btn peq"
                disabled={pagina >= totalPaginas}
                onClick={() => setPagina((p) => Math.min(totalPaginas, p + 1))}
              >
                Próxima
              </button>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
