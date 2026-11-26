"use client";
// Conversa com a API (/api/...) e guarda as respostas em cache (TanStack Query).
// Tudo o que pertence a uma organização usa a chave ["org", idDaOrganização, ...]:
// ao trocar de organização, nada da anterior aparece na tela.
// Tudo o que pertence à Administração da plataforma usa a chave ["admin", ...].
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArquivoMarca,
  ConvitePublico,
  ErroApi,
  Fornecedor,
  MembroEquipe,
  Modelo,
  OrganizacaoDetalhe,
  PedidoConfirmarKit,
  PessoaMembro,
  Preco,
  RespostaAtividades,
  RespostaAvisos,
  RespostaConviteCriado,
  RespostaEquipeAdmin,
  RespostaExecucoes,
  RespostaFornecedores,
  RespostaKit,
  RespostaLerModelos,
  RespostaModelos,
  RespostaOrganizacoesAdmin,
  RespostaPessoas,
  RespostaPrecos,
  RespostaRegistroAcessos,
  RespostaVersoesKit,
  ResultadoAtividade,
  ResultadoTeste,
  RotaAtividade,
  Sessao,
  VersaoKit,
  type Atividade,
  type KitMarca,
  type PedidoAceitarConvite,
  type PedidoAtualizarEquipeAdmin,
  type PedidoCriarConvite,
  type PedidoEditarFornecedor,
  type PedidoEditarModelo,
  type PedidoEntrar,
  type PedidoNovoFornecedor,
  type PedidoNovoModelo,
  type PedidoNovoPreco,
  type PedidoRotaAtividade,
  type PedidoSalvarAlcadas,
} from "@laura-growth/contratos";
import type { z } from "zod";

export class FalhaApi extends Error {
  constructor(public status: number, public codigo: string, mensagem: string) {
    super(mensagem);
  }
}

export function erroSemPermissao(e: unknown): boolean {
  return e instanceof FalhaApi && e.status === 403 && e.codigo === "sem_permissao";
}

async function chamar<T>(metodo: string, caminho: string, esquema: z.ZodType<T> | null, corpo?: unknown): Promise<T> {
  const formulario = corpo instanceof FormData;
  const res = await fetch(caminho, {
    method: metodo,
    credentials: "same-origin",
    headers: corpo === undefined || formulario ? undefined : { "Content-Type": "application/json" },
    body: corpo === undefined ? undefined : formulario ? corpo : JSON.stringify(corpo),
  });
  if (!res.ok) {
    if (res.status === 413) throw new FalhaApi(413, "arquivo_grande", "O arquivo é grande demais.");
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

// ---- Kit de marca (etapas 2 e 2C) ----

export type PedidoSalvarKit = Omit<KitMarca, "atualizadoEm"> & {
  versaoBase: number;
};

export function useKit(orgId: string) {
  return useQuery({
    queryKey: ["org", orgId, "kit"],
    queryFn: () => chamar("GET", "/api/kit", RespostaKit),
    enabled: Boolean(orgId),
    retry: tentarSemBloqueio,
    refetchOnWindowFocus: false,
  });
}

export function useSalvarKit(orgId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (kit: PedidoSalvarKit) => chamar("PUT", "/api/kit", RespostaKit, kit),
    onSuccess: (r) => {
      qc.setQueryData(["org", orgId, "kit"], r);
      qc.invalidateQueries({ queryKey: ["org", orgId, "kit-versoes"] });
    },
  });
}

export function useConfirmarKit(orgId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (dados: PedidoConfirmarKit) =>
      chamar("POST", "/api/kit/confirmar", RespostaKit, PedidoConfirmarKit.parse(dados)),
    onSuccess: (r) => {
      qc.setQueryData(["org", orgId, "kit"], r);
      qc.invalidateQueries({ queryKey: ["org", orgId, "kit-versoes"] });
    },
  });
}

export function useVersoesKit(orgId: string, pagina = 1) {
  return useQuery({
    queryKey: ["org", orgId, "kit-versoes", pagina],
    queryFn: () => chamar("GET", `/api/kit/versoes?pagina=${pagina}`, RespostaVersoesKit),
    enabled: Boolean(orgId),
    retry: tentarSemBloqueio,
  });
}

