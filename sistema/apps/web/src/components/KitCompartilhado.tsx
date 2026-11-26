"use client";
// Componentes e metadados compartilhados entre a tela do kit (/kit) e a visualização de versões (/kit/versoes).
import { AlertTriangle, Check } from "lucide-react";
import type { CampoKit, OrigemCampo } from "@laura-growth/contratos";
import { formatarDataHora } from "@/lib/formato";

export type CampoTextoSimples =
  | "site"
  | "perfis"
  | "tomDeVoz"
  | "publico"
  | "temas"
  | "nuncaDizer"
  | "fontes"
  | "exemplos"
  | "diferencial"
  | "regioes"
  | "comoCompram"
  | "maiorDesafio"
  | "objetivo90Dias"
  | "restricoesSetor";

export type DefinicaoCampoTexto = {
  k: CampoTextoSimples;
  rotulo: string;
  dica?: string;
  longo?: boolean;
};

export const CAMPOS_COMUNICACAO: DefinicaoCampoTexto[] = [
  { k: "site", rotulo: "Site da marca", dica: "Endereço completo, começando com https://" },
  { k: "perfis", rotulo: "Perfis nas redes", dica: "Um por linha", longo: true },
  { k: "tomDeVoz", rotulo: "Tom de voz", longo: true },
  { k: "publico", rotulo: "Público", longo: true },
  { k: "temas", rotulo: "Temas", dica: "Separados por vírgula" },
  { k: "nuncaDizer", rotulo: "O que nunca dizer", dica: "Separado por vírgula" },
  { k: "fontes", rotulo: "Fontes" },
  { k: "exemplos", rotulo: "Exemplos de posts", longo: true },
];

export const ROTULOS_CAMPOS_KIT: Record<CampoKit, string> = {
  site: "Site da marca",
  perfis: "Perfis nas redes",
  tomDeVoz: "Tom de voz",
  publico: "Público",
  temas: "Temas",
  nuncaDizer: "O que nunca dizer",
  fontes: "Fontes",
  exemplos: "Exemplos de posts",
  pilares: "Pilares de conteúdo",
  cores: "Cores",
  produtos: "Produtos e serviços",
  diferencial: "Diferencial",
  modeloNegocio: "Modelo de negócio",
  regioes: "Regiões e idiomas",
  comoCompram: "Como os clientes compram",
  maiorDesafio: "Maior desafio",
  objetivo90Dias: "Objetivo dos próximos 90 dias",
  canais: "Canais",
  restricoesSetor: "Restrições do setor",
};

export function rotuloCampoKit(campo: string): string {
  if (campo in ROTULOS_CAMPOS_KIT) {
    return ROTULOS_CAMPOS_KIT[campo as CampoKit];
  }
  if (campo === "logo") return "Logo";
  if (campo.startsWith("logo:")) return `Logo (${campo.slice(5)})`;
  if (campo === "arquivos") return "Arquivos de referência";
  if (campo.startsWith("arquivos:")) return `Arquivo (${campo.slice(9)})`;
  return campo;
}

export function BadgeOrigem({ origem }: { origem?: OrigemCampo }) {
  if (!origem) return null;
  if (origem.origem === "sugerido_ia") {
    return (
      <span className="badge t-ai" title={`Sugerido pela IA em ${formatarDataHora(origem.definidoEm)}`}>
        Sugerido pela IA
      </span>
    );
  }
  const detalhe = origem.definidoPor
    ? `Confirmado por ${origem.definidoPor} em ${formatarDataHora(origem.definidoEm)}`
    : `Confirmado em ${formatarDataHora(origem.definidoEm)}`;
  return (
    <span className="badge t-success" title={detalhe}>
      Confirmado por você
    </span>
  );
}

export function CabecalhoCampoKit({
  campo,
  htmlFor,
  titulo,
  origem,
  podeEditar = false,
  alteradoLocal = false,
  confirmando = false,
  onConfirmar,
  comoTitulo = false,
}: {
  campo: CampoKit;
  htmlFor?: string;
  titulo?: string;
  origem?: OrigemCampo;
  podeEditar?: boolean;
  alteradoLocal?: boolean;
  confirmando?: boolean;
  onConfirmar?: (campo: CampoKit) => void;
  comoTitulo?: boolean;
}) {
  const texto = titulo ?? ROTULOS_CAMPOS_KIT[campo];
  const mostrarConfirmar = podeEditar && origem?.origem === "sugerido_ia" && Boolean(onConfirmar);

  return (
    <div className="row" style={{ justifyContent: "space-between", alignItems: "center", gap: 8 }}>
      {comoTitulo ? (
        <h2 style={{ margin: 0 }}>{texto}</h2>
      ) : htmlFor ? (
        <label htmlFor={htmlFor} style={{ fontWeight: 500, fontSize: 13 }}>
          {texto}
        </label>
      ) : (
        <span style={{ fontWeight: 500, fontSize: 13 }}>{texto}</span>
      )}
      <div className="row" style={{ gap: 6 }}>
        <BadgeOrigem origem={origem} />
        {mostrarConfirmar && (
          <button
            type="button"
            className="btn peq"
            disabled={confirmando || alteradoLocal}
            title={alteradoLocal ? "Salve o kit para confirmar a edição" : `Confirmar ${texto}`}
            onClick={() => onConfirmar?.(campo)}
          >
            <Check size={14} aria-hidden />
            {confirmando ? "Confirmando…" : "Confirmar"}
          </button>
        )}
      </div>
    </div>
  );
}

export function BlocoOQueEvitar({
  nuncaDizer,
  restricoesSetor,
}: {
  nuncaDizer: string;
  restricoesSetor: string;
}) {
  return (
    <section
      className="card stack"
      data-testid="bloco-o-que-evitar"
      aria-labelledby="titulo-o-que-evitar"
      style={{
        marginBottom: 16,
        background: "var(--laura-warning-soft)",
        borderColor: "var(--laura-brand-border)",
      }}
    >
      <div className="row" style={{ gap: 8 }}>
        <AlertTriangle size={18} aria-hidden style={{ color: "var(--laura-warning)", flex: "none" }} />
        <h2 id="titulo-o-que-evitar" style={{ margin: 0 }}>
          O que evitar
        </h2>
      </div>
      <div className="grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))", gap: 12 }}>
        <div>
          <div className="small muted" style={{ fontWeight: 600, marginBottom: 4 }}>
            O que nunca dizer
          </div>
          <p style={{ margin: 0, whiteSpace: "pre-wrap" }}>
            {nuncaDizer.trim() ? nuncaDizer : <span className="muted">Nada informado.</span>}
          </p>
        </div>
        <div>
          <div className="small muted" style={{ fontWeight: 600, marginBottom: 4 }}>
            Restrições do setor
          </div>
          <p style={{ margin: 0, whiteSpace: "pre-wrap" }}>
            {restricoesSetor.trim() ? restricoesSetor : <span className="muted">Nenhuma restrição informada.</span>}
          </p>
        </div>
      </div>
    </section>
  );
}
