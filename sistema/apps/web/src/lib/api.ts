"use client";
// Conversa com a API (/api/...) e guarda as respostas em cache (TanStack Query).
// Tudo o que pertence a uma organização usa a chave ["org", idDaOrganização, ...]:
// ao trocar de organização, nada da anterior aparece na tela.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ErroApi, OrganizacaoDetalhe, Sessao, type PedidoEntrar } from "@laura-growth/contratos";
import type { z } from "zod";

export class FalhaApi extends Error {
  constructor(public status: number, public codigo: string, mensagem: string) {
    super(mensagem);
  }
}

async function chamar<T>(metodo: string, caminho: string, esquema: z.ZodType<T> | null, corpo?: unknown): Promise<T> {
  const res = await fetch(caminho, {
    method: metodo,
    credentials: "same-origin",
    headers: corpo === undefined ? undefined : { "Content-Type": "application/json" },
    body: corpo === undefined ? undefined : JSON.stringify(corpo),
  });
  if (!res.ok) {
    const e = ErroApi.safeParse(await res.json().catch(() => null));
    throw new FalhaApi(res.status, e.success ? e.data.codigo : "falha", e.success ? e.data.mensagem : "Algo deu errado. Tente de novo.");
  }
  if (!esquema) return undefined as T;
  return esquema.parse(await res.json());
}

const CHAVE_SESSAO = ["sessao"] as const;

export function useSessao() {
  return useQuery({
    queryKey: CHAVE_SESSAO,
    queryFn: () => chamar("GET", "/api/sessao", Sessao),
    retry: (n, e) => !(e instanceof FalhaApi && e.status === 401) && n < 2,
  });
}

export function useEntrar() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: PedidoEntrar) => chamar("POST", "/api/sessao", Sessao, p),
    onSuccess: (s) => {
      qc.clear();
      qc.setQueryData(CHAVE_SESSAO, s);
    },
  });
}

export function useSair() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => chamar("DELETE", "/api/sessao", null),
    onSettled: () => qc.clear(),
  });
}

export function useTrocarOrganizacao() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (organizacaoId: string) => chamar("PUT", "/api/sessao/organizacao", Sessao, { organizacaoId }),
    onSuccess: (s) => {
      // Descarta tudo o que era da organização anterior.
      qc.removeQueries({ queryKey: ["org"] });
      qc.setQueryData(CHAVE_SESSAO, s);
    },
  });
}

export function useOrganizacaoAtiva(orgId: string) {
  return useQuery({
    queryKey: ["org", orgId, "detalhe"],
    queryFn: () => chamar("GET", "/api/organizacao", OrganizacaoDetalhe),
  });
}
