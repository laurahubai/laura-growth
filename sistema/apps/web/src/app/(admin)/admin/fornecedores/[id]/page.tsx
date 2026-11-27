"use client";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { ArrowLeft, DollarSign, Loader2, Play, Plus, RefreshCw, Trash2, X } from "lucide-react";
import {
  nomeTipoFornecedor,
  nomeUnidadePreco,
  PedidoEditarModelo,
  PedidoNovoModelo,
  PedidoNovoPreco,
  Servico,
  type Modelo,
  type UnidadePreco,
} from "@laura-growth/contratos";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import {
  FalhaApi,
  useCriarModelo,
  useCriarPreco,
  useEditarModelo,
  useExcluirFornecedor,
  useFornecedoresAdmin,
  useLerModelos,
  useModelosFornecedor,
  usePrecosModelo,
  useTestarModelo,
} from "@/lib/api";
import {
  etiquetaSituacao,
  formatarDataHora,
  formatarDataIso,
  formatarPrecoUsd,
  formatarUsd,
  hojeIso,
  imagemDataUrlSegura,
} from "@/lib/formato";

function sugerirServico(identificador: string): Servico {
  const s = identificador.toLowerCase();
  if (s.includes("imagem") || s.includes("image") || s.includes("dall-e")) return "imagem";
  return "texto";
}

