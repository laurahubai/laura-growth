"use client";
// Etapa 2: kit de marca da organização ativa (MKT-01, MKT-03, MKT-04).
import { useEffect, useRef, useState } from "react";
import { Download, FileText, Plus, Trash2, Upload, X } from "lucide-react";
import { LIMITES_KIT, type ArquivoMarca, type KitMarca } from "@laura-growth/contratos";
import { urlArquivo, useEnviarArquivo, useKit, useRemoverArquivo, useSalvarKit, useSessao } from "@/lib/api";

type Rascunho = Omit<KitMarca, "atualizadoEm">;
type CampoTexto = "site" | "perfis" | "tomDeVoz" | "publico" | "temas" | "nuncaDizer" | "fontes" | "exemplos";

const CAMPOS: { k: CampoTexto; rotulo: string; dica?: string; longo?: boolean }[] = [
  { k: "site", rotulo: "Site da marca", dica: "Endereço completo, começando com https://" },
  { k: "perfis", rotulo: "Perfis nas redes", dica: "Um por linha" , longo: true },
  { k: "tomDeVoz", rotulo: "Tom de voz", longo: true },
  { k: "publico", rotulo: "Público", longo: true },
  { k: "temas", rotulo: "Temas", dica: "Separados por vírgula" },
  { k: "nuncaDizer", rotulo: "O que nunca dizer", dica: "Separado por vírgula" },
  { k: "fontes", rotulo: "Fontes" },
  { k: "exemplos", rotulo: "Exemplos de posts", longo: true },
];
const NOMES_CORES = ["Principal", "Secundária", "Fundo"];

