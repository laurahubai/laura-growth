"use client";
// Etapas 2 e 2C: kit de marca / DNA da marca da organização ativa (MKT-01, MKT-03, MKT-04, RF-02, RF-10).
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Download, FileText, History, Plus, RefreshCw, Trash2, Upload, X } from "lucide-react";
import {
  CanalKit,
  LIMITES_KIT,
  ModeloNegocio,
  nomeCanalKit,
  nomeModeloNegocio,
  type ArquivoMarca,
  type CampoKit,
  type KitMarca,
  type OrigemCampo,
} from "@laura-growth/contratos";
import {
  BlocoOQueEvitar,
  CabecalhoCampoKit,
  CAMPOS_COMUNICACAO,
} from "@/components/KitCompartilhado";
import { SemAcesso } from "@/components/SemAcesso";
import {
  erroSemPermissao,
  FalhaApi,
  urlArquivo,
  useConfirmarKit,
  useEnviarArquivo,
  useKit,
  useRemoverArquivo,
  useSalvarKit,
  useSessao,
} from "@/lib/api";

type Rascunho = Omit<KitMarca, "organizacaoId" | "atualizadoEm">;

const NOMES_CORES = ["Principal", "Secundária", "Fundo"];

function mesmoValorCampo(a: Rascunho, b: KitMarca, campo: CampoKit): boolean {
  if (campo === "modeloNegocio") {
    return a.modeloNegocio === b.modeloNegocio && a.modeloOutro === b.modeloOutro;
  }
  if (campo === "canais") {
    return (
      a.canaisOutro === b.canaisOutro &&
      JSON.stringify(a.canais) === JSON.stringify(b.canais)
    );
  }
  if (campo === "pilares" || campo === "cores" || campo === "produtos") {
    return JSON.stringify(a[campo]) === JSON.stringify(b[campo]);
  }
  return a[campo] === b[campo];
}

