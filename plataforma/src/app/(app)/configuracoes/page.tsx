"use client";
// Visão geral das Configurações: cards agrupados, nenhum parâmetro aberto.
import Link from "next/link";
import { useLoja } from "@/mock/loja";
import { PageHeader } from "@/components/ui";
import { GRUPOS_CONFIG } from "@/components/Configuracoes";

export default function Configuracoes() {
  const { org } = useLoja();
  if (!org) return null;
  return (
    <>
      <PageHeader titulo="Configurações" descricao={`Escolha o que ajustar. Organização selecionada: ${org.nome}.`} />
      <div className="stack" style={{ gap: 28 }}>
        {GRUPOS_CONFIG.map((g) => (
          <section key={g.titulo} className="stack">
            <div>
              <h2 style={{ margin: 0 }}>{g.titulo}</h2>
              <p className="small muted" style={{ margin: "4px 0 0" }}>{g.descricao}</p>
            </div>
            <div className="config-cards">
              {g.secoes.map((s) => (
                <Link key={s.id} href={`/configuracoes/${s.id}`} className="card config-card">
                  <span className="config-icone"><s.icone size={18} aria-hidden /></span>
                  <span>
                    <b>{s.titulo}</b>
                    <span className="small muted" style={{ display: "block", marginTop: 2 }}>{s.descricao}</span>
                  </span>
                </Link>
              ))}
            </div>
          </section>
        ))}
      </div>
    </>
  );
}
