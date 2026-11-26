"use client";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { PedidoEntrar } from "@laura-growth/contratos";
import { FalhaApi, useEntrar } from "@/lib/api";

export default function Entrar() {
  const router = useRouter();
  const entrar = useEntrar();
  const { register, handleSubmit, setError, formState: { errors } } = useForm<PedidoEntrar>({ defaultValues: { email: "", senha: "" } });

  const enviar = handleSubmit((dados) => {
    const v = PedidoEntrar.safeParse(dados);
    if (!v.success) {
      for (const p of v.error.issues) setError(p.path[0] as keyof PedidoEntrar, { message: p.message });
      return;
    }
    entrar.mutate(v.data, {
      onSuccess: () => {
        const proximo =
          typeof window !== "undefined" ? new URLSearchParams(window.location.search).get("proximo") : null;
        const destino = proximo && proximo.startsWith("/convite/") && !proximo.startsWith("//") ? proximo : "/inicio";
        router.replace(destino);
      },
    });
  });

  const falha = entrar.error instanceof FalhaApi ? entrar.error.message : entrar.error ? "Não foi possível entrar. Tente de novo." : "";

  return (
    <div className="login">
      <form className="card stack" onSubmit={enviar} noValidate>
        <div className="logo" style={{ padding: 0 }}>
          <span className="logo-mark" aria-hidden>L</span> Laura Growth
        </div>
        <div>
          <h1>Entrar</h1>
          <p className="muted">Seu time de marketing com IA</p>
        </div>
        <label className="campo">
          E-mail
          <input type="email" autoComplete="username" placeholder="voce@exemplo.com" aria-invalid={!!errors.email} {...register("email")} />
          {errors.email && <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.email.message}</span>}
        </label>
        <label className="campo">
          Senha
          <input type="password" autoComplete="current-password" aria-invalid={!!errors.senha} {...register("senha")} />
          {errors.senha && <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.senha.message}</span>}
        </label>
        {falha && <p role="alert" style={{ color: "var(--laura-danger)", margin: 0 }}>{falha}</p>}
        <button className="btn primario" type="submit" disabled={entrar.isPending}>
          {entrar.isPending ? "Entrando…" : "Entrar na plataforma"}
        </button>
      </form>
    </div>
  );
}
