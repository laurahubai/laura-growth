"use client";
import { useEffect, useState } from "react";
import { useLoja } from "@/mock/loja";
import { REDES } from "@/mock/regras";
import type { Rede } from "@/mock/types";
import { PageHeader } from "@/components/ui";

export default function Configuracoes() {
  const { org, salvarCadencia, salvarPiloto, restaurarExemplo } = useLoja();
  const [cad, setCad] = useState<Record<Rede, number> | null>(null);
  const [ok, setOk] = useState(false);
  useEffect(() => { if (org) { setCad(org.cadencia); setOk(false); } }, [org?.id]); // eslint-disable-line react-hooks/exhaustive-deps
  if (!org || !cad) return null;

  return (
    <>
      <PageHeader titulo="Configurações" descricao={org.nome} />
      <div className="grid g2">
        <section className="card stack">
          <h2 style={{ margin: 0 }}>Cadência desejada</h2>
          <p className="muted">Quantos posts por semana em cada rede. Valores de exemplo; a definir por organização.</p>
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
        <section className="card stack">
          <h2 style={{ margin: 0 }}>Piloto automático</h2>
          <p className="muted">Ligado: o agente segue sozinho por pauta, texto, visual e revisão e só para na aprovação final. Desligado: ele espera você em cada etapa. A aprovação final é sempre obrigatória.</p>
          <label className="row">
            <input type="checkbox" checked={org.pilotoAutomatico} onChange={(e) => salvarPiloto(e.target.checked)} />
            Piloto automático ligado para a {org.nome}
          </label>
          <p className="small muted">Vale para os posts criados a partir de agora. Em qualquer post você pode pausar para revisar.</p>
        </section>
        <section className="card stack">
          <h2 style={{ margin: 0 }}>Dados do protótipo</h2>
          <p className="muted">Os dados ficam só neste navegador. Restaurar apaga o que você criou e volta aos dados de exemplo das duas organizações.</p>
          <button className="btn perigo" onClick={() => { if (confirm("Restaurar os dados de exemplo? Tudo o que você criou no protótipo será apagado deste navegador.")) restaurarExemplo(); }}>
            Restaurar dados de exemplo
          </button>
        </section>
      </div>
    </>
  );
}