export default function Kit() {
  const { data: sessao } = useSessao();
  const orgId = sessao?.organizacaoAtiva.id ?? "";
  const kit = useKit(orgId);
  const salvar = useSalvarKit(orgId);
  const [r, setR] = useState<Rascunho | null>(null);
  const [alterado, setAlterado] = useState(false);
  // Índice do item recém-adicionado, para levar o cursor até ele.
  const [focar, setFocar] = useState<{ lista: "pilares" | "cores"; i: number } | null>(null);

  useEffect(() => {
    if (kit.data && !alterado) {
      const { atualizadoEm: _, ...resto } = kit.data.kit;
      setR(resto);
    }
  }, [kit.data, alterado]);

  if (!sessao) return null;
  if (kit.isPending || !r) return <p className="muted">Carregando…</p>;
  if (kit.isError) return <p role="alert" style={{ color: "var(--laura-danger)" }}>{kit.error.message}</p>;

  const mudar = (novo: Rascunho) => { setR(novo); setAlterado(true); salvar.reset(); };
  const pilaresPreenchidos = r.pilares.filter((p) => p.trim()).length;
  const atualizado = kit.data.kit.atualizadoEm;

  return (
    <>
      <div className="page-header">
        <div>
          <h1>Kit de marca</h1>
          <p>Base que os agentes usam para criar os posts da <b>{sessao.organizacaoAtiva.nome}</b>. Nunca é usado em outra organização.</p>
        </div>
      </div>
      <form
        className="grid g2"
        onSubmit={(e) => {
          e.preventDefault();
          salvar.mutate(r, { onSuccess: () => setAlterado(false) });
        }}
      >
        <section className="card stack">
          {CAMPOS.map((c) => (
            <label key={c.k} className="campo">
              {c.rotulo}
              {c.longo ? (
                <textarea value={r[c.k]} maxLength={LIMITES_KIT.textoMax} onChange={(e) => mudar({ ...r, [c.k]: e.target.value })} />
              ) : (
                <input value={r[c.k]} maxLength={LIMITES_KIT.textoMax} type={c.k === "site" ? "url" : "text"} onChange={(e) => mudar({ ...r, [c.k]: e.target.value })} />
              )}
              {c.dica && <span className="small muted">{c.dica}</span>}
            </label>
          ))}
          <div className="row" style={{ borderTop: "1px solid var(--laura-border)", paddingTop: 12 }}>
            <button className="btn primario" type="submit" disabled={salvar.isPending || !alterado}>
              {salvar.isPending ? "Salvando…" : "Salvar kit de marca"}
            </button>
            {salvar.isSuccess && !alterado && <span role="status" style={{ color: "var(--laura-success)" }}>Kit salvo.</span>}
            {alterado && !salvar.isPending && <span className="small muted">Alterações não salvas.</span>}
            {salvar.isError && <span role="alert" style={{ color: "var(--laura-danger)" }}>{salvar.error.message}</span>}
          </div>
          {atualizado && <p className="small muted" style={{ margin: 0 }}>Última gravação: {new Date(atualizado).toLocaleString("pt-BR")}</p>}
        </section>

        <div className="stack">
          <section className="card stack">
            <h2 style={{ margin: 0 }}>Pilares de conteúdo</h2>
            <p className="small muted" style={{ margin: 0 }}>Os {LIMITES_KIT.pilaresMin} a {LIMITES_KIT.pilaresMax} assuntos fixos sobre os quais a marca sempre fala. O agente distribui os posts entre eles.</p>
            {r.pilares.map((p, i) => (
              <div key={i} className="row" style={{ flexWrap: "nowrap" }}>
                <input
                  style={{ flex: 1, minWidth: 0 }}
                  autoFocus={focar?.lista === "pilares" && focar.i === i}
                  value={p}
                  maxLength={LIMITES_KIT.pilarMax}
                  aria-label={`Pilar ${i + 1}`}
                  onChange={(e) => mudar({ ...r, pilares: r.pilares.map((x, j) => (j === i ? e.target.value : x)) })}
                />
                <button type="button" className="btn peq" aria-label={`Remover pilar ${i + 1}`} onClick={() => mudar({ ...r, pilares: r.pilares.filter((_, j) => j !== i) })}>
                  <X size={14} aria-hidden />
                </button>
              </div>
            ))}
            {r.pilares.length < LIMITES_KIT.pilaresMax && (
              <button type="button" className="btn peq" style={{ alignSelf: "flex-start" }} onClick={() => { setFocar({ lista: "pilares", i: r.pilares.length }); mudar({ ...r, pilares: [...r.pilares, ""] }); }}>
                <Plus size={14} aria-hidden /> Adicionar pilar
              </button>
            )}
            {pilaresPreenchidos < LIMITES_KIT.pilaresMin && <div className="alerta t-warning">Recomendado: de {LIMITES_KIT.pilaresMin} a {LIMITES_KIT.pilaresMax} pilares.</div>}
          </section>

          <section className="card stack">
            <h2 style={{ margin: 0 }}>Cores</h2>
            {r.cores.map((c, i) => (
              <div key={i} className="row">
                <input type="color" value={c} aria-label={`Cor ${i + 1}`} onChange={(e) => mudar({ ...r, cores: r.cores.map((x, j) => (j === i ? e.target.value : x)) })} />
                <span className="num">{c}</span>
                <span className="small muted">{NOMES_CORES[i] ?? "Apoio"}</span>
                <button type="button" className="btn peq" style={{ marginLeft: "auto" }} aria-label={`Remover cor ${i + 1}`} onClick={() => mudar({ ...r, cores: r.cores.filter((_, j) => j !== i) })}>
                  <X size={14} aria-hidden />
                </button>
              </div>
            ))}
            {r.cores.length < LIMITES_KIT.coresMax && (
              <button type="button" className="btn peq" style={{ alignSelf: "flex-start" }} onClick={() => mudar({ ...r, cores: [...r.cores, "#000000"] })}>
                <Plus size={14} aria-hidden /> Adicionar cor
              </button>
            )}
          </section>

          <Arquivos orgId={orgId} arquivos={kit.data.arquivos} cor={r.cores[0]} inicial={sessao.organizacaoAtiva.nome[0]} />
        </div>
      </form>
    </>
  );
}

