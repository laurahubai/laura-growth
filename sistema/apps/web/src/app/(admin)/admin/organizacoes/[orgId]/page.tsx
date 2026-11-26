"use client";
import { useParams } from "next/navigation";
import { GerenciarPessoas } from "@/components/GerenciarPessoas";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import { useOrganizacoesAdmin } from "@/lib/api";

export default function PaginaPessoasOrganizacaoAdmin() {
  const params = useParams<{ orgId: string }>();
  const orgId = typeof params?.orgId === "string" ? params.orgId : "";
  const organizacoes = useOrganizacoesAdmin();

  const bloqueado = useRedirecionarSemPermissao(organizacoes.error);
  if (bloqueado) return <div className="vazio" role="alert">{bloqueado}</div>;

  const org = organizacoes.data?.organizacoes.find((o) => o.id === orgId);

  return (
    <GerenciarPessoas
      orgId={orgId}
      orgNome={org?.nome}
      baseApi={`/api/admin/organizacoes/${encodeURIComponent(orgId)}`}
      erroSemPermissaoComoSemAcesso={false}
      linkVoltar={{ href: "/admin/organizacoes", rotulo: "Voltar para organizações" }}
    />
  );
}