export default function Kit() {
  const { data: sessao } = useSessao();
  const orgId = sessao?.organizacaoAtiva.id ?? "";
  const kit = useKit(orgId);
  const salvar = useSalvarKit(orgId);
  const confirmar = useConfirmarKit(orgId);
  const [r, setR] = useState<Rascunho | null>(null);
  const [versaoBase, setVersaoBase] = useState(0);
  const [alterado, setAlterado] = useState(false);
  const [confirmandoCampo, setConfirmandoCampo] = useState<CampoKit | null>(null);
  const sincronizarVersaoRef = useRef(false);
  // Índice do item recém-adicionado, para levar o cursor até ele.
  const [focar, setFocar] = useState<{ lista: "pilares" | "cores" | "produtos"; i: number } | null>(null);
  // Cores com código digitado inválido (bloqueiam o salvar) e versão para recriar os campos ao remover uma cor.
  const [coresInvalidas, setCoresInvalidas] = useState<Set<number>>(new Set());
  const [versaoCores, setVersaoCores] = useState(0);

  useEffect(() => {
    if (!kit.data) return;
    if (!alterado) {
      const { organizacaoId: _org, atualizadoEm: _at, ...resto } = kit.data.kit;
      setR(resto);
      setVersaoBase(kit.data.versao);
      sincronizarVersaoRef.current = false;
    } else if (sincronizarVersaoRef.current) {
      setVersaoBase(kit.data.versao);
      sincronizarVersaoRef.current = false;
    }
  }, [kit.data, alterado]);

  if (!sessao) return null;
  if (kit.isPending) return <p className="muted">Carregando…</p>;
  if (erroSemPermissao(kit.error)) {
    return (
      <SemAcesso
        mensagem={kit.error instanceof Error ? kit.error.message : "Você não tem permissão para ver o kit de marca."}
      />
    );
  }
  if (kit.isError) return <p role="alert" style={{ color: "var(--laura-danger)" }}>{kit.error.message}</p>;
  if (!r || !kit.data) return <p className="muted">Carregando…</p>;

  const podeEditar = sessao.alcadas.includes("cmo_dna");
  const origens = kit.data.origens;
  const kitServidor = kit.data.kit;

  const mudar = (novo: Rascunho) => {
    if (!podeEditar) return;
    setR(novo);
    setAlterado(true);
    salvar.reset();
    confirmar.reset();
  };

  const confirmarCampo = (campo: CampoKit) => {
    if (!podeEditar || confirmar.isPending) return;
    setConfirmandoCampo(campo);
    salvar.reset();
    confirmar.mutate(
      { campos: [campo], versaoBase },
      {
        onSuccess: (resp) => {
          setVersaoBase(resp.versao);
        },
        onSettled: () => {
          setConfirmandoCampo(null);
        },
      },
    );
  };

  const recarregarAtual = async () => {
    const res = await kit.refetch();
    if (res.data) {
      const { organizacaoId: _org, atualizadoEm: _at, ...resto } = res.data.kit;
      setR(resto);
      setVersaoBase(res.data.versao);
      setAlterado(false);
      salvar.reset();
      confirmar.reset();
    }
  };

  const pilaresPreenchidos = r.pilares.filter((p) => p.trim()).length;
  const semProdutoPrioritario = r.produtos.length > 0 && !r.produtos.some((p) => p.prioritario);
  const atualizado = kit.data.kit.atualizadoEm;
  const erroAcao = salvar.isError ? salvar.error : confirmar.isError ? confirmar.error : null;
  const erroDesatualizado = erroAcao instanceof FalhaApi && erroAcao.codigo === "kit_desatualizado";

  const propsOrigem = (campo: CampoKit) => ({
    campo,
    origem: origens[campo],
    podeEditar,
    alteradoLocal: !mesmoValorCampo(r, kitServidor, campo),
    confirmando: confirmandoCampo === campo,
    onConfirmar: confirmarCampo,
  });

  return (
    <>
      <div className="page-header">
        <div>
          <h1>Kit de marca</h1>
          <p>
            Base que os agentes usam para criar os posts da <b>{sessao.organizacaoAtiva.nome}</b>. Nunca é usado em outra organização.
          </p>
        </div>
        <div className="row">
          {kit.data.versao > 0 && (
            <span className="badge t-neutral num" data-testid="versao-atual-kit">
              Versão {kit.data.versao}
            </span>
          )}
          <Link href="/kit/versoes" className="btn">
            <History size={16} aria-hidden /> Versões
          </Link>
        </div>
      </div>

      {!podeEditar && (
        <div className="alerta t-info" style={{ marginBottom: 16 }}>
          Modo somente leitura: você pode consultar o kit de marca, mas não tem alçada para editá-lo.
        </div>
      )}

      <BlocoOQueEvitar nuncaDizer={r.nuncaDizer} restricoesSetor={r.restricoesSetor} />

      <form
        className="stack"
        style={{ gap: 16 }}
        onSubmit={(e) => {
          e.preventDefault();
          if (!podeEditar) return;
          confirmar.reset();
          const envio = {
            ...r,
            organizacaoId: kitServidor.organizacaoId,
            modeloOutro: r.modeloNegocio === "outro" ? r.modeloOutro : "",
            versaoBase,
          };
          salvar.mutate(envio, {
            onSuccess: (resp) => {
              const { organizacaoId: _org, atualizadoEm: _at, ...resto } = resp.kit;
              setR(resto);
              setVersaoBase(resp.versao);
              setAlterado(false);
            },
          });
        }}
      >
        <div className="grid g2">
          <div className="stack" style={{ gap: 16 }}>
            <section className="card stack">
              <h2 style={{ margin: 0 }}>Marca e comunicação</h2>
              {CAMPOS_COMUNICACAO.map((c) => (
                <CampoTextoKit
                  key={c.k}
                  campo={c.k}
                  rotulo={c.rotulo}
                  dica={c.dica}
                  longo={c.longo}
                  tipo={c.k === "site" ? "url" : "text"}
                  valor={r[c.k]}
                  origem={origens[c.k]}
                  podeEditar={podeEditar}
                  alteradoLocal={!mesmoValorCampo(r, kitServidor, c.k)}
                  confirmando={confirmandoCampo === c.k}
                  onChange={(v) => mudar({ ...r, [c.k]: v })}
                  onConfirmar={confirmarCampo}
                />
              ))}
            </section>

            <section className="card stack" aria-labelledby="titulo-negocio">
              <h2 id="titulo-negocio" style={{ margin: 0 }}>Negócio</h2>

              <div className="stack" data-testid="campo-produtos">
                <CabecalhoCampoKit {...propsOrigem("produtos")} />
                {r.produtos.map((prod, i) => (
                  <div key={i} className="row" style={{ flexWrap: "nowrap", alignItems: "center" }}>
                    <input
                      style={{ flex: 1, minWidth: 0 }}
                      autoFocus={focar?.lista === "produtos" && focar.i === i}
                      value={prod.nome}
                      maxLength={LIMITES_KIT.produtoNomeMax}
                      disabled={!podeEditar}
                      placeholder="Nome do produto ou serviço"
                      aria-label={`Produto ou serviço ${i + 1}`}
                      onChange={(e) =>
                        mudar({
                          ...r,
                          produtos: r.produtos.map((item, j) =>
                            j === i ? { ...item, nome: e.target.value } : item,
                          ),
                        })
                      }
                    />
                    <label className="row small" style={{ gap: 6, whiteSpace: "nowrap", fontWeight: 500 }}>
                      <input
                        type="checkbox"
                        checked={prod.prioritario}
                        disabled={!podeEditar}
                        aria-label={`Prioritário ${i + 1}`}
                        onChange={(e) =>
                          mudar({
                            ...r,
                            produtos: r.produtos.map((item, j) =>
                              j === i ? { ...item, prioritario: e.target.checked } : item,
                            ),
                          })
                        }
                      />
                      Prioritário
                    </label>
                    {podeEditar && (
                      <button
                        type="button"
                        className="btn peq"
                        aria-label={`Remover produto ${i + 1}`}
                        onClick={() =>
                          mudar({
                            ...r,
                            produtos: r.produtos.filter((_, j) => j !== i),
                          })
                        }
                      >
                        <X size={14} aria-hidden />
                      </button>
                    )}
                  </div>
                ))}
                {podeEditar && r.produtos.length < LIMITES_KIT.produtosMax && (
                  <button
                    type="button"
                    className="btn peq"
                    style={{ alignSelf: "flex-start" }}
                    onClick={() => {
                      setFocar({ lista: "produtos", i: r.produtos.length });
                      mudar({ ...r, produtos: [...r.produtos, { nome: "", prioritario: false }] });
                    }}
                  >
                    <Plus size={14} aria-hidden /> Adicionar produto ou serviço
                  </button>
                )}
                {semProdutoPrioritario && (
                  <div className="alerta t-warning" data-testid="aviso-produto-prioritario">
                    Marque pelo menos um produto ou serviço como prioritário.
                  </div>
                )}
              </div>

              <CampoTextoKit
                campo="diferencial"
                rotulo="Diferencial"
                longo
                valor={r.diferencial}
                origem={origens.diferencial}
                podeEditar={podeEditar}
                alteradoLocal={!mesmoValorCampo(r, kitServidor, "diferencial")}
                confirmando={confirmandoCampo === "diferencial"}
                onChange={(v) => mudar({ ...r, diferencial: v })}
                onConfirmar={confirmarCampo}
              />

              <div className="campo" data-testid="campo-modeloNegocio">
                <CabecalhoCampoKit {...propsOrigem("modeloNegocio")} htmlFor="kit-modeloNegocio" />
                <select
                  id="kit-modeloNegocio"
                  value={r.modeloNegocio}
                  disabled={!podeEditar}
                  onChange={(e) => {
                    const v = e.target.value as ModeloNegocio;
                    mudar({
                      ...r,
                      modeloNegocio: v,
                      modeloOutro: v === "outro" ? r.modeloOutro : "",
                    });
                  }}
                >
                  <option value="">Selecione…</option>
                  {ModeloNegocio.options
                    .filter((m): m is Exclude<ModeloNegocio, ""> => m !== "")
                    .map((m) => (
                      <option key={m} value={m}>
                        {nomeModeloNegocio[m]}
                      </option>
                    ))}
                </select>
                {(r.modeloNegocio === "outro" || (!podeEditar && r.modeloOutro.trim() !== "")) && (
                  <label className="campo" style={{ marginTop: 8 }}>
                    Qual modelo de negócio?
                    <input
                      value={r.modeloOutro}
                      maxLength={LIMITES_KIT.textoMax}
                      disabled={!podeEditar}
                      placeholder="Descreva o modelo de negócio"
                      onChange={(e) => mudar({ ...r, modeloOutro: e.target.value })}
                    />
                  </label>
                )}
              </div>

              <CampoTextoKit
                campo="regioes"
                rotulo="Regiões e idiomas"
                dica="Ex.: Brasil (português), América Latina (espanhol)"
                valor={r.regioes}
                origem={origens.regioes}
                podeEditar={podeEditar}
                alteradoLocal={!mesmoValorCampo(r, kitServidor, "regioes")}
                confirmando={confirmandoCampo === "regioes"}
                onChange={(v) => mudar({ ...r, regioes: v })}
                onConfirmar={confirmarCampo}
              />

              <CampoTextoKit
                campo="comoCompram"
                rotulo="Como os clientes compram"
                longo
                valor={r.comoCompram}
                origem={origens.comoCompram}
                podeEditar={podeEditar}
                alteradoLocal={!mesmoValorCampo(r, kitServidor, "comoCompram")}
                confirmando={confirmandoCampo === "comoCompram"}
                onChange={(v) => mudar({ ...r, comoCompram: v })}
                onConfirmar={confirmarCampo}
              />
            </section>

            <section className="card stack" aria-labelledby="titulo-estrategia">
              <h2 id="titulo-estrategia" style={{ margin: 0 }}>Estratégia</h2>

              <CampoTextoKit
                campo="maiorDesafio"
                rotulo="Maior desafio"
                longo
                valor={r.maiorDesafio}
                origem={origens.maiorDesafio}
                podeEditar={podeEditar}
                alteradoLocal={!mesmoValorCampo(r, kitServidor, "maiorDesafio")}
                confirmando={confirmandoCampo === "maiorDesafio"}
                onChange={(v) => mudar({ ...r, maiorDesafio: v })}
                onConfirmar={confirmarCampo}
              />

              <CampoTextoKit
                campo="objetivo90Dias"
                rotulo="Objetivo dos próximos 90 dias"
                longo
                valor={r.objetivo90Dias}
                origem={origens.objetivo90Dias}
                podeEditar={podeEditar}
                alteradoLocal={!mesmoValorCampo(r, kitServidor, "objetivo90Dias")}
                confirmando={confirmandoCampo === "objetivo90Dias"}
                onChange={(v) => mudar({ ...r, objetivo90Dias: v })}
                onConfirmar={confirmarCampo}
              />

              <div className="campo" data-testid="campo-canais" role="group" aria-label="Canais">
                <CabecalhoCampoKit {...propsOrigem("canais")} />
                <div
                  className="grid"
                  style={{ gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))", gap: 8 }}
                >
                  {CanalKit.options.map((canal) => {
                    const marcado = r.canais.includes(canal);
                    return (
                      <label key={canal} className="row" style={{ gap: 6, fontWeight: 400 }}>
                        <input
                          type="checkbox"
                          checked={marcado}
                          disabled={!podeEditar}
                          onChange={(e) => {
                            const proximos = e.target.checked
                              ? [...r.canais, canal]
                              : r.canais.filter((c) => c !== canal);
                            mudar({ ...r, canais: proximos });
                          }}
                        />
                        {nomeCanalKit[canal]}
                      </label>
                    );
                  })}
                </div>
                <label className="campo" style={{ marginTop: 8 }}>
                  Outro canal
                  <input
                    value={r.canaisOutro}
                    maxLength={LIMITES_KIT.textoMax}
                    disabled={!podeEditar}
                    placeholder="Ex.: Podcast, eventos presenciais"
                    onChange={(e) => mudar({ ...r, canaisOutro: e.target.value })}
                  />
                </label>
              </div>

              <CampoTextoKit
                campo="restricoesSetor"
                rotulo="Restrições do setor"
                dica="Regras regulatórias, conselhos de classe ou limites do segmento."
                longo
                valor={r.restricoesSetor}
                origem={origens.restricoesSetor}
                podeEditar={podeEditar}
                alteradoLocal={!mesmoValorCampo(r, kitServidor, "restricoesSetor")}
                confirmando={confirmandoCampo === "restricoesSetor"}
                onChange={(v) => mudar({ ...r, restricoesSetor: v })}
                onConfirmar={confirmarCampo}
              />
            </section>
          </div>

          <div className="stack" style={{ gap: 16 }}>
            <section className="card stack" data-testid="campo-pilares">
              <CabecalhoCampoKit {...propsOrigem("pilares")} comoTitulo />
              <p className="small muted" style={{ margin: 0 }}>
                Os {LIMITES_KIT.pilaresMin} a {LIMITES_KIT.pilaresMax} assuntos fixos sobre os quais a marca sempre fala. O agente distribui os posts entre eles.
              </p>
              {r.pilares.map((p, i) => (
                <div key={i} className="row" style={{ flexWrap: "nowrap", alignItems: "flex-start" }}>
                  <div className="stack" style={{ flex: 1, minWidth: 0, gap: 4 }}>
                    <textarea
                      rows={3}
                      style={{ minHeight: 72 }}
                      autoFocus={focar?.lista === "pilares" && focar.i === i}
                      value={p}
                      disabled={!podeEditar}
                      maxLength={LIMITES_KIT.pilarMax}
                      aria-label={`Pilar ${i + 1}`}
                      onChange={(e) => mudar({ ...r, pilares: r.pilares.map((x, j) => (j === i ? e.target.value : x)) })}
                    />
                    <span className="small muted num" style={{ alignSelf: "flex-end" }}>
                      {p.length}/{LIMITES_KIT.pilarMax}
                    </span>
                  </div>
                  {podeEditar && (
                    <button
                      type="button"
                      className="btn peq"
                      aria-label={`Remover pilar ${i + 1}`}
                      onClick={() => mudar({ ...r, pilares: r.pilares.filter((_, j) => j !== i) })}
                    >
                      <X size={14} aria-hidden />
                    </button>
                  )}
                </div>
              ))}
              {podeEditar && r.pilares.length < LIMITES_KIT.pilaresMax && (
                <button
                  type="button"
                  className="btn peq"
                  style={{ alignSelf: "flex-start" }}
                  onClick={() => {
                    setFocar({ lista: "pilares", i: r.pilares.length });
                    mudar({ ...r, pilares: [...r.pilares, ""] });
                  }}
                >
                  <Plus size={14} aria-hidden /> Adicionar pilar
                </button>
              )}
              {pilaresPreenchidos < LIMITES_KIT.pilaresMin && (
                <div className="alerta t-warning">
                  Recomendado: de {LIMITES_KIT.pilaresMin} a {LIMITES_KIT.pilaresMax} pilares.
                </div>
              )}
            </section>

            <section className="card stack" data-testid="campo-cores">
              <CabecalhoCampoKit {...propsOrigem("cores")} comoTitulo />
              {r.cores.map((c, i) => (
                <div key={`${versaoCores}-${i}`} className="row">
                  <CampoCor
                    cor={c}
                    indice={i}
                    desabilitado={!podeEditar}
                    onCor={(nova) => mudar({ ...r, cores: r.cores.map((x, j) => (j === i ? nova : x)) })}
                    onValida={(ok) =>
                      setCoresInvalidas((s) => {
                        const n = new Set(s);
                        if (ok) n.delete(i);
                        else n.add(i);
                        return n;
                      })
                    }
                  />
                  <span className="small muted">{NOMES_CORES[i] ?? "Apoio"}</span>
                  {podeEditar && (
                    <button
                      type="button"
                      className="btn peq"
                      style={{ marginLeft: "auto" }}
                      aria-label={`Remover cor ${i + 1}`}
                      onClick={() => {
                        setCoresInvalidas(new Set());
                        setVersaoCores((v) => v + 1);
                        mudar({ ...r, cores: r.cores.filter((_, j) => j !== i) });
                      }}
                    >
                      <X size={14} aria-hidden />
                    </button>
                  )}
                </div>
              ))}
              {podeEditar && r.cores.length < LIMITES_KIT.coresMax && (
                <button
                  type="button"
                  className="btn peq"
                  style={{ alignSelf: "flex-start" }}
                  onClick={() => mudar({ ...r, cores: [...r.cores, "#000000"] })}
                >
                  <Plus size={14} aria-hidden /> Adicionar cor
                </button>
              )}
            </section>

            <Arquivos
              orgId={orgId}
              arquivos={kit.data.arquivos}
              cor={r.cores[0]}
              inicial={sessao.organizacaoAtiva.nome[0]}
              podeEditar={podeEditar}
              onArquivoAlterado={() => {
                sincronizarVersaoRef.current = true;
              }}
            />
          </div>
        </div>

        {(podeEditar || atualizado) && (
          <section className="card stack">
            {podeEditar && (
              <div className="row">
                <button
                  className="btn primario"
                  type="submit"
                  disabled={salvar.isPending || !alterado || coresInvalidas.size > 0}
                >
                  {salvar.isPending ? "Salvando…" : "Salvar kit de marca"}
                </button>
                {salvar.isSuccess && !alterado && (
                  <span role="status" style={{ color: "var(--laura-success)" }}>
                    Kit salvo.
                  </span>
                )}
                {alterado && !salvar.isPending && (
                  <span className="small muted">Alterações não salvas.</span>
                )}
                {coresInvalidas.size > 0 && (
                  <span role="alert" style={{ color: "var(--laura-danger)" }}>
                    Corrija o código da cor antes de salvar.
                  </span>
                )}
                {erroAcao && (
                  <span role="alert" style={{ color: "var(--laura-danger)" }}>
                    {erroAcao.message}
                  </span>
                )}
                {erroDesatualizado && (
                  <button
                    type="button"
                    className="btn peq"
                    onClick={recarregarAtual}
                    disabled={kit.isFetching}
                  >
                    <RefreshCw size={14} aria-hidden /> Recarregar versão atual
                  </button>
                )}
              </div>
            )}
            {atualizado && (
              <p className="small muted" style={{ margin: 0 }}>
                Última gravação: {new Date(atualizado).toLocaleString("pt-BR")}
              </p>
            )}
          </section>
        )}
      </form>
    </>
  );
}