function Arquivos({ orgId, arquivos, cor, inicial }: { orgId: string; arquivos: ArquivoMarca[]; cor?: string; inicial: string }) {
  const enviar = useEnviarArquivo(orgId);
  const remover = useRemoverArquivo(orgId);
  const campoLogo = useRef<HTMLInputElement>(null);
  const campoRef = useRef<HTMLInputElement>(null);
  const logo = arquivos.find((a) => a.tipo === "logo");
  const refs = arquivos.filter((a) => a.tipo === "referencia");

  const escolher = (tipo: "logo" | "referencia") => (e: React.ChangeEvent<HTMLInputElement>) => {
    const arquivo = e.target.files?.[0];
    e.target.value = "";
    if (!arquivo) return;
    if (arquivo.size > LIMITES_KIT.arquivoMaxMB * 1024 * 1024) {
      enviar.reset();
      alert(`O arquivo pode ter no máximo ${LIMITES_KIT.arquivoMaxMB} MB.`);
      return;
    }
    enviar.mutate({ tipo, arquivo });
  };
  const apagar = (a: ArquivoMarca) => {
    if (confirm(`Remover "${a.nome}"? Não dá para desfazer.`)) remover.mutate(a.id);
  };

  return (
    <>
      <section className="card stack">
        <h2 style={{ margin: 0 }}>Logo</h2>
        <div className="row">
          {logo ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={urlArquivo(logo.id)} alt={`Logo: ${logo.nome}`} style={{ width: 64, height: 64, objectFit: "contain", borderRadius: 8, border: "1px solid var(--laura-border)", background: "#fff" }} />
          ) : (
            <span className="logo-mark" style={{ width: 64, height: 64, background: cor ?? "var(--laura-brand)", fontSize: 26 }} aria-hidden>{inicial}</span>
          )}
          <div className="stack" style={{ gap: 6 }}>
            <button type="button" className="btn peq" onClick={() => campoLogo.current?.click()} disabled={enviar.isPending}>
              <Upload size={14} aria-hidden /> {logo ? "Trocar logo" : "Enviar logo"}
            </button>
            {logo && (
              <button type="button" className="btn peq" onClick={() => apagar(logo)} disabled={remover.isPending}>
                <Trash2 size={14} aria-hidden /> Remover
              </button>
            )}
          </div>
        </div>
        <span className="small muted">PNG, JPG ou WebP, até {LIMITES_KIT.arquivoMaxMB} MB.</span>
        <input ref={campoLogo} type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={escolher("logo")} />
      </section>

      <section className="card stack">
        <h2 style={{ margin: 0 }}>Arquivos de referência</h2>
        <p className="small muted" style={{ margin: 0 }}>Manual da marca, exemplos de posts e outras referências. PNG, JPG, WebP ou PDF, até {LIMITES_KIT.arquivoMaxMB} MB cada.</p>
        {refs.length === 0 && <p className="small muted" style={{ margin: 0 }}>Nenhum arquivo enviado.</p>}
        {refs.map((a) => (
          <div key={a.id} className="row" style={{ flexWrap: "nowrap" }}>
            <FileText size={16} aria-hidden style={{ flex: "none" }} />
            <a href={urlArquivo(a.id)} target="_blank" rel="noopener" style={{ flex: 1, minWidth: 0, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{a.nome}</a>
            <span className="small muted num" style={{ flex: "none" }}>{(a.tamanho / 1024 / 1024).toFixed(1)} MB</span>
            <a className="btn peq" href={urlArquivo(a.id, true)} aria-label={`Baixar ${a.nome}`}><Download size={14} aria-hidden /></a>
            <button type="button" className="btn peq" aria-label={`Remover ${a.nome}`} onClick={() => apagar(a)} disabled={remover.isPending}><Trash2 size={14} aria-hidden /></button>
          </div>
        ))}
        <button type="button" className="btn peq" style={{ alignSelf: "flex-start" }} onClick={() => campoRef.current?.click()} disabled={enviar.isPending}>
          <Upload size={14} aria-hidden /> {enviar.isPending ? "Enviando…" : "Enviar arquivo"}
        </button>
        <input ref={campoRef} type="file" accept="image/png,image/jpeg,image/webp,application/pdf" hidden onChange={escolher("referencia")} />
        {enviar.isError && <span role="alert" style={{ color: "var(--laura-danger)" }}>{enviar.error.message}</span>}
        {remover.isError && <span role="alert" style={{ color: "var(--laura-danger)" }}>{remover.error.message}</span>}
      </section>
    </>
  );
}
