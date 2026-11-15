"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { Bell, Calendar, CheckCheck, Coins, FileText, Home, LogOut, Palette, Plug, Plus, Settings } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { estadoDoPost } from "@/mock/regras";

const MENU = [
  { grupo: "Trabalho", itens: [
    { href: "/inicio", nome: "Início", icone: Home },
    { href: "/aprovacoes", nome: "Aprovações", icone: CheckCheck },
    { href: "/posts", nome: "Posts", icone: FileText },
    { href: "/calendario", nome: "Calendário", icone: Calendar },
  ] },
  { grupo: "Organização", itens: [
    { href: "/kit", nome: "Kit de marca", icone: Palette },
    { href: "/conexoes", nome: "Conexões", icone: Plug },
    { href: "/configuracoes", nome: "Configurações", icone: Settings },
  ] },
  { grupo: "Operação Laura", itens: [{ href: "/custos", nome: "Custos", icone: Coins }] },
];

export function Shell({ children }: { children: React.ReactNode }) {
  const { pronto, sessao, dados, org, trocarOrg, sair } = useLoja();
  const rota = usePathname();
  const router = useRouter();

  useEffect(() => {
    if (pronto && !sessao) router.replace("/entrar");
  }, [pronto, sessao, router]);

  if (!pronto || !sessao || !dados || !org) return <div className="vazio">Carregando…</div>;

  const pendentes = dados.posts.filter((p) => p.orgId === org.id && ["aguardando_aprovacao", "atrasado"].includes(estadoDoPost(p, org))).length;
  const naFila = dados.posts.filter((p) => p.orgId === org.id && p.etapa === 5 && !p.descartado).length;

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
                {href === "/aprovacoes" && naFila > 0 && <span className="badge t-ai num" style={{ marginLeft: "auto" }}>{naFila}</span>}
              </Link>
            ))}
          </div>
        ))}
        <div className="sidebar-rodape">Protótipo com dados de exemplo. Nada é publicado de verdade.</div>
      </nav>
      <div className="main">
        <header className="topbar">
          <label className="row" style={{ gap: 6 }}>
            <span className="small muted">Organização</span>
            <select value={org.id} onChange={(e) => trocarOrg(e.target.value)} aria-label="Trocar organização">
              {dados.orgs.map((o) => (
                <option key={o.id} value={o.id}>{o.nome}</option>
              ))}
            </select>
          </label>
          <Link href="/onboarding" className="btn peq"><Plus size={14} aria-hidden /> Nova marca</Link>
          <span className="espaco" />
          <span className="mock-aviso">Dados de exemplo</span>
          <Link href={naFila > 0 ? "/aprovacoes" : "/inicio"} className="btn peq" title="Itens que precisam de você" aria-label={`${pendentes} itens aguardando você`}>
            <Bell size={16} aria-hidden /> <span className="num">{pendentes}</span>
          </Link>
          <span className="small muted">{sessao.email}</span>
          <button className="btn peq" onClick={sair} title="Sair" aria-label="Sair">
            <LogOut size={16} aria-hidden />
          </button>
        </header>
        <main className="conteudo">{children}</main>
      </div>
    </div>
  );
}
