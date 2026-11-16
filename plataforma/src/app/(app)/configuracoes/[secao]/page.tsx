"use client";
// Uma configuração por vez, com o menu lateral para ir às outras.
import Link from "next/link";
import { use } from "react";
import { ChevronLeft } from "lucide-react";
import { useLoja } from "@/mock/loja";
import { acharSecao, MenuConfig } from "@/components/Configuracoes";

export default function SecaoConfiguracao({ params }: { params: Promise<{ secao: string }> }) {
  const { secao } = use(params);
  const { org } = useLoja();
  const atual = acharSecao(secao);
  if (!org) return null;
  return (
    <>
      <Link href="/configuracoes" className="small row" style={{ gap: 4, marginBottom: 8 }}><ChevronLeft size={14} aria-hidden /> Configurações</Link>
      <div className="config-layout">
        <MenuConfig ativa={secao} />
        <div className="stack">
          {atual ? <atual.Componente /> : <section className="card vazio">Configuração não encontrada. <Link href="/configuracoes">Ver todas</Link></section>}
        </div>
      </div>
    </>
  );
}
