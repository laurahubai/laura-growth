import type { SituacaoLimite } from "@laura-growth/contratos";

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
