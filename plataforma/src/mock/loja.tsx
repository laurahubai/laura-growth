"use client";
// Camada única de dados do mock. Para ligar o backend, trocar as ações abaixo por chamadas à API.
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { Dados, Etapa, KitMarca, Organizacao, Pauta, Post, Rede, Texto, Versao, Visual } from "./types";
import { dadosDeExemplo, VERSAO_DADOS } from "./exemplo";
import { consumoFicticio, gerarTexto, gerarVisual, passosAoVivo, revisar, sugerirPauta, type ItemDoPlano } from "./agente-falso";

const CHAVE = "laura-growth-mock";
const CHAVE_SESSAO = "laura-growth-mock-sessao";
const espera = (ms: number) => new Promise((r) => setTimeout(r, ms));
const agora = () => new Date().toISOString();
const NOME_ETAPA: Record<number, string> = { 1: "Pauta", 2: "Texto", 3: "Visual", 4: "Revisão" };
const novoId = (p: string) => `${p}-${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;

interface Sessao {
  email: string;
  orgId: string;
}

function ler<T>(k: string): T | null {
  try {
    const v = localStorage.getItem(k);
    return v ? (JSON.parse(v) as T) : null;
  } catch {
    return null;
  }
}
function gravar(k: string, v: unknown) {
  try {
    if (v === null) localStorage.removeItem(k);
    else localStorage.setItem(k, JSON.stringify(v));
  } catch {
    /* navegador sem armazenamento: segue só em memória */
  }
}

function useLojaInterna() {
  const [dados, setDados] = useState<Dados | null>(null);
  const [sessao, setSessao] = useState<Sessao | null>(null);
  const [pronto, setPronto] = useState(false);
  const ref = useRef<Dados | null>(null);
  ref.current = dados;

  useEffect(() => {
    const d = ler<Dados>(CHAVE);
    setDados(d && d.versao === VERSAO_DADOS ? { ...d, posts: d.posts.map((p) => ({ ...p, trabalhando: false })) } : dadosDeExemplo());
    setSessao(ler<Sessao>(CHAVE_SESSAO));
    setPronto(true);
  }, []);
  useEffect(() => {
    if (dados) gravar(CHAVE, dados);
  }, [dados]);
  useEffect(() => {
    if (pronto) gravar(CHAVE_SESSAO, sessao);
  }, [sessao, pronto]);

  const quem = sessao?.email ?? "Você";

  const mudarPost = useCallback((id: string, f: (p: Post) => Post) => {
    setDados((d) => (d ? { ...d, posts: d.posts.map((p) => (p.id === id ? f(p) : p)) } : d));
  }, []);

  /** Simula o agente trabalhando ao vivo e grava custo (CST-01). */
  const rodarAgente = useCallback(
    async (postId: string, etapa: Etapa, comentario?: string, encadeado = false): Promise<void> => {
      const { agente, passos } = passosAoVivo(etapa, !!comentario);
      mudarPost(postId, (p) => ({ ...p, trabalhando: true, atividade: [...(encadeado ? p.atividade : []), { em: agora(), texto: `${agente} começou: ${NOME_ETAPA[etapa]}` }] }));
      for (const passo of passos) {
        await espera(650);
        mudarPost(postId, (p) => ({ ...p, atividade: [...p.atividade, { em: agora(), texto: passo }] }));
      }
      await espera(500);
      const d = ref.current!;
      const post = d.posts.find((p) => p.id === postId)!;
      const org = d.orgs.find((o) => o.id === post.orgId)!;
      const novo: Partial<Post> = {};
      let anterior: Versao["conteudo"] | undefined;
      if (etapa === 1) {
        anterior = post.pauta.tema.startsWith("(") ? undefined : post.pauta;
        novo.pauta = comentario ? { ...post.pauta, tema: `${post.pauta.tema} (${comentario})` } : sugerirPauta(org);
      } else if (etapa === 2) {
        anterior = post.texto;
        novo.texto = gerarTexto(org, post.pauta, comentario, post.texto);
      } else if (etapa === 3) {
        anterior = post.visual;
        novo.visual = gerarVisual(post.pauta, post.visual);
      } else if (etapa === 4) {
        anterior = post.revisao;
        novo.revisao = revisar(org, post);
      }
      const c = consumoFicticio(etapa, post.pauta.formato === "carrossel" ? 4 : 1);
      setDados((dd) =>
        dd
          ? {
              ...dd,
              posts: dd.posts.map((p) =>
                p.id !== postId
                  ? p
                  : {
                      ...p,
                      ...novo,
                      trabalhando: false,
                      versoes: anterior
                        ? [...p.versoes, { id: novoId("v"), etapa, autor: "ia", em: agora(), comentario, conteudo: anterior }]
                        : p.versoes,
                      atividade: [...p.atividade, { em: agora(), texto: p.piloto ? `${NOME_ETAPA[etapa]}: concluída.` : "Pronto. Aguardando sua revisão." }],
                      historico: [...p.historico, { em: agora(), quem: agente, acao: comentario ? `Gerou nova versão (etapa ${etapa})` : `Concluiu a etapa ${etapa}` }],
                    },
              ),
              custos: [
                ...dd.custos,
                {
                  id: novoId("custo"),
                  orgId: post.orgId,
                  postId,
                  agente,
                  etapa,
                  fornecedor: c.fornecedor,
                  unidade: c.unidade,
                  quantidade: c.quantidade,
                  precoUnitarioUsd: c.preco,
                  custoUsd: c.quantidade * c.preco,
                  em: agora(),
                },
              ],
            }
          : dd,
      );
      // Piloto automático (POST-09): segue sozinho até a aprovação final, que continua obrigatória.
      await espera(80);
      const depois = ref.current?.posts.find((p) => p.id === postId);
      if (!depois || !depois.piloto || depois.descartado || depois.etapa !== etapa || etapa > 4) return;
      const prox = (etapa + 1) as Etapa;
      mudarPost(postId, (p) => ({
        ...p,
        etapa: prox,
        trabalhando: prox <= 4,
        atividade: prox === 5 ? [...p.atividade, { em: agora(), texto: "Pronto. Aguardando sua aprovação final." }] : p.atividade,
        historico: [...p.historico, { em: agora(), quem: "Piloto automático", acao: `Seguiu para a etapa ${prox} sem parar` }],
      }));
      if (prox <= 4) {
        await espera(80);
        await rodarAgente(postId, prox, undefined, true);
      }
    },
    [mudarPost],
  );

  const evento = (acao: string) => ({ em: agora(), quem, acao });

  const acoes = {
    entrar: (email: string) => setSessao({ email, orgId: ref.current?.orgs[0].id ?? "laura-growth" }),
    sair: () => setSessao(null),
    trocarOrg: (orgId: string) => setSessao((s) => (s ? { ...s, orgId } : s)),
    salvarKit: (kit: KitMarca) =>
      setDados((d) => (d ? { ...d, orgs: d.orgs.map((o) => (o.id === sessao?.orgId ? { ...o, kit } : o)) } : d)),
    salvarCadencia: (cadencia: Record<Rede, number>) =>
      setDados((d) => (d ? { ...d, orgs: d.orgs.map((o) => (o.id === sessao?.orgId ? { ...o, cadencia } : o)) } : d)),
    alternarConexao: (rede: Rede) =>
      setDados((d) =>
        d
          ? {
              ...d,
              orgs: d.orgs.map((o) =>
                o.id === sessao?.orgId
                  ? { ...o, conexoes: { ...o.conexoes, [rede]: o.conexoes[rede] === "ativa" ? "nao_conectada" : "ativa" } }
                  : o,
              ),
            }
          : d,
      ),
    novoPost: (origem: "sugestao" | "pedido", pauta: Pauta, data: string) => {
      const id = novoId("post");
      const piloto = !!ref.current?.orgs.find((o) => o.id === sessao!.orgId)?.pilotoAutomatico;
      const direto = piloto && origem === "pedido"; // a pauta veio da pessoa: o agente já parte para o texto
      const post: Post = {
        id,
        orgId: sessao!.orgId,
        origem,
        etapa: direto ? 2 : 1,
        data,
        descartado: false,
        piloto,
        pauta,
        versoes: [],
        historico: [evento(origem === "pedido" ? "Pediu um novo post" : "Pediu uma sugestão de pauta")],
        atividade: [],
        trabalhando: false,
      };
      setDados((d) => (d ? { ...d, posts: [post, ...d.posts] } : d));
      if (origem === "sugestao") setTimeout(() => rodarAgente(id, 1), 50);
      else if (direto) setTimeout(() => rodarAgente(id, 2), 50);
      return id;
    },
    /** Plano semanal em lote (CAL-04). Aprovar o plano cria os posts; não aprova nenhum para publicação. */
    criarPlano: (itens: ItemDoPlano[], orgId?: string) => {
      const alvo = orgId ?? sessao!.orgId;
      const piloto = !!ref.current?.orgs.find((o) => o.id === alvo)?.pilotoAutomatico;
      const novos: Post[] = itens.map((it) => ({
        id: novoId("post"),
        orgId: alvo,
        origem: "sugestao",
        etapa: piloto ? 2 : 1,
        data: it.data,
        descartado: false,
        piloto,
        pauta: it.pauta,
        versoes: [],
        historico: [{ em: agora(), quem: "Strategy Agent", acao: "Propôs a pauta no plano semanal" }, evento("Aprovou o plano semanal")],
        atividade: [],
        trabalhando: false,
      }));
      const custosPlano = novos.map((p) => {
        const c = consumoFicticio(1);
        return { id: novoId("custo"), orgId: alvo, postId: p.id, agente: "Strategy Agent" as const, etapa: 1 as Etapa, fornecedor: c.fornecedor, unidade: c.unidade, quantidade: c.quantidade, precoUnitarioUsd: c.preco, custoUsd: c.quantidade * c.preco, em: agora() };
      });
      setDados((d) => (d ? { ...d, posts: [...novos, ...d.posts], custos: [...d.custos, ...custosPlano] } : d));
      if (piloto)
        setTimeout(async () => {
          for (const p of novos) await rodarAgente(p.id, 2); // um post por vez
        }, 80);
      return novos.length;
    },
    /** Onboarding de nova marca (MKT-05): cria a organização e já entra nela. */
    criarOrg: (org: Organizacao, itens: ItemDoPlano[]) => {
      setDados((d) => (d ? { ...d, orgs: [...d.orgs, org] } : d));
      setSessao((s) => (s ? { ...s, orgId: org.id } : s));
      if (itens.length) setTimeout(() => acoes.criarPlano(itens, org.id), 150); // depois de a organização existir
    },
    salvarPiloto: (pilotoAutomatico: boolean) =>
      setDados((d) => (d ? { ...d, orgs: d.orgs.map((o) => (o.id === sessao?.orgId ? { ...o, pilotoAutomatico } : o)) } : d)),
    /** Pausa o piloto neste post: o agente termina o passo atual e espera a pessoa. */
    pausarPiloto: (postId: string) =>
      mudarPost(postId, (x) => ({ ...x, piloto: false, historico: [...x.historico, evento("Pausou o piloto automático neste post")] })),
    aprovarEtapa: (postId: string, retomarPiloto = false) => {
      const p = ref.current!.posts.find((x) => x.id === postId)!;
      const prox = (p.etapa + 1) as Etapa;
      mudarPost(postId, (x) => ({
        ...x,
        etapa: prox,
        piloto: retomarPiloto ? true : x.piloto,
        aprovadoPor: p.etapa === 5 ? quem : x.aprovadoPor,
        historico: [...x.historico, evento(p.etapa === 5 ? "Aprovou o post (aprovação final)" : `Aprovou a etapa ${p.etapa}`)],
      }));
      if (prox <= 4) setTimeout(() => rodarAgente(postId, prox), 80);
    },
    pedirAjuste: (postId: string, comentario: string) => {
      const p = ref.current!.posts.find((x) => x.id === postId)!;
      mudarPost(postId, (x) => ({ ...x, historico: [...x.historico, evento(`Pediu ajuste: "${comentario}"`)] }));
      rodarAgente(postId, p.etapa, comentario);
    },
    devolver: (postId: string, comentario: string) => {
      mudarPost(postId, (x) => ({ ...x, etapa: 2, historico: [...x.historico, evento(`Devolveu na aprovação final: "${comentario}"`)] }));
      setTimeout(() => rodarAgente(postId, 2, comentario), 50);
    },
    editarManual: (postId: string, campo: "pauta" | "texto" | "visual", valor: Pauta | Texto | Visual) =>
      mudarPost(postId, (x) => ({
        ...x,
        [campo]: valor,
        versoes: x[campo] ? [...x.versoes, { id: novoId("v"), etapa: (campo === "pauta" ? 1 : campo === "texto" ? 2 : 3) as Etapa, autor: "ia", em: agora(), conteudo: x[campo]! }] : x.versoes,
        historico: [...x.historico, evento(`Editou à mão (${campo})`)],
      })),
    restaurarVersao: (postId: string, versaoId: string) =>
      mudarPost(postId, (x) => {
        const v = x.versoes.find((y) => y.id === versaoId)!;
        const campo = v.etapa === 1 ? "pauta" : v.etapa === 2 ? "texto" : v.etapa === 3 ? "visual" : "revisao";
        return { ...x, [campo]: v.conteudo, historico: [...x.historico, evento(`Restaurou uma versão anterior (${campo})`)] };
      }),
    descartar: (postId: string) =>
      mudarPost(postId, (x) => ({ ...x, descartado: true, historico: [...x.historico, evento("Descartou o post")] })),
    reagendar: (postId: string, data: string) =>
      mudarPost(postId, (x) => ({ ...x, data, historico: [...x.historico, evento(`Reagendou para ${data}`)] })),
    marcarPublicado: (postId: string) =>
      mudarPost(postId, (x) => ({ ...x, publicacao: "publicado", historico: [...x.historico, evento("Marcou como publicado manualmente")] })),
    simularPublicacao: (postId: string, ok: boolean) =>
      mudarPost(postId, (x) => ({
        ...x,
        publicacao: ok ? "publicado" : "falhou",
        historico: [...x.historico, { em: agora(), quem: "Social Agent", acao: ok ? "Publicou e confirmou que foi ao ar (simulado)" : "Falha ao publicar (simulado)" }],
      })),
    tentarDeNovo: (postId: string) =>
      mudarPost(postId, (x) => ({ ...x, publicacao: undefined, historico: [...x.historico, evento("Pediu nova tentativa de publicação")] })),
    restaurarExemplo: () => setDados(dadosDeExemplo()),
  };

  const org = dados?.orgs.find((o) => o.id === sessao?.orgId) ?? null;
  return { pronto, dados, sessao, org, ...acoes };
}

type Loja = ReturnType<typeof useLojaInterna>;
const Ctx = createContext<Loja | null>(null);

export function LojaProvider({ children }: { children: React.ReactNode }) {
  const loja = useLojaInterna();
  return <Ctx.Provider value={loja}>{children}</Ctx.Provider>;
}
export function useLoja() {
  const c = useContext(Ctx);
  if (!c) throw new Error("useLoja fora do LojaProvider");
  return c;
}
export function usePostsDaOrg() {
  const { dados, org } = useLoja();
  return useMemo(() => (dados && org ? dados.posts.filter((p) => p.orgId === org.id) : []), [dados, org]);
}