export default function PaginaModelosFornecedor() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const fornecedorId = typeof params?.id === "string" ? params.id : "";

  const fornecedores = useFornecedoresAdmin();
  const modelos = useModelosFornecedor(fornecedorId);
  const lerModelos = useLerModelos(fornecedorId);
  const criarModelo = useCriarModelo(fornecedorId);
  const editarModelo = useEditarModelo(fornecedorId);
  const excluirFornecedor = useExcluirFornecedor();
  const testar = useTestarModelo(fornecedorId);

  const [servicosLidos, setServicosLidos] = useState<Record<string, Servico>>({});
  const [incluindoId, setIncluindoId] = useState<string>("");
  const [erroIncluirLido, setErroIncluirLido] = useState<string>("");
  const [erroAcaoModelo, setErroAcaoModelo] = useState<string>("");
  const [testandoId, setTestandoId] = useState<string>("");
  const [modeloPrecos, setModeloPrecos] = useState<Modelo | null>(null);
  const [modeloEditar, setModeloEditar] = useState<Modelo | null>(null);
  const [confirmandoExclusao, setConfirmandoExclusao] = useState(false);

  const bloqueado = useRedirecionarSemPermissao(
    fornecedores.error ||
      modelos.error ||
      lerModelos.error ||
      criarModelo.error ||
      editarModelo.error ||
      excluirFornecedor.error ||
      testar.error,
  );
  if (bloqueado) return <div className="vazio" role="alert">{bloqueado}</div>;

  const fornecedor = fornecedores.data?.fornecedores.find((f) => f.id === fornecedorId);
  const cadastrados = new Set((modelos.data?.modelos ?? []).map((m) => m.identificador));

  const incluirLido = (identificador: string) => {
    setErroIncluirLido("");
    setIncluindoId(identificador);
    const servico = servicosLidos[identificador] ?? sugerirServico(identificador);
    criarModelo.mutate(
      { identificador, servico, origem: "lido" },
      {
        onError: (e) => {
          setErroIncluirLido(e instanceof FalhaApi ? e.message : "Não foi possível incluir o modelo.");
        },
        onSettled: () => setIncluindoId(""),
      },
    );
  };

  const executarTeste = (modeloId: string) => {
    setTestandoId(modeloId);
    testar.mutate(modeloId);
  };

  const alternarAtivo = (m: Modelo) => {
    setErroAcaoModelo("");
    editarModelo.mutate(
      { id: m.id, dados: { ativo: !m.ativo } },
      {
        onError: (e) => {
          setErroAcaoModelo(e instanceof FalhaApi ? e.message : "Não foi possível atualizar o modelo.");
        },
      },
    );
  };

  return (
    <>
      <div className="stack" style={{ gap: 16 }}>
        <div>
          <Link href="/admin/fornecedores" className="small row" style={{ width: "fit-content" }}>
            <ArrowLeft size={14} aria-hidden /> Voltar para fornecedores
          </Link>
        </div>

        <div className="page-header" style={{ marginBottom: 0 }}>
          <div>
            <div className="row">
              <h1>{fornecedor ? fornecedor.nome : "Modelos do fornecedor"}</h1>
              {fornecedor?.simulado && <span className="badge t-ai">Simulado</span>}
              {fornecedor && (
                <span className={`badge ${fornecedor.ativo ? "t-success" : "t-neutral"}`}>
                  {fornecedor.ativo ? "Ativo" : "Inativo"}
                </span>
              )}
            </div>
            {fornecedor && (
              <p>
                {nomeTipoFornecedor[fornecedor.tipo]}
                {" · "}
                {fornecedor.chaveFinal ? `Chave ••••${fornecedor.chaveFinal}` : "Sem chave"}
                {" · "}
                Gasto no mês: <span className="num">{formatarUsd(fornecedor.gastoMesUsd)}</span> de{" "}
                <span className="num">{formatarUsd(fornecedor.limiteMensalUsd)}</span>
              </p>
            )}
          </div>

          <div className="row">
            <button
              type="button"
              className="btn primario"
              disabled={lerModelos.isPending || !fornecedorId}
              onClick={() => {
                setErroIncluirLido("");
                lerModelos.mutate();
              }}
            >
              {lerModelos.isPending ? (
                <Loader2 size={16} className="girar" aria-hidden />
              ) : (
                <RefreshCw size={16} aria-hidden />
              )}
              {lerModelos.isPending ? "Lendo modelos…" : "Ler modelos disponíveis"}
            </button>
            {fornecedor && (
              <button
                type="button"
                className="btn perigo"
                disabled={excluirFornecedor.isPending}
                onClick={() => {
                  excluirFornecedor.reset();
                  setConfirmandoExclusao(true);
                }}
              >
                <Trash2 size={16} aria-hidden /> Excluir
              </button>
            )}
          </div>
        </div>

        {fornecedores.isSuccess && !fornecedor && (
          <div className="alerta t-danger" role="alert">
            Fornecedor não encontrado.
          </div>
        )}

        {lerModelos.isError && (
          <div className="alerta t-danger" role="alert">
            {lerModelos.error.message}
          </div>
        )}

        {lerModelos.data && (
          <section className="card stack">
            <h2 style={{ margin: 0 }}>Modelos disponíveis no fornecedor</h2>
            {erroIncluirLido && (
              <div className="alerta t-danger" role="alert">
                {erroIncluirLido}
              </div>
            )}
            {lerModelos.data.modelos.length === 0 ? (
              <p className="muted" style={{ margin: 0 }}>
                Nenhum modelo retornado pelo fornecedor.
              </p>
            ) : (
              <div className="rolagem">
                <table>
                  <thead>
                    <tr>
                      <th>Identificador</th>
                      <th>Serviço</th>
                      <th>Situação</th>
                    </tr>
                  </thead>
                  <tbody>
                    {lerModelos.data.modelos.map((m) => {
                      const jaExiste = m.jaCadastrado || cadastrados.has(m.identificador);
                      const servicoSelecionado = servicosLidos[m.identificador] ?? sugerirServico(m.identificador);
                      return (
                        <tr key={m.identificador}>
                          <td className="num" style={{ fontWeight: 500 }}>
                            {m.identificador}
                          </td>
                          <td>
                            {jaExiste ? (
                              <span className="small muted">—</span>
                            ) : (
                              <select
                                aria-label={`Serviço para ${m.identificador}`}
                                value={servicoSelecionado}
                                onChange={(e) =>
                                  setServicosLidos((prev) => ({
                                    ...prev,
                                    [m.identificador]: e.target.value as Servico,
                                  }))
                                }
                              >
                                <option value="texto">Texto</option>
                                <option value="imagem">Imagem</option>
                              </select>
                            )}
                          </td>
                          <td>
                            {jaExiste ? (
                              <span className="badge t-success">Já cadastrado</span>
                            ) : (
                              <button
                                type="button"
                                className="btn peq primario"
                                disabled={criarModelo.isPending && incluindoId === m.identificador}
                                onClick={() => incluirLido(m.identificador)}
                              >
                                <Plus size={14} aria-hidden />
                                {criarModelo.isPending && incluindoId === m.identificador ? "Incluindo…" : "Incluir"}
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        )}

        {(testar.isPending || testar.isError || testar.isSuccess) && (
          <section
            className="card stack"
            data-testid="resultado-teste"
            aria-busy={testar.isPending}
          >
            {testar.isPending && (
              <div className="row">
                <Loader2 size={18} className="girar" aria-hidden />
                <span>Executando teste do modelo…</span>
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
                    <h2 style={{ margin: 0 }}>Resultado do teste</h2>
                    {testar.data.simulado && <span className="badge t-ai">Simulado</span>}
                  </div>
                  <span className="small muted">
                    {testar.data.modelo.fornecedorNome} — {testar.data.modelo.identificador}
                  </span>
                </div>
                {testar.data.texto !== null && <div className="legenda">{testar.data.texto}</div>}
                {testar.data.imagemDataUrl !== null && (
                  <div>
                    {imagemDataUrlSegura(testar.data.imagemDataUrl) ? (
                      <>
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img
                          src={testar.data.imagemDataUrl}
                          alt="Imagem gerada no teste"
                          style={{ maxWidth: 320, borderRadius: 8, border: "1px solid var(--laura-border)" }}
                        />
                      </>
                    ) : (
                      <p className="small muted" role="alert" style={{ margin: 0 }}>
                        imagem em formato não reconhecido
                      </p>
                    )}
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

        <section className="card stack" aria-busy={modelos.isPending}>
          <h2 style={{ margin: 0 }}>Modelos cadastrados</h2>
          {erroAcaoModelo && (
            <div className="alerta t-danger" role="alert">
              {erroAcaoModelo}
            </div>
          )}
          {modelos.isPending && <p className="vazio">Carregando modelos…</p>}
          {modelos.isError && (
            <div className="alerta t-danger" role="alert">
              {modelos.error.message}
            </div>
          )}
          {modelos.data && modelos.data.modelos.length === 0 && (
            <p className="vazio">Nenhum modelo cadastrado ainda.</p>
          )}
          {modelos.data && modelos.data.modelos.length > 0 && (
            <div className="rolagem">
              <table>
                <thead>
                  <tr>
                    <th>Modelo</th>
                    <th>Serviço</th>
                    <th>Origem</th>
                    <th>Preços vigentes</th>
                    <th>Limite próprio</th>
                    <th className="d">Gasto no mês</th>
                    <th>Situação</th>
                    <th>Ativo</th>
                    <th>Ações</th>
                  </tr>
                </thead>
                <tbody>
                  {modelos.data.modelos.map((m) => {
                    const sit = etiquetaSituacao(m.situacao);
                    const testandoEste = testar.isPending && testandoId === m.id;
                    return (
                      <tr key={m.id}>
                        <td>
                          <div className="row" style={{ gap: 6 }}>
                            <span className="num" style={{ fontWeight: 600 }}>
                              {m.identificador}
                            </span>
                            {m.simulado && <span className="badge t-ai">Simulado</span>}
                          </div>
                        </td>
                        <td>{m.servico === "imagem" ? "Imagem" : "Texto"}</td>
                        <td className="small muted">{m.origem === "lido" ? "Lido" : "Manual"}</td>
                        <td className="small num">
                          {m.servico === "texto" ? (
                            <div className="stack" style={{ gap: 2 }}>
                              <span>
                                Entrada:{" "}
                                {m.precosVigentes.tokenEntrada !== null
                                  ? `${formatarPrecoUsd(m.precosVigentes.tokenEntrada)} / 1M`
                                  : "Sem preço"}
                              </span>
                              <span>
                                Saída:{" "}
                                {m.precosVigentes.tokenSaida !== null
                                  ? `${formatarPrecoUsd(m.precosVigentes.tokenSaida)} / 1M`
                                  : "Sem preço"}
                              </span>
                            </div>
                          ) : m.precosVigentes.imagem !== null ? (
                            `${formatarPrecoUsd(m.precosVigentes.imagem)} / imagem`
                          ) : (
                            "Sem preço"
                          )}
                        </td>
                        <td className="small num">
                          {m.limiteMensalUsd !== null
                            ? `${formatarUsd(m.limiteMensalUsd)} (${m.avisoPercentual ?? fornecedor?.avisoPercentual ?? 80}%)`
                            : "Usa o do fornecedor"}
                        </td>
                        <td className="d">{formatarUsd(m.gastoMesUsd)}</td>
                        <td>
                          <span className={`badge ${sit.classe}`}>{sit.texto}</span>
                        </td>
                        <td>
                          <span className={`badge ${m.ativo ? "t-success" : "t-neutral"}`}>
                            {m.ativo ? "Ativo" : "Inativo"}
                          </span>
                        </td>
                        <td>
                          <div className="row" style={{ flexWrap: "nowrap" }}>
                            <button
                              type="button"
                              className="btn peq primario"
                              disabled={testar.isPending}
                              onClick={() => executarTeste(m.id)}
                            >
                              {testandoEste ? (
                                <Loader2 size={14} className="girar" aria-hidden />
                              ) : (
                                <Play size={14} aria-hidden />
                              )}
                              Testar
                            </button>
                            <button type="button" className="btn peq" onClick={() => setModeloPrecos(m)}>
                              <DollarSign size={14} aria-hidden /> Preços
                            </button>
                            <button type="button" className="btn peq" onClick={() => setModeloEditar(m)}>
                              Editar
                            </button>
                            <button
                              type="button"
                              className="btn peq"
                              disabled={editarModelo.isPending}
                              onClick={() => alternarAtivo(m)}
                            >
                              {m.ativo ? "Desativar" : "Reativar"}
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

        <FormularioModeloManual fornecedorId={fornecedorId} />
      </div>

      {modeloPrecos && (
        <GavetaPrecosModelo
          modelo={modeloPrecos}
          fornecedorId={fornecedorId}
          aoFechar={() => setModeloPrecos(null)}
        />
      )}

      {modeloEditar && (
        <GavetaEditarModelo
          modelo={modeloEditar}
          fornecedorId={fornecedorId}
          aoFechar={() => setModeloEditar(null)}
        />
      )}

      {confirmandoExclusao && fornecedor && (
        <DialogoExcluirFornecedor
          fornecedorNome={fornecedor.nome}
          enviando={excluirFornecedor.isPending}
          erro={
            excluirFornecedor.error instanceof FalhaApi
              ? excluirFornecedor.error.message
              : excluirFornecedor.error
                ? "Não foi possível excluir o fornecedor."
                : ""
          }
          aoCancelar={() => {
            if (!excluirFornecedor.isPending) {
              excluirFornecedor.reset();
              setConfirmandoExclusao(false);
            }
          }}
          aoConfirmar={() => {
            excluirFornecedor.mutate(fornecedor.id, {
              onSuccess: () => {
                setConfirmandoExclusao(false);
                router.push("/admin/fornecedores");
              },
            });
          }}
        />
      )}
    </>
  );
}

function DialogoExcluirFornecedor({
  fornecedorNome,
  enviando,
  erro,
  aoCancelar,
  aoConfirmar,
}: {
  fornecedorNome: string;
  enviando: boolean;
  erro: string;
  aoCancelar: () => void;
  aoConfirmar: () => void;
}) {
  useEffect(() => {
    const aoTeclar = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !enviando) aoCancelar();
    };
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [aoCancelar, enviando]);

  return (
    <div
      className="drawer-fundo"
      style={{ justifyContent: "center", alignItems: "center", padding: 16 }}
      onClick={aoCancelar}
    >
      <div
        className="card stack"
        style={{ width: "min(460px, 100%)", boxShadow: "var(--shadow-overlay)" }}
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-excluir-fornecedor"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="row" style={{ justifyContent: "space-between" }}>
          <h2 id="titulo-excluir-fornecedor" style={{ margin: 0 }}>
            Excluir fornecedor
          </h2>
          <button
            type="button"
            className="btn peq"
            aria-label="Fechar"
            disabled={enviando}
            onClick={aoCancelar}
          >
            <X size={16} aria-hidden />
          </button>
        </div>

        <p style={{ margin: 0 }}>
          Tem certeza de que deseja excluir o fornecedor <b>{fornecedorNome}</b>?
        </p>
        <p className="small muted" style={{ margin: 0 }}>
          Os modelos, preços e a chave cadastrada deste fornecedor serão removidos. O histórico de execuções e
          custos será mantido.
        </p>

        {erro && (
          <div className="alerta t-danger" role="alert">
            {erro}
          </div>
        )}

        <div className="row" style={{ justifyContent: "flex-end", marginTop: 8 }}>
          <button type="button" className="btn" disabled={enviando} onClick={aoCancelar}>
            Cancelar
          </button>
          <button type="button" className="btn perigo" disabled={enviando} onClick={aoConfirmar}>
            {enviando ? "Excluindo…" : "Confirmar exclusão"}
          </button>
        </div>
      </div>
    </div>
  );
}

type CamposNovoModelo = {
  identificador: string;
  servico: Servico;
};

function FormularioModeloManual({ fornecedorId }: { fornecedorId: string }) {
  const criar = useCriarModelo(fornecedorId);
  useRedirecionarSemPermissao(criar.error);

  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors },
  } = useForm<CamposNovoModelo>({
    defaultValues: { identificador: "", servico: "texto" },
  });

  const enviar = handleSubmit((campos) => {
    criar.reset();
    const validado = PedidoNovoModelo.safeParse({
      identificador: campos.identificador,
      servico: campos.servico,
      origem: "manual",
    });
    if (!validado.success) {
      for (const issue of validado.error.issues) {
        const campo = issue.path[0] as keyof CamposNovoModelo | undefined;
        if (campo) setError(campo, { message: issue.message });
      }
      return;
    }

    criar.mutate(validado.data, {
      onSuccess: () => reset({ identificador: "", servico: campos.servico }),
      onError: (e) => {
        if (e instanceof FalhaApi && e.codigo === "modelo_repetido") {
          setError("identificador", { message: e.message });
        }
      },
    });
  });

  const erroGeral =
    criar.error instanceof FalhaApi ? criar.error.message : criar.error ? "Não foi possível incluir o modelo." : "";

  return (
    <section className="card stack">
      <h2 style={{ margin: 0 }}>Incluir modelo à mão</h2>
      <form className="row" style={{ alignItems: "flex-end" }} onSubmit={enviar} noValidate>
        <label className="campo" style={{ flex: "1 1 260px" }}>
          Identificador do modelo
          <input
            type="text"
            placeholder="Ex.: gpt-4o-mini ou simulado-texto"
            aria-invalid={!!errors.identificador}
            {...register("identificador")}
          />
          {errors.identificador && (
            <span className="small" style={{ color: "var(--laura-danger)" }}>
              {errors.identificador.message}
            </span>
          )}
        </label>

        <label className="campo" style={{ minWidth: 160 }}>
          Serviço
          <select {...register("servico")}>
            <option value="texto">Texto</option>
            <option value="imagem">Imagem</option>
          </select>
        </label>

        <button type="submit" className="btn primario" disabled={criar.isPending || !fornecedorId}>
          <Plus size={16} aria-hidden /> {criar.isPending ? "Incluindo…" : "Incluir modelo"}
        </button>
      </form>
      {erroGeral && (
        <p role="alert" style={{ color: "var(--laura-danger)", margin: 0 }}>
          {erroGeral}
        </p>
      )}
    </section>
  );
}

type CamposNovoPreco = {
  unidade: UnidadePreco;
  precoUsd: string;
  vigenciaInicio: string;
};

function GavetaPrecosModelo({
  modelo,
  fornecedorId,
  aoFechar,
}: {
  modelo: Modelo;
  fornecedorId: string;
  aoFechar: () => void;
}) {
  const precos = usePrecosModelo(modelo.id);
  const criarPreco = useCriarPreco(modelo.id, fornecedorId);
  const bloqueado = useRedirecionarSemPermissao(precos.error || criarPreco.error);

  const unidadesPermitidas: UnidadePreco[] =
    modelo.servico === "imagem" ? ["imagem"] : ["token_entrada", "token_saida"];

  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors },
  } = useForm<CamposNovoPreco>({
    defaultValues: {
      unidade: unidadesPermitidas[0],
      precoUsd: "",
      vigenciaInicio: hojeIso(),
    },
  });

  useEffect(() => {
    const aoTeclar = (e: KeyboardEvent) => {
      if (e.key === "Escape") aoFechar();
    };
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [aoFechar]);

  if (bloqueado) return null;

  const enviar = handleSubmit((campos) => {
    criarPreco.reset();
    const valor = Number(campos.precoUsd);
    if (campos.precoUsd.trim() === "" || !Number.isFinite(valor) || valor < 0) {
      setError("precoUsd", { message: "Informe um valor maior ou igual a zero." });
      return;
    }

    const validado = PedidoNovoPreco.safeParse({
      unidade: campos.unidade,
      precoUsd: valor,
      vigenciaInicio: campos.vigenciaInicio,
    });
    if (!validado.success) {
      for (const issue of validado.error.issues) {
        const campo = issue.path[0] as keyof CamposNovoPreco | undefined;
        if (campo) setError(campo, { message: issue.message });
      }
      return;
    }

    criarPreco.mutate(validado.data, {
      onSuccess: () => {
        reset({
          unidade: campos.unidade,
          precoUsd: "",
          vigenciaInicio: campos.vigenciaInicio,
        });
      },
    });
  });

  const erroGeral =
    criarPreco.error instanceof FalhaApi
      ? criarPreco.error.message
      : criarPreco.error
        ? "Não foi possível salvar o preço."
        : "";

  return (
    <div className="drawer-fundo" onClick={aoFechar}>
      <div
        className="drawer stack"
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-precos-modelo"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="row" style={{ justifyContent: "space-between" }}>
          <div>
            <h2 id="titulo-precos-modelo" style={{ margin: 0 }}>Preços do modelo</h2>
            <p className="small muted" style={{ margin: 0 }}>
              {modelo.identificador} · {modelo.servico === "imagem" ? "Imagem" : "Texto"}
            </p>
          </div>
          <button type="button" className="btn peq" aria-label="Fechar" onClick={aoFechar}>
            <X size={16} aria-hidden />
          </button>
        </div>

        <form className="card stack" style={{ padding: 16 }} onSubmit={enviar} noValidate>
          <h3 style={{ margin: 0 }}>Novo preço</h3>
          <label className="campo">
            Unidade
            <select {...register("unidade")}>
              {unidadesPermitidas.map((u) => (
                <option key={u} value={u}>
                  {nomeUnidadePreco[u]}
                </option>
              ))}
            </select>
          </label>

          <div className="grid g2" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <label className="campo">
              Valor (US$)
              <input
                type="number"
                step="0.000001"
                min="0"
                aria-invalid={!!errors.precoUsd}
                {...register("precoUsd")}
              />
              {errors.precoUsd && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.precoUsd.message}
                </span>
              )}
            </label>

            <label className="campo">
              Vigência (início)
              <input
                type="date"
                aria-invalid={!!errors.vigenciaInicio}
                {...register("vigenciaInicio")}
              />
              {errors.vigenciaInicio && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.vigenciaInicio.message}
                </span>
              )}
            </label>
          </div>

          {erroGeral && (
            <p role="alert" style={{ color: "var(--laura-danger)", margin: 0 }}>
              {erroGeral}
            </p>
          )}

          <button type="submit" className="btn primario" disabled={criarPreco.isPending}>
            {criarPreco.isPending ? "Salvando…" : "Adicionar preço"}
          </button>
        </form>

        <div className="stack">
          <h3 style={{ margin: 0 }}>Histórico</h3>
          {precos.isPending && <p className="muted">Carregando preços…</p>}
          {precos.isError && (
            <div className="alerta t-danger" role="alert">
              {precos.error.message}
            </div>
          )}
          {precos.data && precos.data.precos.length === 0 && (
            <p className="muted" style={{ margin: 0 }}>
              Nenhum preço cadastrado ainda.
            </p>
          )}
          {precos.data && precos.data.precos.length > 0 && (
            <div className="rolagem">
              <table>
                <thead>
                  <tr>
                    <th>Unidade</th>
                    <th className="d">Valor</th>
                    <th>Vigência</th>
                    <th>Criado em</th>
                  </tr>
                </thead>
                <tbody>
                  {precos.data.precos.map((p) => (
                    <tr key={p.id}>
                      <td className="small">{nomeUnidadePreco[p.unidade]}</td>
                      <td className="d">{formatarPrecoUsd(p.precoUsd)}</td>
                      <td className="num small">{formatarDataIso(p.vigenciaInicio)}</td>
                      <td className="num small muted">{formatarDataHora(p.criadoEm)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

type CamposEditarModelo = {
  servico: Servico;
  limiteMensalUsd: string;
  avisoPercentual: string;
  ativo: boolean;
};

function GavetaEditarModelo({
  modelo,
  fornecedorId,
  aoFechar,
}: {
  modelo: Modelo;
  fornecedorId: string;
  aoFechar: () => void;
}) {
  const editar = useEditarModelo(fornecedorId);
  const bloqueado = useRedirecionarSemPermissao(editar.error);

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<CamposEditarModelo>({
    defaultValues: {
      servico: modelo.servico,
      limiteMensalUsd: modelo.limiteMensalUsd !== null ? String(modelo.limiteMensalUsd) : "",
      avisoPercentual: modelo.avisoPercentual !== null ? String(modelo.avisoPercentual) : "",
      ativo: modelo.ativo,
    },
  });

  useEffect(() => {
    const aoTeclar = (e: KeyboardEvent) => {
      if (e.key === "Escape") aoFechar();
    };
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [aoFechar]);

  if (bloqueado) return null;

  const enviar = handleSubmit((campos) => {
    editar.reset();
    let limite: number | null = null;
    if (campos.limiteMensalUsd.trim() !== "") {
      const n = Number(campos.limiteMensalUsd);
      if (!Number.isFinite(n) || n <= 0) {
        setError("limiteMensalUsd", { message: "Informe um limite maior que zero ou deixe vazio." });
        return;
      }
      limite = n;
    }

    let aviso: number | null = null;
    if (campos.avisoPercentual.trim() !== "") {
      const n = Number(campos.avisoPercentual);
      if (!Number.isInteger(n) || n < 1 || n > 100) {
        setError("avisoPercentual", { message: "Informe um inteiro entre 1 e 100 ou deixe vazio." });
        return;
      }
      aviso = n;
    }

    const validado = PedidoEditarModelo.safeParse({
      servico: campos.servico,
      limiteMensalUsd: limite,
      avisoPercentual: aviso,
      ativo: campos.ativo,
    });
    if (!validado.success) {
      for (const issue of validado.error.issues) {
        const campo = issue.path[0] as keyof CamposEditarModelo | undefined;
        if (campo) setError(campo, { message: issue.message });
      }
      return;
    }

    editar.mutate(
      { id: modelo.id, dados: validado.data },
      { onSuccess: () => aoFechar() },
    );
  });

  const erroGeral =
    editar.error instanceof FalhaApi ? editar.error.message : editar.error ? "Não foi possível salvar." : "";

  return (
    <div className="drawer-fundo" onClick={aoFechar}>
      <div
        className="drawer stack"
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-editar-modelo"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="row" style={{ justifyContent: "space-between" }}>
          <div>
            <h2 id="titulo-editar-modelo" style={{ margin: 0 }}>Editar modelo</h2>
            <p className="small muted num" style={{ margin: 0 }}>{modelo.identificador}</p>
          </div>
          <button type="button" className="btn peq" aria-label="Fechar" onClick={aoFechar}>
            <X size={16} aria-hidden />
          </button>
        </div>

        <form className="stack" onSubmit={enviar} noValidate>
          <label className="campo">
            Serviço
            <select {...register("servico")}>
              <option value="texto">Texto</option>
              <option value="imagem">Imagem</option>
            </select>
          </label>

          <label className="campo">
            Limite mensal próprio (US$)
            <input
              type="number"
              step="0.01"
              min="0.01"
              placeholder="Vazio = usa o do fornecedor"
              aria-invalid={!!errors.limiteMensalUsd}
              {...register("limiteMensalUsd")}
            />
            <span className="small muted">Deixe vazio para usar o limite do fornecedor.</span>
            {errors.limiteMensalUsd && (
              <span className="small" style={{ color: "var(--laura-danger)" }}>
                {errors.limiteMensalUsd.message}
              </span>
            )}
          </label>

          <label className="campo">
            Aviso próprio (%)
            <input
              type="number"
              step="1"
              min="1"
              max="100"
              placeholder="Vazio = usa o do fornecedor"
              aria-invalid={!!errors.avisoPercentual}
              {...register("avisoPercentual")}
            />
            {errors.avisoPercentual && (
              <span className="small" style={{ color: "var(--laura-danger)" }}>
                {errors.avisoPercentual.message}
              </span>
            )}
          </label>

          <label className="row" style={{ fontWeight: 500, fontSize: 13 }}>
            <input type="checkbox" {...register("ativo")} />
            Modelo ativo
          </label>

          {erroGeral && (
            <p role="alert" style={{ color: "var(--laura-danger)", margin: 0 }}>
              {erroGeral}
            </p>
          )}

          <div className="row" style={{ justifyContent: "flex-end", marginTop: 8 }}>
            <button type="button" className="btn" onClick={aoFechar}>
              Cancelar
            </button>
            <button type="submit" className="btn primario" disabled={editar.isPending}>
              {editar.isPending ? "Salvando…" : "Salvar alterações"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
