"use client";
// Onboarding assistido de nova marca (MKT-05): o agente rascunha o kit, a pessoa corrige,
// define a cadência e termina com o plano da primeira semana (CAL-04).
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ArrowLeft, ArrowRight, Plus, Sparkles, X } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { PASSOS_ONBOARDING, rascunharKit } from "@/mock/agente-falso";
import { REDES } from "@/mock/regras";
import type { KitMarca, Organizacao, Rede } from "@/mock/types";
import { PageHeader, TagIA } from "@/components/ui";
import { PlanoSemanal } from "@/components/PlanoSemanal";

const PASSOS = ["Sobre a marca", "Revisar o kit", "Cadência", "Primeira semana"];
const espera = (ms: number) => new Promise((r) => setTimeout(r, ms));

export default function Onboarding() {
  const { dados, criarOrg } = useLoja();
  const router = useRouter();
  const [passo, setPasso] = useState(0);
  const [nome, setNome] = useState("");
  const [tipo, setTipo] = useState<Organizacao["tipo"]>("clinica_estetica");
  const [site, setSite] = useState("");
  const [perfis, setPerfis] = useState("");
  const [aoVivo, setAoVivo] = useState<string[] | null>(null);
  const [kit, setKit] = useState<KitMarca | null>(null);
  const [cadencia, setCadencia] = useState<Record<Rede, number>>({ instagram: 3, facebook: 1, linkedin: 1 });
  const [piloto, setPiloto] = useState(true);
  if (!dados) return null;

  const nomeEmUso = dados.orgs.some((o) => o.nome.trim().toLowerCase() === nome.trim().toLowerCase());
  const rascunhar = async () => {
    setAoVivo([]);
    for (const p of PASSOS_ONBOARDING) {
      await espera(600);
      setAoVivo((x) => [...(x ?? []), p]);
    }
    await espera(400);
    setKit(rascunharKit(nome.trim(), tipo, site.trim(), perfis.trim()));
    setAoVivo(null);
    setPasso(1);
  };

  const org: Organizacao | null = kit && {
    id: `org-${nome.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-")}-${dados.orgs.length + 1}`,
    nome: nome.trim(),
    tipo,
    kit: { ...kit, pilares: kit.pilares.map((p) => p.trim()).filter(Boolean) },
    cadencia,
    pilotoAutomatico: piloto,
    conexoes: { instagram: "nao_conectada", facebook: "nao_conectada", linkedin: "nao_conectada" },
  };
  const texto = (k: "tomDeVoz" | "publico" | "nuncaDizer", rotulo: string, longo = false) => (
    <label className="campo">
      {rotulo}
      {longo ? <textarea value={kit![k]} onChange={(e) => setKit({ ...kit!, [k]: e.target.value })} /> : <input value={kit![k]} onChange={(e) => setKit({ ...kit!, [k]: e.target.value })} />}
    </label>
  );

  return (
    <>
      <PageHeader titulo="Nova marca" descricao="O agente rascunha o kit de marca; você corrige, define a cadência e aprova o plano da primeira semana." />
      <ol className="stepper" aria-label="Passos do onboarding" style={{ listStyle: "none", padding: 0, gridTemplateColumns: "repeat(4, 1fr)" }}>
        {PASSOS.map((p, i) => (
          <li key={p} className={`passo ${i < passo ? "feito" : ""} ${i === passo ? "atual" : ""}`} aria-current={i === passo ? "step" : undefined}>
            <b>{i + 1}. {p}</b>
            {i < passo ? "Concluído" : i === passo ? "Agora" : "A seguir"}
          </li>
        ))}
      </ol>

      {passo === 0 && (
        <section className="card stack" style={{ maxWidth: 640 }}>
          <label className="campo">Nome da marca<input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Ex.: Clínica Exemplo" autoFocus /></label>
          {nomeEmUso && <div className="alerta t-warning">Já existe uma organização com este nome.</div>}
          <label className="campo">
            Tipo de organização
            <select value={tipo} onChange={(e) => setTipo(e.target.value as Organizacao["tipo"])}>
              <option value="clinica_estetica">Clínica de estética</option>
              <option value="empresa">Empresa</option>
            </select>
          </label>
          <label className="campo">Site da marca<input value={site} onChange={(e) => setSite(e.target.value)} placeholder="Endereço do site" /></label>
          <label className="campo">Perfis nas redes<input value={perfis} onChange={(e) => setPerfis(e.target.value)} placeholder="Ex.: @marca no Instagram, página no LinkedIn" /></label>
          {aoVivo ? (
            <div className="card" style={{ background: "var(--laura-ai-soft)", borderColor: "transparent" }} aria-live="polite">
              <p className="row" style={{ margin: 0 }}><span className="pulsar" /> <b>Strategy Agent rascunhando o kit…</b></p>
              <ul className="atividade">{aoVivo.map((p) => <li key={p}>{p}</li>)}</ul>
            </div>
          ) : (
            <button className="btn primario" style={{ alignSelf: "flex-start" }} disabled={!nome.trim() || nomeEmUso} onClick={rascunhar}>
              <Sparkles size={16} aria-hidden /> Rascunhar kit com o agente
            </button>
          )}
          <p className="small muted">No protótipo o agente não lê o site de verdade: o rascunho é um exemplo. No produto, ele lerá o site e os perfis informados.</p>
        </section>
      )}

      {passo === 1 && kit && (
        <section className="card stack" style={{ maxWidth: 720 }}>
          <div className="row"><h2 style={{ margin: 0 }}>Rascunho do kit de {nome}</h2><TagIA texto="Rascunho de exemplo" /></div>
          <p className="muted" style={{ margin: 0 }}>Corrija o que não representa a marca. Você pode mudar tudo depois em Kit de marca.</p>
          {texto("tomDeVoz", "Tom de voz", true)}
          {texto("publico", "Público", true)}
          <div className="stack" style={{ gap: 6 }}>
            <b className="small">Pilares de conteúdo (de 3 a 5 assuntos fixos da marca)</b>
            {kit.pilares.map((p, i) => (
              <div key={i} className="row">
                <input style={{ flex: 1 }} value={p} aria-label={`Pilar ${i + 1}`} onChange={(e) => setKit({ ...kit, pilares: kit.pilares.map((x, j) => (j === i ? e.target.value : x)) })} />
                <button className="btn peq" aria-label={`Remover pilar ${i + 1}`} onClick={() => setKit({ ...kit, pilares: kit.pilares.filter((_, j) => j !== i) })}><X size={14} aria-hidden /></button>
              </div>
            ))}
            {kit.pilares.length < 5 && <button className="btn peq" style={{ alignSelf: "flex-start" }} onClick={() => setKit({ ...kit, pilares: [...kit.pilares, ""] })}><Plus size={14} aria-hidden /> Adicionar pilar</button>}
          </div>
          {texto("nuncaDizer", "O que nunca dizer (separado por vírgula)")}
          <label className="row">
            <input type="color" value={kit.cores[0]} onChange={(e) => setKit({ ...kit, cores: [e.target.value, kit.cores[1], kit.cores[2]] })} aria-label="Cor principal" />
            Cor principal da marca <span className="small muted">(logo e demais cores: em Kit de marca)</span>
          </label>
          <div className="row">
            <button className="btn" onClick={() => setPasso(0)}><ArrowLeft size={16} aria-hidden /> Voltar</button>
            <button className="btn primario" disabled={kit.pilares.filter((p) => p.trim()).length < 1} onClick={() => setPasso(2)}>Kit revisado, continuar <ArrowRight size={16} aria-hidden /></button>
          </div>
        </section>
      )}

      {passo === 2 && (
        <section className="card stack" style={{ maxWidth: 640 }}>
          <h2 style={{ margin: 0 }}>Quantos posts por semana?</h2>
          <p className="muted" style={{ margin: 0 }}>Valores de exemplo; ajuste para a realidade da marca. Você pode mudar depois em Configurações.</p>
          {REDES.map((r) => (
            <label key={r.id} className="campo" style={{ maxWidth: 240 }}>
              {r.nome} (posts por semana)
              <input type="number" min={0} max={21} value={cadencia[r.id]} onChange={(e) => setCadencia({ ...cadencia, [r.id]: Math.max(0, Number(e.target.value)) })} />
            </label>
          ))}
          <label className="row">
            <input type="checkbox" checked={piloto} onChange={(e) => setPiloto(e.target.checked)} />
            Piloto automático: o agente segue sozinho até a aprovação final
          </label>
          <div className="row">
            <button className="btn" onClick={() => setPasso(1)}><ArrowLeft size={16} aria-hidden /> Voltar</button>
            <button className="btn primario" onClick={() => setPasso(3)}>Continuar <ArrowRight size={16} aria-hidden /></button>
          </div>
        </section>
      )}

      {passo === 3 && org && (
        <section className="card stack" style={{ maxWidth: 720 }}>
          <h2 style={{ margin: 0 }}>Plano da primeira semana de {org.nome}</h2>
          <p className="muted" style={{ margin: 0 }}>Para fechar o onboarding, o agente propõe os primeiros posts. No protótipo o plano é semanal.</p>
          <PlanoSemanal org={org} posts={[]} rotuloAprovar="Concluir e criar" aoAprovar={(itens) => { criarOrg(org, itens); router.push("/calendario"); }} />
          <div className="row">
            <button className="btn" onClick={() => setPasso(2)}><ArrowLeft size={16} aria-hidden /> Voltar</button>
            <button className="btn" onClick={() => { criarOrg(org, []); router.push("/kit"); }}>Concluir sem plano</button>
          </div>
        </section>
      )}
    </>
  );
}
