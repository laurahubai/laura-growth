import {
  rotuloAlcada,
  type Alcada,
  type EstadoEnvioConvite,
  type RegistroAcesso,
  type SituacaoLimite,
} from "@laura-growth/contratos";

const fmtUsd2 = new Intl.NumberFormat("pt-BR", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const fmtUsd6 = new Intl.NumberFormat("pt-BR", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 6,
  maximumFractionDigits: 6,
});

const fmtPrecoFlex = new Intl.NumberFormat("pt-BR", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
  maximumFractionDigits: 6,
});

export function formatarUsd(valor: number): string {
  if (valor > 0 && valor < 0.01) return fmtUsd6.format(valor);
  return fmtUsd2.format(valor);
}

export function formatarPrecoUsd(valor: number): string {
  if (valor > 0 && valor < 0.01) return fmtUsd6.format(valor);
  return fmtPrecoFlex.format(valor);
}

export function formatarDataHora(iso: string): string {
  return new Date(iso).toLocaleString("pt-BR");
}

export function formatarDataIso(data: string): string {
  const partes = data.split("-");
  if (partes.length !== 3) return data;
  return `${partes[2]}/${partes[1]}/${partes[0]}`;
}

export function hojeIso(): string {
  try {
    return new Intl.DateTimeFormat("en-CA", { timeZone: "America/Sao_Paulo" }).format(new Date());
  } catch {
    const d = new Date();
    const ano = d.getFullYear();
    const mes = String(d.getMonth() + 1).padStart(2, "0");
    const dia = String(d.getDate()).padStart(2, "0");
    return `${ano}-${mes}-${dia}`;
  }
}

export function etiquetaSituacao(situacao: SituacaoLimite): { texto: string; classe: string } {
  if (situacao === "limite") return { texto: "No limite", classe: "t-danger" };
  if (situacao === "aviso") return { texto: "Aviso", classe: "t-warning" };
  return { texto: "Normal", classe: "t-success" };
}

const PREFIXOS_IMAGEM_SEGUROS = [
  "data:image/png;base64,",
  "data:image/jpeg;base64,",
  "data:image/webp;base64,",
];

export function imagemDataUrlSegura(url: string): boolean {
  return PREFIXOS_IMAGEM_SEGUROS.some((p) => url.startsWith(p));
}

export function etiquetaEnvioConvite(estado: EstadoEnvioConvite): { texto: string; classe: string } {
  if (estado === "enviado") return { texto: "Enviado", classe: "t-success" };
  if (estado === "falhou") return { texto: "Não enviado", classe: "t-danger" };
  return { texto: "Pendente", classe: "t-warning" };
}

const ROTULOS_ACAO_ACESSO: Record<string, string> = {
  alcada_concedida: "Alçada concedida",
  alcada_revogada: "Alçada revogada",
  membro_removido: "Pessoa removida",
  convite_criado: "Convite criado",
  convite_reenviado: "Convite reenviado",
  convite_revogado: "Convite cancelado",
  convite_aceito: "Convite aceito",
  admin_plataforma_concedido: "Administração concedida",
  admin_plataforma_revogado: "Administração revogada",
};

export function rotuloAcaoAcesso(acao: string): string {
  return ROTULOS_ACAO_ACESSO[acao] ?? acao;
}

function nomeAlcadaSeguro(codigo: unknown): string {
  if (typeof codigo !== "string") return "";
  return rotuloAlcada[codigo as Alcada] ?? codigo;
}

export function descreverDetalheAcesso(reg: RegistroAcesso): string {
  const d = reg.detalhe ?? {};
  let base = "—";
  if (typeof d.alcada === "string") {
    base = nomeAlcadaSeguro(d.alcada);
  } else if (Array.isArray(d.alcadas) && d.alcadas.length > 0) {
    base = d.alcadas.map(nomeAlcadaSeguro).filter(Boolean).join(", ");
  } else if (Array.isArray(d.alcadasAnteriores) && d.alcadasAnteriores.length > 0) {
    base = `Alçadas anteriores: ${d.alcadasAnteriores.map(nomeAlcadaSeguro).filter(Boolean).join(", ")}`;
  } else if (typeof d.adminPlataforma === "boolean") {
    base = d.adminPlataforma ? "Tornou-se administrador da plataforma" : "Deixou de ser administrador da plataforma";
  } else if (typeof d.envioEstado === "string") {
    const e = d.envioEstado as EstadoEnvioConvite;
    if (e === "enviado" || e === "falhou" || e === "pendente") {
      base = `Envio: ${etiquetaEnvioConvite(e).texto.toLowerCase()}`;
    }
  }
  if (d.via_admin_plataforma === true) {
    return base === "—" ? "Via administração da plataforma" : `${base} · via administração da plataforma`;
  }
  return base;
}
