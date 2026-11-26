"use client";
import { GerenciarPessoas } from "@/components/GerenciarPessoas";
import { useSessao } from "@/lib/api";

export default function PaginaPessoas() {
  const { data: sessao } = useSessao();
  if (!sessao) return null;

  return (
    <GerenciarPessoas
      orgId={sessao.organizacaoAtiva.id}
      orgNome={sessao.organizacaoAtiva.nome}
    />
  );
}