export function useVersaoKit(orgId: string, numero: number | null) {
  return useQuery({
    queryKey: ["org", orgId, "kit-versao", numero],
    queryFn: () => chamar("GET", `/api/kit/versoes/${numero}`, VersaoKit),
    enabled: Boolean(orgId) && numero !== null && Number.isInteger(numero) && numero > 0,
    retry: tentarSemBloqueio,
  });
}

export function useEnviarArquivo(orgId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tipo, arquivo }: { tipo: "logo" | "referencia"; arquivo: File }) => {
      const f = new FormData();
      f.append("arquivo", arquivo);
      const q = new URLSearchParams({ tipo, organizacaoId: orgId });
      return chamar("POST", `/api/kit/arquivos?${q}`, ArquivoMarca, f);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["org", orgId, "kit"] });
      qc.invalidateQueries({ queryKey: ["org", orgId, "kit-versoes"] });
    },
  });
}

export function useRemoverArquivo(orgId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => chamar("DELETE", `/api/kit/arquivos/${encodeURIComponent(id)}?${new URLSearchParams({ organizacaoId: orgId })}`, null),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["org", orgId, "kit"] });
      qc.invalidateQueries({ queryKey: ["org", orgId, "kit-versoes"] });
    },
  });
}

export const urlArquivo = (id: string, baixar = false) => `/api/kit/arquivos/${encodeURIComponent(id)}${baixar ? "?baixar=1" : ""}`;

// ---- Administração da plataforma (etapa 2A-1) ----

const tentarSemBloqueio = (n: number, e: Error) =>
  !(e instanceof FalhaApi && (e.status === 401 || e.status === 403 || e.status === 404)) && n < 2;

const tentarAdmin = tentarSemBloqueio;

export function useAvisosAdmin(habilitado = true) {
  return useQuery({
    queryKey: ["admin", "avisos"],
    queryFn: () => chamar("GET", "/api/admin/avisos", RespostaAvisos),
    enabled: habilitado,
    retry: tentarAdmin,
  });
}

export function useFornecedoresAdmin() {
  return useQuery({
    queryKey: ["admin", "fornecedores"],
    queryFn: () => chamar("GET", "/api/admin/fornecedores", RespostaFornecedores),
    retry: tentarAdmin,
  });
}

export function useCriarFornecedor() {
  const qc = useQueryClient();
  return useMutation({
    gcTime: 0,
    mutationFn: (p: PedidoNovoFornecedor) => chamar("POST", "/api/admin/fornecedores", Fornecedor, p),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores"] });
      qc.invalidateQueries({ queryKey: ["admin", "avisos"] });
    },
  });
}

export function useEditarFornecedor() {
  const qc = useQueryClient();
  return useMutation({
    gcTime: 0,
    mutationFn: ({ id, dados }: { id: string; dados: PedidoEditarFornecedor }) =>
      chamar("PATCH", `/api/admin/fornecedores/${encodeURIComponent(id)}`, Fornecedor, dados),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores"] });
      qc.invalidateQueries({ queryKey: ["admin", "avisos"] });
      qc.invalidateQueries({ queryKey: ["admin", "atividades"] });
    },
  });
}

export function useLerModelos(fornecedorId: string) {
  return useMutation({
    mutationFn: () =>
      chamar("POST", `/api/admin/fornecedores/${encodeURIComponent(fornecedorId)}/ler-modelos`, RespostaLerModelos),
  });
}

export function useModelosFornecedor(fornecedorId: string) {
  return useQuery({
    queryKey: ["admin", "fornecedores", fornecedorId, "modelos"],
    queryFn: () =>
      chamar("GET", `/api/admin/fornecedores/${encodeURIComponent(fornecedorId)}/modelos`, RespostaModelos),
    enabled: Boolean(fornecedorId),
    retry: tentarAdmin,
  });
}

export function useTodosModelosAdmin(fornecedorIds: string[]) {
  return useQueries({
    queries: fornecedorIds.map((id) => ({
      queryKey: ["admin", "fornecedores", id, "modelos"] as const,
      queryFn: () =>
        chamar("GET", `/api/admin/fornecedores/${encodeURIComponent(id)}/modelos`, RespostaModelos),
      enabled: Boolean(id),
      retry: tentarAdmin,
    })),
  });
}