function CampoTextoKit({
  campo,
  rotulo,
  dica,
  longo,
  tipo = "text",
  valor,
  origem,
  podeEditar,
  alteradoLocal,
  confirmando,
  onChange,
  onConfirmar,
}: {
  campo: CampoKit;
  rotulo: string;
  dica?: string;
  longo?: boolean;
  tipo?: "text" | "url";
  valor: string;
  origem?: OrigemCampo;
  podeEditar: boolean;
  alteradoLocal: boolean;
  confirmando: boolean;
  onChange: (valor: string) => void;
  onConfirmar: (campo: CampoKit) => void;
}) {
  const id = `kit-${campo}`;
  return (
    <div className="campo" data-testid={`campo-${campo}`}>
      <CabecalhoCampoKit
        campo={campo}
        htmlFor={id}
        titulo={rotulo}
        origem={origem}
        podeEditar={podeEditar}
        alteradoLocal={alteradoLocal}
        confirmando={confirmando}
        onConfirmar={onConfirmar}
      />
      {longo ? (
        <textarea
          id={id}
          value={valor}
          maxLength={LIMITES_KIT.textoMax}
          disabled={!podeEditar}
          onChange={(e) => onChange(e.target.value)}
        />
      ) : (
        <input
          id={id}
          value={valor}
          maxLength={LIMITES_KIT.textoMax}
          disabled={!podeEditar}
          type={tipo}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
      {dica && <span className="small muted">{dica}</span>}
    </div>
  );
}

// Seletor de cor + campo para digitar o código hexa (#RRGGBB, aceita sem # e minúsculas).
function CampoCor({
  cor,
  indice,
  desabilitado,
  onCor,
  onValida,
}: {
  cor: string;
  indice: number;
  desabilitado: boolean;
  onCor: (c: string) => void;
  onValida: (ok: boolean) => void;
}) {
  const [texto, setTexto] = useState(cor);
  const normalizar = (t: string) => {
    const s = t.trim().replace(/^#?/, "#");
    return /^#[0-9a-fA-F]{6}$/.test(s) ? s.toLowerCase() : null;
  };
  const valido = normalizar(texto) !== null;

  // Cor escolhida no seletor (ou vinda de fora) atualiza o texto.
  useEffect(() => {
    setTexto((t) => (normalizar(t) === cor.toLowerCase() ? t : cor));
  }, [cor]);
  useEffect(() => {
    onValida(valido);
  }, [valido]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <>
      <input
        type="color"
        value={cor}
        disabled={desabilitado}
        aria-label={`Cor ${indice + 1}`}
        onChange={(e) => onCor(e.target.value)}
      />
      <input
        className="num"
        style={{ width: "9ch", borderColor: valido ? undefined : "var(--laura-danger)" }}
        value={texto}
        disabled={desabilitado}
        maxLength={7}
        placeholder="#RRGGBB"
        spellCheck={false}
        aria-label={`Código hexa da cor ${indice + 1}`}
        aria-invalid={!valido}
        onChange={(e) => {
          setTexto(e.target.value);
          const n = normalizar(e.target.value);
          if (n && n !== cor.toLowerCase()) onCor(n);
        }}
        onBlur={() => {
          const n = normalizar(texto);
          if (n) setTexto(n);
        }}
      />
      {!valido && (
        <span className="small" style={{ color: "var(--laura-danger)" }}>
          Use o formato #RRGGBB
        </span>
      )}
    </>
  );
}

function Arquivos({
  orgId,
  arquivos,
  cor,
  inicial,
  podeEditar,
  onArquivoAlterado,
}: {
  orgId: string;
  arquivos: ArquivoMarca[];
  cor?: string;
  inicial: string;
  podeEditar: boolean;
  onArquivoAlterado: () => void;
}) {
  const enviar = useEnviarArquivo(orgId);
  const remover = useRemoverArquivo(orgId);
  const campoLogo = useRef<HTMLInputElement>(null);
  const campoRef = useRef<HTMLInputElement>(null);
  const logo = arquivos.find((a) => a.tipo === "logo");
  const refs = arquivos.filter((a) => a.tipo === "referencia");

  const escolher = (tipo: "logo" | "referencia") => (e: React.ChangeEvent<HTMLInputElement>) => {
    const arquivo = e.target.files?.[0];
    e.target.value = "";
    if (!arquivo || !podeEditar) return;
    if (arquivo.size > LIMITES_KIT.arquivoMaxMB * 1024 * 1024) {
      enviar.reset();
      alert(`O arquivo pode ter no máximo ${LIMITES_KIT.arquivoMaxMB} MB.`);
      return;
    }
    enviar.mutate({ tipo, arquivo }, { onSuccess: onArquivoAlterado });
  };
  const apagar = (a: ArquivoMarca) => {
    if (!podeEditar) return;
    if (confirm(`Remover "${a.nome}"? Não dá para desfazer.`)) {
      remover.mutate(a.id, { onSuccess: onArquivoAlterado });
    }
  };

  return (
    <>
      <section className="card stack">
        <h2 style={{ margin: 0 }}>Logo</h2>
        <div className="row">
          {logo ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={urlArquivo(logo.id)}
              alt={`Logo: ${logo.nome}`}
              style={{
                width: 64,
                height: 64,
                objectFit: "contain",
                borderRadius: 8,
                border: "1px solid var(--laura-border)",
                background: "#fff",
              }}
            />
          ) : (
            <span
              className="logo-mark"
              style={{ width: 64, height: 64, background: cor ?? "var(--laura-brand)", fontSize: 26 }}
              aria-hidden
            >
              {inicial}
            </span>
          )}
          {podeEditar && (
            <div className="stack" style={{ gap: 6 }}>
              <button
                type="button"
                className="btn peq"
                onClick={() => campoLogo.current?.click()}
                disabled={enviar.isPending}
              >
                <Upload size={14} aria-hidden /> {logo ? "Trocar logo" : "Enviar logo"}
              </button>
              {logo && (
                <button
                  type="button"
                  className="btn peq"
                  onClick={() => apagar(logo)}
                  disabled={remover.isPending}
                >
                  <Trash2 size={14} aria-hidden /> Remover
                </button>
              )}
            </div>
          )}
        </div>
        <span className="small muted">PNG, JPG ou WebP, até {LIMITES_KIT.arquivoMaxMB} MB.</span>
        {podeEditar && (
          <input
            ref={campoLogo}
            type="file"
            accept="image/png,image/jpeg,image/webp"
            hidden
            onChange={escolher("logo")}
          />
        )}
      </section>

      <section className="card stack">
        <h2 style={{ margin: 0 }}>Arquivos de referência</h2>
        <p className="small muted" style={{ margin: 0 }}>
          Manual da marca, exemplos de posts e outras referências. PNG, JPG, WebP ou PDF, até {LIMITES_KIT.arquivoMaxMB} MB cada.
        </p>
        {refs.length === 0 && <p className="small muted" style={{ margin: 0 }}>Nenhum arquivo enviado.</p>}
        {refs.map((a) => (
          <div key={a.id} className="row" style={{ flexWrap: "nowrap" }}>
            <FileText size={16} aria-hidden style={{ flex: "none" }} />
            <a
              href={urlArquivo(a.id)}
              target="_blank"
              rel="noopener"
              style={{ flex: 1, minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
            >
              {a.nome}
            </a>
            <span className="small muted num" style={{ flex: "none" }}>
              {(a.tamanho / 1024 / 1024).toFixed(1)} MB
            </span>
            <a className="btn peq" href={urlArquivo(a.id, true)} aria-label={`Baixar ${a.nome}`}>
              <Download size={14} aria-hidden />
            </a>
            {podeEditar && (
              <button
                type="button"
                className="btn peq"
                aria-label={`Remover ${a.nome}`}
                onClick={() => apagar(a)}
                disabled={remover.isPending}
              >
                <Trash2 size={14} aria-hidden />
              </button>
            )}
          </div>
        ))}
        {podeEditar && (
          <>
            <button
              type="button"
              className="btn peq"
              style={{ alignSelf: "flex-start" }}
              onClick={() => campoRef.current?.click()}
              disabled={enviar.isPending}
            >
              <Upload size={14} aria-hidden /> {enviar.isPending ? "Enviando…" : "Enviar arquivo"}
            </button>
            <input
              ref={campoRef}
              type="file"
              accept="image/png,image/jpeg,image/webp,application/pdf"
              hidden
              onChange={escolher("referencia")}
            />
          </>
        )}
        {enviar.isError && (
          <span role="alert" style={{ color: "var(--laura-danger)" }}>
            {enviar.error.message}
          </span>
        )}
        {remover.isError && (
          <span role="alert" style={{ color: "var(--laura-danger)" }}>
            {remover.error.message}
          </span>
        )}
      </section>
    </>
  );
}
