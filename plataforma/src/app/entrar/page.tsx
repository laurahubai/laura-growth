"use client";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useLoja } from "@/mock/loja";

export default function Entrar() {
  const { entrar } = useLoja();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [erro, setErro] = useState("");

  function enviar(e: React.FormEvent) {
    e.preventDefault();
    if (!email.includes("@") || senha.length < 4) {
      setErro("Confira o e-mail e use uma senha com pelo menos 4 caracteres.");
      return;
    }
    entrar(email);
    router.push("/inicio");
  }

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
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" placeholder="voce@exemplo.com" />
        </label>
        <label className="campo">
          Senha
          <input type="password" value={senha} onChange={(e) => setSenha(e.target.value)} autoComplete="off" />
        </label>
        {erro && <p role="alert" style={{ color: "var(--laura-danger)" }}>{erro}</p>}
        <button className="btn primario" type="submit">Entrar na plataforma</button>
        <p className="small muted">Protótipo: qualquer e-mail e senha funcionam. Não use sua senha real.</p>
      </form>
    </div>
  );
}
