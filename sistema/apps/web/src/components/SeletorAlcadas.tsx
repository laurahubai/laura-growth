"use client";
import {
  ALCADAS,
  CONJUNTOS_ALCADAS,
  rotuloAlcada,
  type Alcada,
  type ConjuntoAlcadas,
} from "@laura-growth/contratos";
import { useState } from "react";

export function SeletorAlcadas({
  alcadas,
  aoMudar,
  conjuntos = CONJUNTOS_ALCADAS,
  rotuloConjunto = "Conjunto pronto",
  desabilitado = false,
}: {
  alcadas: Alcada[];
  aoMudar: (novas: Alcada[]) => void;
  conjuntos?: readonly ConjuntoAlcadas[];
  rotuloConjunto?: string;
  desabilitado?: boolean;
}) {
  const [conjuntoVisual, setConjuntoVisual] = useState("");
  const selecionadas = new Set(alcadas);

  const aplicarConjunto = (codigo: string) => {
    setConjuntoVisual(codigo);
    const encontrado = conjuntos.find((c) => c.codigo === codigo);
    if (encontrado) {
      aoMudar([...encontrado.alcadas]);
    }
  };

  const alternarAlcada = (codigo: Alcada) => {
    setConjuntoVisual("");
    const proximo = new Set(selecionadas);
    if (proximo.has(codigo)) proximo.delete(codigo);
    else proximo.add(codigo);
    aoMudar(ALCADAS.filter((a) => proximo.has(a)));
  };

  return (
    <div className="stack">
      <label className="campo">
        {rotuloConjunto}
        <select
          value={conjuntoVisual}
          disabled={desabilitado}
          onChange={(e) => aplicarConjunto(e.target.value)}
        >
          <option value="">Escolha um conjunto para pré-preencher…</option>
          {conjuntos.map((c) => (
            <option key={c.codigo} value={c.codigo}>
              {c.nome}
            </option>
          ))}
        </select>
        <span className="small muted">
          Aplicar um conjunto apenas pré-preenche as caixas abaixo; você pode ajustar antes de salvar.
        </span>
      </label>

      <fieldset className="stack" style={{ border: "1px solid var(--laura-border)", borderRadius: "var(--radius-sm)", padding: 12, margin: 0, gap: 8 }}>
        <legend className="small muted" style={{ padding: "0 6px" }}>Alçadas</legend>
        {ALCADAS.map((a) => (
          <label key={a} className="row" style={{ fontWeight: 400, cursor: desabilitado ? "not-allowed" : "pointer" }}>
            <input
              type="checkbox"
              checked={selecionadas.has(a)}
              disabled={desabilitado}
              onChange={() => alternarAlcada(a)}
            />
            <span>{rotuloAlcada[a]}</span>
          </label>
        ))}
      </fieldset>
    </div>
  );
}
