"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { AlertTriangle, ArrowLeft, LogOut, ScrollText, Server, ShieldCheck, Workflow } from "lucide-react";
import { erroSemPermissao, FalhaApi, useAvisosAdmin, useSair, useSessao } from "@/lib/api";
import { formatarUsd } from "@/lib/formato";

const MENU_ADMIN = [
  { href: "/admin/fornecedores", nome: "Fornecedores", icone: Server },
  { href: "/admin/atividades", nome: "Atividades", icone: Workflow },
  { href: "/admin/registros", nome: "Registros", icone: ScrollText },
];

export function useRedirecionarSemPermissao(erro: unknown): string | null {
  const router = useRouter();
  const expirada = erro instanceof FalhaApi && erro.status === 401;
  const bloqueado = erroSemPermissao(erro);
  useEffect(() => {
    if (expirada) router.replace("/entrar");
    else if (bloqueado) router.replace("/inicio");
  }, [expirada, bloqueado, router]);
  if (expirada) return "Sua sessão expirou. Redirecionando…";
  if (bloqueado) return "Você não tem acesso à Administração.";
  return null;
}

export function ShellAdmin({ children }: { children: React.ReactNode }) {
  const sessao = useSessao();
  const sair = useSair();
  const rota = usePathname();
  const router = useRouter();

  const ehAdmin = Boolean(sessao.data?.usuario.adminPlataforma);
  const avisos = useAvisosAdmin(ehAdmin);
  const semSessao =
    (sessao.error instanceof FalhaApi && sessao.error.status === 401) ||
    (avisos.error instanceof FalhaApi && avisos.error.status === 401);
  const semPermissaoApi = erroSemPermissao(avisos.error);
  const semPermissao = Boolean(sessao.data && !ehAdmin) || semPermissaoApi;

  useEffect(() => {
    if (semSessao) router.replace("/entrar");
  }, [semSessao, router]);

  useEffect(() => {
    if (semPermissao) router.replace("/inicio");
  }, [semPermissao, router]);

  if (sessao.isPending || semSessao) return <div className="vazio">Carregando…</div>;
  if (sessao.isError) return <div className="vazio" role="alert">Não foi possível carregar. Recarregue a página.</div>;
  if (semPermissao) return <div className="vazio" role="alert">Você não tem acesso à Administração.</div>;

  const s = sessao.data;

  return (
    <div className="shell">
      <nav className="sidebar" aria-label="Menu da Administração">
        <div className="logo">
          <span className="logo-mark" style={{ background: "var(--laura-primary)" }} aria-hidden>L</span>
          <span>Administração</span>
        </div>
        <div className="nav-grupo">IA e Custos</div>
        {MENU_ADMIN.map(({ href, nome, icone: I }) => (
          <Link key={href} href={href} className={`nav-item ${rota.startsWith(href) ? "ativo" : ""}`}>
            <I size={18} aria-hidden /> {nome}
          </Link>
        ))}
        <div className="nav-grupo" style={{ marginTop: "auto" }}>Operação</div>
        <Link href="/inicio" className="nav-item">
          <ArrowLeft size={18} aria-hidden /> Voltar para a Operação
        </Link>
      </nav>
      <div className="main">
        <header className="topbar topbar-admin">
          <span className="admin-rotulo">
            <ShieldCheck size={18} aria-hidden /> Administração da plataforma
          </span>
          <span className="espaco" />
          <span className="small muted">{s.usuario.email}</span>
          <button
            className="btn peq"
            title="Sair"
            aria-label="Sair"
            onClick={() => sair.mutate(undefined, { onSettled: () => router.replace("/entrar") })}
          >
            <LogOut size={16} aria-hidden />
          </button>
        </header>
        <main className="conteudo">
          {avisos.isError && !semPermissaoApi && (
            <div className="admin-faixa-avisos">
              <div className="alerta t-danger" role="alert">{avisos.error.message}</div>
            </div>
          )}
          {avisos.data && avisos.data.avisos.length > 0 && (
            <div className="admin-faixa-avisos" data-testid="faixa-avisos">
              {avisos.data.avisos.map((a) => {
                const pct = a.limiteMensalUsd > 0 ? Math.round((a.gastoMesUsd / a.limiteMensalUsd) * 100) : 0;
                const prefixo = a.alvo === "fornecedor" ? "Fornecedor" : "Modelo";
                return (
                  <div
                    key={`${a.alvo}-${a.id}`}
                    role="alert"
                    className={`alerta ${a.situacao === "limite" ? "t-danger" : "t-warning"}`}
                  >
                    <AlertTriangle size={18} aria-hidden style={{ flex: "none", marginTop: 1 }} />
                    <span>
                      {prefixo} <b>{a.nome}</b> usou{" "}
                      <span className="num">{formatarUsd(a.gastoMesUsd)}</span> de{" "}
                      <span className="num">{formatarUsd(a.limiteMensalUsd)}</span> ({pct}%)
                    </span>
                  </div>
                );
              })}
            </div>
          )}
          {children}
        </main>
      </div>
    </div>
  );
}
