"use client";
import { useEffect, useState } from "react";
import { useLoja } from "@/mock/loja";
import type { KitMarca } from "@/mock/types";
import { PageHeader } from "@/components/ui";

const CAMPOS: { k: Exclude<keyof KitMarca, "cores">; rotulo: string; longo?: boolean }[] = [
  { k: "tomDeVoz", rotulo: "Tom de voz", longo: true },
  { k: "publico", rotulo: "Público", longo: true },
  { k: "temas", rotulo: "Temas (separados por vírgula)" },
  { k: "nuncaDizer", rotulo: "O que nunca dizer (separado por vírgula)" },
  { k: "fontes", rotulo: "Fontes" },
  { k: "exemplos", rotulo: "Exemplos de posts", longo: true },
];

export default function Kit() {
  const { org, salvarKit } = useLoja();
  const [kit, setKit] = useState<KitMarca | null>(null);
  const [salvo, setSalvo] = useState(false);
  useEffect(() => { if (org) { setKit(org.kit); setSalvo(false); } }, [org?.id]); // eslint-disable-line react-hooks/exhaustive-deps
  if (!org || !kit) return null;

  return (
    <>
      <PageHeader titulo="Kit de marca" descricao={`Base que os agentes usam para criar os posts da ${org.nome}. Nunca é usado em outra organização.`} />
      <form className="grid g2" onSubmit={(e) => { e.preventDefault(); salvarKit(kit); setSalvo(true); }}>
        <section className="card stack">
          {CAMPOS.map((c) => (
            <label key={c.k} className="campo">
              {c.rotulo}
              {c.longo ? (
                <textarea value={kit[c.k]} onChange={(e) => { setKit({ ...kit, [c.k]: e.target.value }); setSalvo(false); }} />
              ) : (
                <input value={kit[c.k]} onChange={(e) => { setKit({ ...kit, [c.k]: e.target.value }); setSalvo(false); }} />
              )}
            </label>
          ))}
          <div className="row">
            <button className="btn primario" type="submit">Salvar kit de marca</button>
            {salvo && <span role="status" style={{ color: "var(--laura-success)" }}>Kit salvo.</span>}
          </div>
        </section>
        <section className="card stack">
          <h2 style={{ margin: 0 }}>Cores</h2>
          {kit.cores.map((c, i) => (
            <label key={i} className="row">
              <input type="color" value={c} onChange={(e) => { setKit({ ...kit, cores: kit.cores.map((x, j) => (j === i ? e.target.value : x)) }); setSalvo(false); }} aria-label={`Cor ${i + 1}`} />
              <span className="num">{c}</span>
              <span className="small muted">{["Principal", "Secundária", "Fundo"][i]}</span>
            </label>
          ))}
          <h2 style={{ margin: "12px 0 0" }}>Logo</h2>
          <div className="row">
            <span className="logo-mark" style={{ width: 48, height: 48, background: kit.cores[0], fontSize: 20 }}>{org.nome[0]}</span>
            <span className="small muted">Envio de logo e arquivos: entra com o backend (o mock usa a inicial).</span>
          </div>
        </section>
      </form>
    </>
  );
}