export function useCriarModelo(fornecedorId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: PedidoNovoModelo) =>
      chamar("POST", `/api/admin/fornecedores/${encodeURIComponent(fornecedorId)}/modelos`, Modelo, p),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores", fornecedorId, "modelos"] });
      qc.invalidateQueries({ queryKey: ["admin", "atividades"] });
    },
  });
}

export function useEditarModelo(fornecedorId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, dados }: { id: string; dados: PedidoEditarModelo }) =>
      chamar("PATCH", `/api/admin/modelos/${encodeURIComponent(id)}`, Modelo, dados),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores", fornecedorId, "modelos"] });
      qc.invalidateQueries({ queryKey: ["admin", "atividades"] });
      qc.invalidateQueries({ queryKey: ["admin", "avisos"] });
    },
  });
}

export function usePrecosModelo(modeloId: string) {
  return useQuery({
    queryKey: ["admin", "modelos", modeloId, "precos"],
    queryFn: () => chamar("GET", `/api/admin/modelos/${encodeURIComponent(modeloId)}/precos`, RespostaPrecos),
    enabled: Boolean(modeloId),
    retry: tentarAdmin,
  });
}

export function useCriarPreco(modeloId: string, fornecedorId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: PedidoNovoPreco) =>
      chamar("POST", `/api/admin/modelos/${encodeURIComponent(modeloId)}/precos`, Preco, p),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "modelos", modeloId, "precos"] });
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores", fornecedorId, "modelos"] });
      qc.invalidateQueries({ queryKey: ["admin", "atividades"] });
    },
  });
}

export function useTestarModelo(fornecedorId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (modeloId: string) =>
      chamar("POST", `/api/admin/modelos/${encodeURIComponent(modeloId)}/testar`, ResultadoTeste),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["admin", "avisos"] });
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores"] });
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores", fornecedorId, "modelos"] });
      qc.invalidateQueries({ queryKey: ["admin", "execucoes"] });
    },
  });
}

export function useAtividadesAdmin() {
  return useQuery({
    queryKey: ["admin", "atividades"],
    queryFn: () => chamar("GET", "/api/admin/atividades", RespostaAtividades),
    retry: tentarAdmin,
  });
}

export function useSalvarAtividade() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ atividade, dados }: { atividade: Atividade; dados: PedidoRotaAtividade }) =>
      chamar("PUT", `/api/admin/atividades/${encodeURIComponent(atividade)}`, RotaAtividade, dados),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "atividades"] });
    },
  });
}

export function useTestarAtividade() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (atividade: Atividade) =>
      chamar("POST", `/api/admin/atividades/${encodeURIComponent(atividade)}/testar`, ResultadoAtividade),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["admin", "avisos"] });
      qc.invalidateQueries({ queryKey: ["admin", "fornecedores"] });
      qc.invalidateQueries({ queryKey: ["admin", "atividades"] });
      qc.invalidateQueries({ queryKey: ["admin", "execucoes"] });
    },
  });
}

export function useExecucoesAdmin(limite = 100) {
  return useQuery({
    queryKey: ["admin", "execucoes", limite],
    queryFn: () => chamar("GET", `/api/admin/execucoes?limite=${limite}`, RespostaExecucoes),
    retry: tentarAdmin,
  });
}

// ---- Acessos, Pessoas e Convites (etapa 2B) ----

export type EscopoPessoas = "org" | "admin";

function chaveEscopoPessoas(orgId: string, escopo: EscopoPessoas) {
  return escopo === "org" ? (["org", orgId] as const) : (["admin", "organizacoes", orgId] as const);
}

function invalidarEscopoPessoas(qc: ReturnType<typeof useQueryClient>, orgId: string, escopo: EscopoPessoas) {
  qc.invalidateQueries({ queryKey: [...chaveEscopoPessoas(orgId, escopo), "pessoas"] });
  qc.invalidateQueries({ queryKey: [...chaveEscopoPessoas(orgId, escopo), "registro-acessos"] });
  if (escopo === "admin") {
    qc.invalidateQueries({ queryKey: ["org", orgId, "pessoas"] });
    qc.invalidateQueries({ queryKey: ["org", orgId, "registro-acessos"] });
    qc.invalidateQueries({ queryKey: CHAVE_SESSAO });
  }
}

