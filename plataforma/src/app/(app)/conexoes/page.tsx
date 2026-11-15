"use client";
import { useLoja } from "@/mock/loja";
import { REDES } from "@/mock/regras";
import { PageHeader } from "@/components/ui";

const ROTULO = { nao_conectada: ["Não conectada", "t-neutral"], ativa: ["Ativa (simulada)", "t-success"], expirada: ["Expirada", "t-warning"] } as const;

export default function Conexoes() {
  const { org, alternarConexao } = useLoja();
  if (!org) return null;
  return (
    <>
      <PageHeader titulo="Conexões" descricao={`Contas de redes sociais da ${org.nome}. Pertencem à organização.`} />
      <div className="alerta t-info" style={{ marginBottom: 16 }}>
        A publicação automática depende da autorização de cada rede (a verificar). Enquanto isso, posts aprovados ficam prontos para publicação manual. Aqui você pode <b>simular</b> uma conexão para homologar o fluxo de agendamento.
      </div>
      <div className="grid g3">
        {REDES.map((r) => {
          const [rot, tom] = ROTULO[org.conexoes[r.id]];
          return (
            <section key={r.id} className="card stack">
              <div className="row" style={{ justifyContent: "space-between" }}>
                <h2 style={{ margin: 0 }}>{r.nome}</h2>
                <span className={`badge ${tom}`}>{rot}</span>
              </div>
              <button className="btn" onClick={() => alternarConexao(r.id)}>
                {org.conexoes[r.id] === "ativa" ? "Desconectar (simulado)" : "Simular conexão"}
              </button>
            </section>
          );
        })}
      </div>
    </>
  );
}
