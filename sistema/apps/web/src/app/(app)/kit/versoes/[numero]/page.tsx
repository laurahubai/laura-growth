"use client";
// Etapa 2C: visualização somente leitura de uma versão histórica do kit (C4).
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Bot, User } from "lucide-react";
import {
  nomeCanalKit,
  nomeModeloNegocio,
  type CampoKit,
  type ModeloNegocio,
  type OrigemCampo,
} from "@laura-growth/contratos";
import {
  BadgeOrigem,
  BlocoOQueEvitar,
  CAMPOS_COMUNICACAO,
  rotuloCampoKit,
} from "@/components/KitCompartilhado";
import { SemAcesso } from "@/components/SemAcesso";
import { erroSemPermissao, FalhaApi, useSessao, useVersaoKit } from "@/lib/api";
import { formatarDataHora } from "@/lib/formato";

export default function VersaoKitDetalhe() {
  const params = useParams<{ numero: string }>();
  const { data: sessao } = useSessao();
  const orgId = sessao?.organizacaoAtiva.id ?? "";
  const numeroBruto = Number(params.numero);
  const numeroValido = Number.isInteger(numeroBruto) && numeroBruto > 0 ? numeroBruto : null;
  const versao = useVersaoKit(orgId, numeroValido);

  if (!sessao) return null;

  if (numeroValido === null) {
    return (
      <section className="card stack" role="alert" style={{ maxWidth: 520 }}>
        <h1 style={{ margin: 0 }}>Versão inválida</h1>
        <p style={{ margin: 0 }}>O número da versão informado não é válido.</p>
        <div>
          <Link href="/kit/versoes" className="btn">
            Voltar para versões
          </Link>
        </div>
      </section>
    );
  }

  if (versao.isPending) return <p className="muted">Carregando…</p>;

  if (erroSemPermissao(versao.error)) {
    return (
      <SemAcesso
        mensagem={
          versao.error instanceof Error
            ? versao.error.message
            : "Você não tem permissão para ver esta versão do kit de marca."
        }
      />
    );
  }

  if (versao.error instanceof FalhaApi && versao.error.status === 404) {
    return (
      <section className="card stack" role="alert" data-testid="versao-nao-encontrada" style={{ maxWidth: 520 }}>
        <h1 style={{ margin: 0 }}>Versão não encontrada</h1>
        <p style={{ margin: 0 }}>{versao.error.message}</p>
        <div>
          <Link href="/kit/versoes" className="btn">
            Voltar para versões
          </Link>
        </div>
      </section>
    );
  }

  if (versao.isError) {
    return (
      <p role="alert" style={{ color: "var(--laura-danger)" }}>
        {versao.error.message}
      </p>
    );
  }

  const v = versao.data;
  const d = v.dados;
  const origens = v.origens;
  const textoModelo =
    d.modeloNegocio === ""
      ? ""
      : d.modeloNegocio === "outro" && d.modeloOutro.trim()
        ? `${nomeModeloNegocio.outro}: ${d.modeloOutro}`
        : nomeModeloNegocio[d.modeloNegocio as Exclude<ModeloNegocio, "">];

  const listaCanais = [
    ...d.canais.map((c) => nomeCanalKit[c]),
    ...(d.canaisOutro.trim() ? [`Outro: ${d.canaisOutro.trim()}`] : []),
  ];

  return (
    <div className="stack" style={{ gap: 16 }}>
      <div className="page-header" style={{ marginBottom: 0 }}>
        <div>
          <h1>Versão {v.numero} do kit de marca</h1>
          <div className="row small muted" style={{ marginTop: 4, gap: 10 }}>
            {v.autorTipo === "ia" ? (
              <span className="badge t-ai">
                <Bot size={12} aria-hidden /> IA{v.autorNome ? ` · ${v.autorNome}` : ""}
              </span>
            ) : (
              <span className="row" style={{ gap: 4 }}>
                <User size={14} aria-hidden /> {v.autorNome ?? "Pessoa"}
              </span>
            )}
            <span className="num">{formatarDataHora(v.criadoEm)}</span>
          </div>
        </div>
        <div className="row">
          <Link href="/kit/versoes" className="btn">
            <ArrowLeft size={16} aria-hidden /> Voltar para versões
          </Link>
          <Link href="/kit" className="btn">
            Kit atual
          </Link>
        </div>
      </div>

      <div className="alerta t-info" data-testid="aviso-versao-somente-leitura">
        Versão histórica em modo somente leitura.
      </div>

      {v.camposAlterados.length > 0 && (
        <section className="card stack" style={{ gap: 8 }}>
          <h2 style={{ margin: 0 }}>Campos alterados nesta versão</h2>
          <div className="row" style={{ gap: 6 }}>
            {v.camposAlterados.map((c) => (
              <span key={c} className="badge t-neutral">
                {rotuloCampoKit(c)}
              </span>
            ))}
          </div>
        </section>
      )}

      <BlocoOQueEvitar nuncaDizer={d.nuncaDizer} restricoesSetor={d.restricoesSetor} />

      <div className="grid g2">
        <div className="stack" style={{ gap: 16 }}>
          <section className="card stack">
            <h2 style={{ margin: 0 }}>Marca e comunicação</h2>
            {CAMPOS_COMUNICACAO.map((c) => (
              <CampoLeitura
                key={c.k}
                campo={c.k}
                rotulo={c.rotulo}
                valor={d[c.k]}
                origem={origens[c.k]}
              />
            ))}
          </section>

          <section className="card stack">
            <h2 style={{ margin: 0 }}>Negócio</h2>

            <div className="stack" style={{ gap: 6 }} data-testid="versao-campo-produtos">
              <div className="row" style={{ justifyContent: "space-between" }}>
                <span style={{ fontWeight: 500, fontSize: 13 }}>Produtos e serviços</span>
                <BadgeOrigem origem={origens.produtos} />
              </div>
              {d.produtos.length === 0 ? (
                <p className="muted" style={{ margin: 0 }}>Não informado.</p>
              ) : (
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {d.produtos.map((p, i) => (
                    <li key={i} style={{ marginBottom: 4 }}>
                      {p.nome}{" "}
                      {p.prioritario && <span className="badge t-brand">Prioritário</span>}
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <CampoLeitura
              campo="diferencial"
              rotulo="Diferencial"
              valor={d.diferencial}
              origem={origens.diferencial}
            />
            <CampoLeitura
              campo="modeloNegocio"
              rotulo="Modelo de negócio"
              valor={textoModelo}
              origem={origens.modeloNegocio}
            />
            <CampoLeitura
              campo="regioes"
              rotulo="Regiões e idiomas"
              valor={d.regioes}
              origem={origens.regioes}
            />
            <CampoLeitura
              campo="comoCompram"
              rotulo="Como os clientes compram"
              valor={d.comoCompram}
              origem={origens.comoCompram}
            />
          </section>

          <section className="card stack">
            <h2 style={{ margin: 0 }}>Estratégia</h2>

            <CampoLeitura
              campo="maiorDesafio"
              rotulo="Maior desafio"
              valor={d.maiorDesafio}
              origem={origens.maiorDesafio}
            />
            <CampoLeitura
              campo="objetivo90Dias"
              rotulo="Objetivo dos próximos 90 dias"
              valor={d.objetivo90Dias}
              origem={origens.objetivo90Dias}
            />
            <CampoLeitura
              campo="canais"
              rotulo="Canais"
              valor={listaCanais.join(", ")}
              origem={origens.canais}
            />
            <CampoLeitura
              campo="restricoesSetor"
              rotulo="Restrições do setor"
              valor={d.restricoesSetor}
              origem={origens.restricoesSetor}
            />
          </section>
        </div>

        <div className="stack" style={{ gap: 16 }}>
          <section className="card stack" data-testid="versao-campo-pilares">
            <div className="row" style={{ justifyContent: "space-between" }}>
              <h2 style={{ margin: 0 }}>Pilares de conteúdo</h2>
              <BadgeOrigem origem={origens.pilares} />
            </div>
            {d.pilares.length === 0 ? (
              <p className="muted" style={{ margin: 0 }}>Nenhum pilar definido.</p>
            ) : (
              <ol style={{ margin: 0, paddingLeft: 18 }}>
                {d.pilares.map((p, i) => (
                  <li key={i} style={{ marginBottom: 6, whiteSpace: "pre-wrap" }}>
                    {p}
                  </li>
                ))}
              </ol>
            )}
          </section>

          <section className="card stack" data-testid="versao-campo-cores">
            <div className="row" style={{ justifyContent: "space-between" }}>
              <h2 style={{ margin: 0 }}>Cores</h2>
              <BadgeOrigem origem={origens.cores} />
            </div>
            {d.cores.length === 0 ? (
              <p className="muted" style={{ margin: 0 }}>Nenhuma cor definida.</p>
            ) : (
              <div className="stack" style={{ gap: 8 }}>
                {d.cores.map((c, i) => (
                  <div key={i} className="row">
                    <span
                      aria-hidden
                      style={{
                        width: 24,
                        height: 24,
                        borderRadius: 6,
                        background: c,
                        border: "1px solid var(--laura-border)",
                      }}
                    />
                    <span className="num">{c}</span>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}

function CampoLeitura({
  campo,
  rotulo,
  valor,
  origem,
}: {
  campo: CampoKit;
  rotulo: string;
  valor: string;
  origem?: OrigemCampo;
}) {
  return (
    <div className="stack" style={{ gap: 4 }} data-testid={`versao-campo-${campo}`}>
      <div className="row" style={{ justifyContent: "space-between" }}>
        <span style={{ fontWeight: 500, fontSize: 13 }}>{rotulo}</span>
        <BadgeOrigem origem={origem} />
      </div>
      <p style={{ margin: 0, whiteSpace: "pre-wrap" }}>
        {valor.trim() ? valor : <span className="muted">Não informado.</span>}
      </p>
    </div>
  );
}
