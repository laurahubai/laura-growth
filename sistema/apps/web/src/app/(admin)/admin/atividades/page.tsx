"use client";
import { useEffect, useState } from "react";
import { Loader2, Play, Save } from "lucide-react";
import {
  nomeAtividade,
  servicoAtividade,
  type Atividade,
  type Modelo,
  type RotaAtividade,
} from "@laura-growth/contratos";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import {
  FalhaApi,
  useAtividadesAdmin,
  useFornecedoresAdmin,
  useSalvarAtividade,
  useTestarAtividade,
  useTodosModelosAdmin,
} from "@/lib/api";
import { formatarUsd } from "@/lib/formato";

type SelecaoRota = { principal: string; reserva: string };

export default function PaginaAtividades() {
  const atividades = useAtividadesAdmin();
  const fornecedores = useFornecedoresAdmin();
  const salvar = useSalvarAtividade();
  const testar = useTestarAtividade();

  const listaFornecedores = fornecedores.data?.fornecedores ?? [];
  const fornecedorIds = listaFornecedores.map((f) => f.id);
  const consultasModelos = useTodosModelosAdmin(fornecedorIds);

  const [rascunhos, setRascunhos] = useState<Record<string, SelecaoRota>>({});
  const [salvandoAtividade, setSalvandoAtividade] = useState<Atividade | null>(null);
  const [erroLinha, setErroLinha] = useState<Record<string, string>>({});
  const [salvoLinha, setSalvoLinha] = useState<Record<string, boolean>>({});
  const [testandoAtividade, setTestandoAtividade] = useState<Atividade | null>(null);

  const erroModelos = consultasModelos.find((q) => q.error)?.error;
  const bloqueado = useRedirecionarSemPermissao(
    atividades.error || fornecedores.error || erroModelos || salvar.error || testar.error,
  );

  useEffect(() => {
    if (!atividades.data) return;
    setRascunhos((anterior) => {
      const proximo: Record<string, SelecaoRota> = { ...anterior };
      for (const r of atividades.data.atividades) {
        if (!proximo[r.atividade]) {
          proximo[r.atividade] = {
            principal: r.principal?.id ?? "",
            reserva: r.reserva?.id ?? "",
          };
        }
      }
      return proximo;
    });
  }, [atividades.data]);

  if (bloqueado) return <div className="vazio" role="alert">Você não tem acesso à Administração.</div>;

  const fornecedoresAtivos = new Map(listaFornecedores.map((f) => [f.id, f.ativo]));
  const modelosAtivos: Modelo[] = [];
  for (const consulta of consultasModelos) {
    if (consulta.data) {
      for (const m of consulta.data.modelos) {
        if (m.ativo && (fornecedoresAtivos.get(m.fornecedorId) ?? true)) {
          modelosAtivos.push(m);
        }
      }
    }
  }
  modelosAtivos.sort((a, b) =>
    `${a.fornecedorNome} ${a.identificador}`.localeCompare(`${b.fornecedorNome} ${b.identificador}`, "pt-BR"),
  );

  const modelosCarregando = fornecedores.isPending || consultasModelos.some((q) => q.isPending);

  const mudarSelecao = (atividade: Atividade, campo: "principal" | "reserva", valor: string, rota: RotaAtividade) => {
    setErroLinha((prev) => ({ ...prev, [atividade]: "" }));
    setSalvoLinha((prev) => ({ ...prev, [atividade]: false }));
    setRascunhos((prev) => {
      const atual = prev[atividade] ?? {
        principal: rota.principal?.id ?? "",
        reserva: rota.reserva?.id ?? "",
      };
      return { ...prev, [atividade]: { ...atual, [campo]: valor } };
    });
  };

  const salvarRota = (rota: RotaAtividade) => {
    const sel = rascunhos[rota.atividade] ?? {
      principal: rota.principal?.id ?? "",
      reserva: rota.reserva?.id ?? "",
    };
    setErroLinha((prev) => ({ ...prev, [rota.atividade]: "" }));
    setSalvoLinha((prev) => ({ ...prev, [rota.atividade]: false }));
    setSalvandoAtividade(rota.atividade);

    salvar.mutate(
      {
        atividade: rota.atividade,
        dados: {
          modeloPrincipalId: sel.principal || null,
          modeloReservaId: sel.reserva || null,
        },
      },
      {
        onSuccess: (atualizada) => {
          setRascunhos((prev) => ({
            ...prev,
            [rota.atividade]: {
              principal: atualizada.principal?.id ?? "",
              reserva: atualizada.reserva?.id ?? "",
            },
          }));
          setSalvoLinha((prev) => ({ ...prev, [rota.atividade]: true }));
        },
        onError: (e) => {
          setErroLinha((prev) => ({
            ...prev,
            [rota.atividade]: e instanceof FalhaApi ? e.message : "Não foi possível salvar a atividade.",
          }));
        },
        onSettled: () => setSalvandoAtividade(null),
      },
    );
  };

  const testarRota = (atividade: Atividade) => {
    setTestandoAtividade(atividade);
    testar.mutate(atividade);
  };

  return (
    <div className="stack" style={{ gap: 16 }}>
      <div className="page-header" style={{ marginBottom: 0 }}>
        <div>
          <h1>Atividades de IA</h1>
          <p>Escolha o modelo principal e o modelo reserva de cada atividade da plataforma.</p>
        </div>
      </div>

      {(testar.isPending || testar.isError || testar.isSuccess) && (
        <section
          className="card stack"
          data-testid="resultado-atividade"
          aria-busy={testar.isPending}
        >
          {testar.isPending && (
            <div className="row">
              <Loader2 size={18} className="girar" aria-hidden />
              <span>Executando teste da atividade…</span>
            </div>
          )}
          {testar.isError && (
            <div className="alerta t-danger" role="alert">
              {testar.error.message}
            </div>
          )}
          {testar.isSuccess && (
            <>
              <div className="row" style={{ justifyContent: "space-between" }}>
                <div className="row">
                  <h2 style={{ margin: 0 }}>
                    Resultado da atividade: {nomeAtividade[testar.data.atividade]}
                  </h2>
                  {testar.data.atendidoPelaReserva && (
                    <span className="badge t-warning">Atendida pela reserva</span>
                  )}
                  {testar.data.simulado && <span className="badge t-ai">Simulado</span>}
                </div>
                <span className="small muted">
                  Atendida por: <b>{testar.data.atendidoPor.fornecedorNome}</b> —{" "}
                  <span className="num">{testar.data.atendidoPor.identificador}</span>
                </span>
              </div>

              <div className="stack" style={{ gap: 6 }}>
                <h3 style={{ margin: 0 }}>Tentativas</h3>
                <ul className="atividade">
                  {testar.data.tentativas.map((t, idx) => (
                    <li key={`${t.modelo.id}-${idx}`} className="row">
                      <span className="num" style={{ fontWeight: 500 }}>
                        {t.modelo.fornecedorNome} — {t.modelo.identificador}
                      </span>
                      {t.status === "ok" && <span className="badge t-success">ok</span>}
                      {t.status === "falha" && (
                        <span className="badge t-danger">falhou: {t.erro ?? "erro desconhecido"}</span>
                      )}
                      {t.status === "pulado_limite" && (
                        <span className="badge t-warning">pulado: limite</span>
                      )}
                    </li>
                  ))}
                </ul>
              </div>

              {testar.data.texto !== null && <div className="legenda">{testar.data.texto}</div>}
              {testar.data.imagemDataUrl !== null && (
                <div>
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={testar.data.imagemDataUrl}
                    alt="Imagem gerada no teste"
                    style={{ maxWidth: 320, borderRadius: 8, border: "1px solid var(--laura-border)" }}
                  />
                </div>
              )}

              <dl className="grid g4" style={{ margin: 0 }}>
                <div>
                  <dt className="small muted">Tokens de entrada</dt>
                  <dd style={{ margin: 0 }} className="num">
                    {testar.data.tokensEntrada}
                  </dd>
                </div>
                <div>
                  <dt className="small muted">Tokens de saída</dt>
                  <dd style={{ margin: 0 }} className="num">
                    {testar.data.tokensSaida}
                  </dd>
                </div>
                <div>
                  <dt className="small muted">Imagens</dt>
                  <dd style={{ margin: 0 }} className="num">
                    {testar.data.imagens}
                  </dd>
                </div>
                <div>
                  <dt className="small muted">Custo</dt>
                  <dd style={{ margin: 0 }} className="num">
                    <b>{formatarUsd(testar.data.custoUsd)}</b>
                  </dd>
                </div>
              </dl>
            </>
          )}
        </section>
      )}

      <section className="card" aria-busy={atividades.isPending}>
        {atividades.isPending && <p className="vazio">Carregando atividades…</p>}
        {atividades.isError && (
          <div className="alerta t-danger" role="alert">
            {atividades.error.message}
          </div>
        )}
        {fornecedores.isError && (
          <div className="alerta t-danger" role="alert" style={{ marginBottom: 12 }}>
            {fornecedores.error.message}
          </div>
        )}
        {atividades.data && atividades.data.atividades.length === 0 && (
          <p className="vazio">Nenhuma atividade configurada.</p>
        )}
        {atividades.data && atividades.data.atividades.length > 0 && (
          <div className="rolagem">
            <table>
              <thead>
                <tr>
                  <th>Atividade</th>
                  <th>Principal</th>
                  <th>Reserva</th>
                  <th>Situação</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {atividades.data.atividades.map((r) => {
                  const servico = servicoAtividade[r.atividade];
                  const nome = nomeAtividade[r.atividade] ?? r.nome;
                  const opcoes = modelosAtivos.filter((m) => m.servico === servico);
                  const sel = rascunhos[r.atividade] ?? {
                    principal: r.principal?.id ?? "",
                    reserva: r.reserva?.id ?? "",
                  };
                  const principalInativo =
                    r.principal && !opcoes.some((m) => m.id === r.principal?.id) ? r.principal : null;
                  const reservaInativa =
                    r.reserva && !opcoes.some((m) => m.id === r.reserva?.id) ? r.reserva : null;

                  const salvandoEsta = salvar.isPending && salvandoAtividade === r.atividade;
                  const testandoEsta = testar.isPending && testandoAtividade === r.atividade;

                  return (
                    <tr key={r.atividade} data-testid={`atividade-${r.atividade}`}>
                      <td>
                        <div className="stack" style={{ gap: 4 }}>
                          <span style={{ fontWeight: 600 }}>{nome}</span>
                          <span className="small muted">
                            Serviço: {servico === "imagem" ? "Imagem" : "Texto"}
                          </span>
                        </div>
                      </td>
                      <td>
                        <select
                          aria-label={`Principal de ${nome}`}
                          value={sel.principal}
                          disabled={modelosCarregando}
                          onChange={(e) => mudarSelecao(r.atividade, "principal", e.target.value, r)}
                        >
                          <option value="">Selecione…</option>
                          {principalInativo && (
                            <option value={principalInativo.id} disabled>
                              {principalInativo.fornecedorNome} — {principalInativo.identificador} (inativo)
                            </option>
                          )}
                          {opcoes.map((m) => (
                            <option key={m.id} value={m.id}>
                              {m.fornecedorNome} — {m.identificador}
                              {m.simulado ? " (Simulado)" : ""}
                            </option>
                          ))}
                        </select>
                      </td>
                      <td>
                        <select
                          aria-label={`Reserva de ${nome}`}
                          value={sel.reserva}
                          disabled={modelosCarregando}
                          onChange={(e) => mudarSelecao(r.atividade, "reserva", e.target.value, r)}
                        >
                          <option value="">Selecione…</option>
                          {reservaInativa && (
                            <option value={reservaInativa.id} disabled>
                              {reservaInativa.fornecedorNome} — {reservaInativa.identificador} (inativo)
                            </option>
                          )}
                          {opcoes.map((m) => (
                            <option key={m.id} value={m.id}>
                              {m.fornecedorNome} — {m.identificador}
                              {m.simulado ? " (Simulado)" : ""}
                            </option>
                          ))}
                        </select>
                      </td>
                      <td>
                        <div className="stack" style={{ gap: 4 }}>
                          {r.pronta ? (
                            <span className="badge t-success">Pronta</span>
                          ) : (
                            <span className="badge t-warning">
                              Não roda: falta {r.falta.join(", ")}
                            </span>
                          )}
                          {erroLinha[r.atividade] && (
                            <span className="small" role="alert" style={{ color: "var(--laura-danger)" }}>
                              {erroLinha[r.atividade]}
                            </span>
                          )}
                          {salvoLinha[r.atividade] && !erroLinha[r.atividade] && (
                            <span className="small" role="status" style={{ color: "var(--laura-success)" }}>
                              Salvo.
                            </span>
                          )}
                        </div>
                      </td>
                      <td>
                        <div className="row" style={{ flexWrap: "nowrap" }}>
                          <button
                            type="button"
                            className="btn peq"
                            disabled={salvandoEsta}
                            onClick={() => salvarRota(r)}
                          >
                            <Save size={14} aria-hidden /> {salvandoEsta ? "Salvando…" : "Salvar"}
                          </button>
                          <button
                            type="button"
                            className="btn peq primario"
                            disabled={testar.isPending}
                            onClick={() => testarRota(r.atividade)}
                          >
                            {testandoEsta ? (
                              <Loader2 size={14} className="girar" aria-hidden />
                            ) : (
                              <Play size={14} aria-hidden />
                            )}
                            Testar atividade
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
