"use client";
import type { RegistroAcesso } from "@laura-growth/contratos";
import { descreverDetalheAcesso, formatarDataHora, rotuloAcaoAcesso } from "@/lib/formato";

function formatarPessoa(nome: string | null, email: string | null): string {
  if (nome && email) return `${nome} (${email})`;
  return nome || email || "—";
}

export function TabelaRegistroAcessos({
  registros,
  vazio = "Nenhuma alteração registrada ainda.",
}: {
  registros: RegistroAcesso[];
  vazio?: string;
}) {
  if (registros.length === 0) {
    return <p className="muted" style={{ margin: 0 }}>{vazio}</p>;
  }

  return (
    <div className="rolagem">
      <table>
        <thead>
          <tr>
            <th>Quando</th>
            <th>Quem fez</th>
            <th>Pessoa</th>
            <th>Ação</th>
            <th>Detalhe</th>
          </tr>
        </thead>
        <tbody>
          {registros.map((r) => (
            <tr key={r.id}>
              <td className="num small">{formatarDataHora(r.em)}</td>
              <td className="small">{formatarPessoa(r.atorNome, r.atorEmail)}</td>
              <td className="small">{formatarPessoa(r.alvoNome, r.alvoEmail)}</td>
              <td>{rotuloAcaoAcesso(r.acao)}</td>
              <td className="small muted">{descreverDetalheAcesso(r)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