export function usePessoas(orgId: string, baseApi = "/api", escopo: EscopoPessoas = "org") {
  return useQuery({
    queryKey: [...chaveEscopoPessoas(orgId, escopo), "pessoas"],
    queryFn: () => chamar("GET", `${baseApi}/pessoas`, RespostaPessoas),
    enabled: Boolean(orgId),
    retry: tentarSemBloqueio,
  });
}

export function useRegistroAcessos(orgId: string, limite = 100, baseApi = "/api", escopo: EscopoPessoas = "org") {
  return useQuery({
    queryKey: [...chaveEscopoPessoas(orgId, escopo), "registro-acessos", limite],
    queryFn: () => chamar("GET", `${baseApi}/pessoas/registro?limite=${limite}`, RespostaRegistroAcessos),
    enabled: Boolean(orgId),
    retry: tentarSemBloqueio,
  });
}

export function useSalvarAlcadasPessoa(orgId: string, baseApi = "/api", escopo: EscopoPessoas = "org") {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ usuarioId, alcadas }: { usuarioId: string } & PedidoSalvarAlcadas) =>
      chamar("PUT", `${baseApi}/pessoas/${encodeURIComponent(usuarioId)}/alcadas`, PessoaMembro, { alcadas }),
    onSuccess: () => {
      invalidarEscopoPessoas(qc, orgId, escopo);
    },
  });
}

export function useRemoverPessoa(orgId: string, baseApi = "/api", escopo: EscopoPessoas = "org") {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (usuarioId: string) =>
      chamar("DELETE", `${baseApi}/pessoas/${encodeURIComponent(usuarioId)}`, null),
    onSuccess: () => {
      invalidarEscopoPessoas(qc, orgId, escopo);
      qc.invalidateQueries({ queryKey: ["org", orgId, "detalhe"] });
      qc.invalidateQueries({ queryKey: ["admin", "organizacoes"] });
    },
  });
}

export function useCriarConvite(orgId: string, baseApi = "/api", escopo: EscopoPessoas = "org") {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (dados: PedidoCriarConvite) =>
      chamar("POST", `${baseApi}/convites`, RespostaConviteCriado, dados),
    onSuccess: () => {
      invalidarEscopoPessoas(qc, orgId, escopo);
    },
  });
}

export function useReenviarConvite(orgId: string, baseApi = "/api", escopo: EscopoPessoas = "org") {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      chamar("POST", `${baseApi}/convites/${encodeURIComponent(id)}/reenviar`, RespostaConviteCriado),
    onSuccess: () => {
      invalidarEscopoPessoas(qc, orgId, escopo);
    },
  });
}

export function useCancelarConvite(orgId: string, baseApi = "/api", escopo: EscopoPessoas = "org") {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      chamar("DELETE", `${baseApi}/convites/${encodeURIComponent(id)}`, null),
    onSuccess: () => {
      invalidarEscopoPessoas(qc, orgId, escopo);
    },
  });
}

export function useOrganizacoesAdmin() {
  return useQuery({
    queryKey: ["admin", "organizacoes"],
    queryFn: () => chamar("GET", "/api/admin/organizacoes", RespostaOrganizacoesAdmin),
    retry: tentarAdmin,
  });
}

export function useConvitePublico(token: string) {
  return useQuery({
    queryKey: ["convite", token],
    queryFn: () => chamar("GET", `/api/convites/${encodeURIComponent(token)}`, ConvitePublico),
    enabled: Boolean(token),
    retry: tentarSemBloqueio,
  });
}

export function useAceitarConvite(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (dados: PedidoAceitarConvite) =>
      chamar("POST", `/api/convites/${encodeURIComponent(token)}/aceitar`, Sessao, dados),
    onSuccess: (s) => {
      qc.clear();
      qc.setQueryData(CHAVE_SESSAO, s);
    },
  });
}

export function useEquipeAdmin() {
  return useQuery({
    queryKey: ["admin", "equipe"],
    queryFn: () => chamar("GET", "/api/admin/equipe", RespostaEquipeAdmin),
    retry: tentarAdmin,
  });
}

export function useAtualizarEquipeAdmin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ usuarioId, dados }: { usuarioId: string; dados: PedidoAtualizarEquipeAdmin }) =>
      chamar("PUT", `/api/admin/equipe/${encodeURIComponent(usuarioId)}`, MembroEquipe, dados),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "equipe"] });
      qc.invalidateQueries({ queryKey: ["sessao"] });
    },
  });
}
