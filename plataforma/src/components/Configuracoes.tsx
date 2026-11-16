"use client";
// Seções da tela de Configurações. Para criar uma configuração nova: escreva o componente
// e acrescente um item em GRUPOS_CONFIG. Ela aparece sozinha nos cards e no menu lateral.
import Link from "next/link";
import { useEffect, useState } from "react";
import { CalendarClock, Cpu, Database, KeyRound, Workflow, type LucideIcon } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { REDES } from "@/mock/regras";
import type { Rede } from "@/mock/types";
import { ApiPorEtapa, ApisCadastradas } from "@/components/ApisDeIA";

export type SecaoConfig = { id: string; titulo: string; descricao: string; icone: LucideIcon; Componente: () => React.ReactNode };
export type GrupoConfig = { titulo: string; descricao: string; secoes: SecaoConfig[] };

function Cadencia() {
  const { org, salvarCadencia } = useLoja();
  const [cad, setCad] = useState<Record<Rede, number> | null>(null);
  const [ok, setOk] = useState(false);
  useEffect(() => { if (org) { setCad(org.cadencia); setOk(false); } }, [org?.id]); // eslint-disable-line react-hooks/exhaustive-deps
  if (!org || !cad) return null;
  return (
    <section className="card stack">
      <h2 style={{ margin: 0 }}>Cadência desejada</h2>
      <p className="muted">Quantos posts por semana em cada rede, para a {org.nome}. Valores de exemplo; a definir por organização.</p>
      {REDES.map((r) => (
        <label key={r.id} className="campo" style={{ maxWidth: 240 }}>
          {r.nome} (posts por semana)
          <input type="number" min={0} max={21} value={cad[r.id]} onChange={(e) => { setCad({ ...cad, [r.id]: Number(e.target.value) }); setOk(false); }} />
        </label>
      ))}
      <div className="row">
        <button className="btn primario" onClick={() => { salvarCadencia(cad); setOk(true); }}>Salvar cadência</button>
        {ok && <span role="status" style={{ color: "var(--laura-success)" }}>Cadência salva.</span>}
      </div>
    </section>
  );
}

function Piloto() {
  const { org, salvarPiloto } = useLoja();
  if (!org) return null;
  return (
    <section className="card stack">
      <h2 style={{ margin: 0 }}>Piloto automático</h2>
      <p className="muted">Ligado: o agente segue sozinho por pauta, texto, visual e revisão e só para na aprovação final. Desligado: ele espera você em cada etapa. A aprovação final é sempre obrigatória.</p>
      <label className="row">
        <input type="checkbox" checked={org.pilotoAutomatico} onChange={(e) => salvarPiloto(e.target.checked)} />
        Piloto automático ligado para a {org.nome}
      </label>
      <p className="small muted">Vale para os posts criados a partir de agora. Em qualquer post você pode pausar para revisar.</p>
    </section>
  );
}

function DadosPrototipo() {
  const { restaurarExemplo } = useLoja();
  return (
    <section className="card stack">
      <h2 style={{ margin: 0 }}>Dados do protótipo</h2>
      <p className="muted">Os dados ficam só neste navegador. Restaurar apaga o que você criou e volta aos dados de exemplo das duas organizações.</p>
      <button className="btn perigo" style={{ alignSelf: "flex-start" }} onClick={() => { if (confirm("Restaurar os dados de exemplo? Tudo o que você criou no protótipo será apagado deste navegador.")) restaurarExemplo(); }}>
        Restaurar dados de exemplo
      </button>
    </section>
  );
}

export const GRUPOS_CONFIG: GrupoConfig[] = [
  {
    titulo: "Agente e publicação",
    descricao: "Como o agente trabalha para a organização selecionada.",
    secoes: [
      { id: "cadencia", titulo: "Cadência", descricao: "Quantos posts por semana em cada rede.", icone: CalendarClock, Componente: Cadencia },
      { id: "piloto", titulo: "Piloto automático", descricao: "Se o agente avança sozinho até a aprovação final.", icone: Workflow, Componente: Piloto },
    ],
  },
  {
    titulo: "Inteligência artificial",
    descricao: "Vale para todas as organizações.",
    secoes: [
      { id: "apis", titulo: "APIs de IA", descricao: "Cadastro das conexões com os modelos de IA.", icone: KeyRound, Componente: ApisCadastradas },
      { id: "api-por-etapa", titulo: "API por etapa", descricao: "Qual API e qual reserva cada etapa usa.", icone: Cpu, Componente: ApiPorEtapa },
    ],
  },
  {
    titulo: "Protótipo",
    descricao: "Só existe nesta versão de teste.",
    secoes: [
      { id: "prototipo", titulo: "Dados do protótipo", descricao: "Voltar aos dados de exemplo.", icone: Database, Componente: DadosPrototipo },
    ],
  },
];

export const acharSecao = (id: string) => GRUPOS_CONFIG.flatMap((g) => g.secoes).find((s) => s.id === id);

export function MenuConfig({ ativa }: { ativa: string }) {
  return (
    <nav className="config-menu" aria-label="Configurações">
      <Link href="/configuracoes" className="nav-item">Visão geral</Link>
      {GRUPOS_CONFIG.map((g) => (
        <div key={g.titulo}>
          <div className="nav-grupo">{g.titulo}</div>
          {g.secoes.map((s) => (
            <Link key={s.id} href={`/configuracoes/${s.id}`} className={`nav-item ${s.id === ativa ? "ativo" : ""}`} aria-current={s.id === ativa ? "page" : undefined}>
              <s.icone size={16} aria-hidden /> {s.titulo}
            </Link>
          ))}
        </div>
      ))}
    </nav>
  );
}
