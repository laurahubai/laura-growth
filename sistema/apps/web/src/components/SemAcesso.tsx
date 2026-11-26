"use client";
import Link from "next/link";
import { ShieldAlert } from "lucide-react";

export function SemAcesso({
  titulo = "Sem acesso",
  mensagem = "Você não tem permissão para acessar esta área.",
  voltarHref = "/inicio",
  voltarRotulo = "Voltar para o início",
}: {
  titulo?: string;
  mensagem?: string;
  voltarHref?: string;
  voltarRotulo?: string;
}) {
  return (
    <section className="card stack" role="alert" data-testid="estado-sem-acesso" style={{ maxWidth: 520 }}>
      <div className="row" style={{ gap: 10 }}>
        <ShieldAlert size={22} aria-hidden style={{ color: "var(--laura-danger)", flex: "none" }} />
        <h1 style={{ margin: 0 }}>{titulo}</h1>
      </div>
      <p style={{ margin: 0 }}>{mensagem}</p>
      <div>
        <Link href={voltarHref} className="btn">
          {voltarRotulo}
        </Link>
      </div>
    </section>
  );
}
