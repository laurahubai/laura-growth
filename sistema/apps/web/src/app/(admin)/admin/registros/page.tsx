"use client";
import { RefreshCw } from "lucide-react";
import {
  nomeAtividade,
  nomeBaseCalculoPreco,
  type Atividade,
  type Execucao,
} from "@laura-growth/contratos";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import { useExecucoesAdmin } from "@/lib/api";
import { formatarDataHora, formatarUsd } from "@/lib/formato";

function rotuloAtividade(valor: string): string {
  if (valor === "teste") return "Teste";
  return nomeAtividade[valor as Atividade] ?? valor;
}

function badgeStatus(status: Execucao["status"]) {
  if (status === "ok") return <span className="badge t-success">Ok</span>;
  if (status === "pulado_limite") return <span className="badge t-warning">Pulado (limite)</span>;
  return <span className="badge t-danger">Falha</span>;
}

export default function PaginaRegistros() {
  const execucoes = useExecucoesAdmin(100);
  const bloqueado = useRedirecionarSemPermissao(execucoes.error);
  if (bloqueado) return <div className="vazio" role="alert">{bloqueado}</div>;

  return (
    <>
      <div className="page-header">
        <div>
          <h1>Registros de execuções</h1>
          <p>Últimas 100 chamadas de IA registradas na plataforma, com tokens e custo em US$.</p>
        </div>
        <button
          type="button"
          className="btn"
          disabled={execucoes.isFetching}
          onClick={() => execucoes.refetch()}
        >
          <RefreshCw size={16} className={execucoes.isFetching ? "girar" : undefined} aria-hidden />
          {execucoes.isFetching ? "Atualizando…" : "Atualizar"}
        </button>
      </div>

      <section className="card" aria-busy={execucoes.isPending}>
        {execucoes.isPending && <p className="vazio">Carregando registros…</p>}
        {execucoes.isError && (
          <div className="alerta t-danger" role="alert">
            {execucoes.error.message}
          </div>
        )}
        {execucoes.data && execucoes.data.execucoes.length === 0 && (
          <p className="vazio">Nenhum registro de execução ainda.</p>
        )}
        {execucoes.data && execucoes.data.execucoes.length > 0 && (
          <div className="rolagem">
            <table>
              <thead>
                <tr>
                  <th>Data/hora</th>
                  <th>Atividade</th>
                  <th>Organização</th>
                  <th>Fornecedor</th>
                  <th>Modelo</th>
                  <th className="d">Tentativa</th>
                  <th>Reserva</th>
                  <th>Status</th>
                  <th>Erro</th>
                  <th className="d">Tokens</th>
                  <th className="d">Imagens</th>
                  <th className="d">Custo (US$)</th>
                  <th>Base</th>
                </tr>
              </thead>
              <tbody>
                {execucoes.data.execucoes.map((e) => (
                  <tr key={e.id}>
                    <td className="num small">{formatarDataHora(e.criadoEm)}</td>
                    <td>{rotuloAtividade(e.atividade)}</td>
                    <td className="small">{e.organizacaoNome ?? "—"}</td>
                    <td>{e.fornecedorNome}</td>
                    <td>
                      <div className="row" style={{ gap: 6 }}>
                        <span className="num small">{e.modeloIdentificador}</span>
                        {e.simulado && <span className="badge t-ai">Simulado</span>}
                      </div>
                    </td>
                    <td className="d">{e.tentativa}</td>
                    <td>{e.atendidoPelaReserva ? "Sim" : "Não"}</td>
                    <td>{badgeStatus(e.status)}</td>
                    <td className="small muted">{e.erro ?? "—"}</td>
                    <td className="d small">
                      {e.tokensEntradaImagem !== null ? (
                        <div className="stack" style={{ gap: 2 }}>
                          <span>Texto: {e.tokensEntrada}</span>
                          <span>Img: {e.tokensEntradaImagem}</span>
                          <span>Saída: {e.tokensSaida}</span>
                        </div>
                      ) : (
                        <>
                          {e.tokensEntrada} / {e.tokensSaida}
                        </>
                      )}
                    </td>
                    <td className="d">{e.imagens}</td>
                    <td className="d">
                      {formatarUsd(e.custoUsd)}
                      {e.custoEstimado && <span className="small muted"> (estimado)</span>}
                    </td>
                    <td className="small">
                      {e.basePreco ? nomeBaseCalculoPreco[e.basePreco] : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  );
}
