"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { Home, LogOut } from "lucide-react";
import { FalhaApi, useSair, useSessao, useTrocarOrganizacao } from "@/lib/api";

// Itens entram no menu conforme as etapas do roteiro ficam prontas.
const MENU = [{ grupo: "Trabalho", itens: [{ href: "/inicio", nome: "Início", icone: Home }] }];

export function Shell({ children }: { children: React.ReactNode }) {
  const sessao = useSessao();
  const trocar = useTrocarOrganizacao();
  const sair = useSair();
  const rota = usePathname();
  const router = useRouter();

  const semSessao = sessao.error instanceof FalhaApi && sessao.error.status === 401;
  useEffect(() => {
    if (semSessao) router.replace("/entrar");
  }, [semSessao, router]);

  if (sessao.isPending || semSessao) return <div className="vazio">Carregando…</div>;
  if (sessao.isError) return <div className="vazio" role="alert">Não foi possível carregar. Recarregue a página.</div>;
  const s = sessao.data;

  return (
    <div className="shell">
      <nav className="sidebar" aria-label="Menu principal">
        <div className="logo">
          <span className="logo-mark" aria-hidden>L</span> Laura Growth
        </div>
        {MENU.map((g) => (
          <div key={g.grupo} style={{ display: "contents" }}>
            <div className="nav-grupo">{g.grupo}</div>
            {g.itens.map(({ href, nome, icone: I }) => (
              <Link key={href} href={href} className={`nav-item ${rota.startsWith(href) ? "ativo" : ""}`}>
                <I size={18} aria-hidden /> {nome}
              </Link>
            ))}
          </div>
        ))}
      </nav>
      <div className="main">
        <header className="topbar">
          <label className="row" style={{ gap: 6 }}>
            <span className="small muted">Organização</span>
            <select
              value={s.organizacaoAtiva.id}
              onChange={(e) => trocar.mutate(e.target.value)}
              disabled={trocar.isPending || s.organizacoes.length < 2}
              aria-label="Trocar organização"
            >
              {s.organizacoes.map((o) => (
                <option key={o.id} value={o.id}>{o.nome}</option>
              ))}
            </select>
          </label>
          {trocar.isError && <span className="small" role="alert" style={{ color: "var(--laura-danger)" }}>{trocar.error.message}</span>}
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
        <main className="conteudo" key={s.organizacaoAtiva.id}>{children}</main>
      </div>
    </div>
  );
}
