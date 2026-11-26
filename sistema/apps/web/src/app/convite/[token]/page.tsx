"use client";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { PedidoEntrar } from "@laura-growth/contratos";
import {
  FalhaApi,
  useAceitarConvite,
  useConvitePublico,
  useEntrar,
  useSair,
  useSessao,
} from "@/lib/api";

function mensagemErroAceite(erro: unknown): string {
  if (!(erro instanceof FalhaApi)) {
    return "Não foi possível aceitar o convite. Tente de novo.";
  }
  if (erro.message) return erro.message;
  if (erro.codigo === "login_necessario") return "Entre com a conta convidada para aceitar o convite.";
  if (erro.codigo === "email_diferente") return "Você está conectado com outro e-mail. Saia e entre com o e-mail convidado.";
  if (erro.codigo === "convite_invalido") return "Este convite não é mais válido (expirou, já foi usado ou foi cancelado).";
  return "Não foi possível aceitar o convite.";
}

export default function PaginaConvite() {
  const params = useParams<{ token: string }>();
  const token = typeof params?.token === "string" ? params.token : "";
  const router = useRouter();
  const qc = useQueryClient();

  const convite = useConvitePublico(token);
  const sessao = useSessao();
  const aceitar = useAceitarConvite(token);
  const entrar = useEntrar();
  const sair = useSair();

  const formCadastro = useForm<{ nome: string; senha: string }>({
    defaultValues: { nome: "", senha: "" },
  });
  const formLogin = useForm<PedidoEntrar>({
    defaultValues: { email: "", senha: "" },
  });

  if (convite.isPending) {
    return (
      <div className="login">
        <div className="card">
          <p className="muted" style={{ margin: 0 }}>Carregando convite…</p>
        </div>
      </div>
    );
  }

  if (convite.isError) {
    const invalido =
      convite.error instanceof FalhaApi &&
      (convite.error.status === 404 || convite.error.codigo === "convite_invalido");
    return (
      <div className="login">
        <div className="card stack" role="alert">
          <div className="logo" style={{ padding: 0 }}>
            <span className="logo-mark" aria-hidden>L</span> Laura Growth
          </div>
          <h1>{invalido ? "Convite inválido ou expirado" : "Não foi possível abrir o convite"}</h1>
          <p className="muted" style={{ margin: 0 }}>
            {convite.error instanceof FalhaApi && convite.error.message
              ? convite.error.message
              : "Este convite não existe, já foi utilizado, expirou ou foi cancelado."}
          </p>
          <div>
            <Link href="/entrar" className="btn">
              Ir para o login
            </Link>
          </div>
        </div>
      </div>
    );
  }

  const dados = convite.data;

  const aceitarContaNova = formCadastro.handleSubmit((campos) => {
    const nome = campos.nome.trim();
    if (!nome) {
      formCadastro.setError("nome", { message: "Informe seu nome." });
      return;
    }
    if (campos.senha.length < 10) {
      formCadastro.setError("senha", { message: "A senha precisa ter pelo menos 10 caracteres." });
      return;
    }
    if (campos.senha.length > 72) {
      formCadastro.setError("senha", { message: "A senha pode ter no máximo 72 caracteres." });
      return;
    }
    aceitar.mutate(
      { nome, senha: campos.senha },
      { onSuccess: () => router.replace("/inicio") },
    );
  });

  const entrarInline = formLogin.handleSubmit((campos) => {
    const v = PedidoEntrar.safeParse(campos);
    if (!v.success) {
      for (const issue of v.error.issues) {
        formLogin.setError(issue.path[0] as keyof PedidoEntrar, { message: issue.message });
      }
      return;
    }
    aceitar.reset();
    entrar.mutate(v.data);
  });

  const emailDiferente =
    aceitar.error instanceof FalhaApi &&
    (aceitar.error.status === 403 || aceitar.error.codigo === "email_diferente");

  return (
    <div className="login">
      <div className="card stack">
        <div className="logo" style={{ padding: 0 }}>
          <span className="logo-mark" aria-hidden>L</span> Laura Growth
        </div>

        <div>
          <h1>Convite para {dados.organizacaoNome}</h1>
          <p className="muted" style={{ margin: "6px 0 0" }}>
            E-mail convidado: <b className="num">{dados.emailMascarado}</b>
          </p>
        </div>

        {!dados.contaExistente ? (
          <form className="stack" onSubmit={aceitarContaNova} noValidate>
            <p className="small muted" style={{ margin: 0 }}>
              Crie sua conta para entrar na organização <b>{dados.organizacaoNome}</b>.
            </p>
            <label className="campo">
              Nome
              <input
                type="text"
                autoComplete="name"
                aria-invalid={!!formCadastro.formState.errors.nome}
                {...formCadastro.register("nome")}
              />
              {formCadastro.formState.errors.nome && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {formCadastro.formState.errors.nome.message}
                </span>
              )}
            </label>
            <label className="campo">
              Senha
              <input
                type="password"
                autoComplete="new-password"
                aria-invalid={!!formCadastro.formState.errors.senha}
                {...formCadastro.register("senha")}
              />
              <span className="small muted">Mínimo de 10 caracteres.</span>
              {formCadastro.formState.errors.senha && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {formCadastro.formState.errors.senha.message}
                </span>
              )}
            </label>
            {aceitar.isError && (
              <div className="alerta t-danger" role="alert">
                {mensagemErroAceite(aceitar.error)}
              </div>
            )}
            <button type="submit" className="btn primario" disabled={aceitar.isPending}>
              {aceitar.isPending ? "Criando conta…" : "Criar conta e aceitar convite"}
            </button>
          </form>
        ) : sessao.data ? (
          <div className="stack">
            <p className="small" style={{ margin: 0 }}>
              Já existe uma conta com este e-mail. Você está conectado como <b>{sessao.data.usuario.email}</b>.
            </p>
            {aceitar.isError && (
              <div className="alerta t-danger" role="alert">
                {mensagemErroAceite(aceitar.error)}
              </div>
            )}
            <button
              type="button"
              className="btn primario"
              disabled={aceitar.isPending}
              onClick={() =>
                aceitar.mutate(
                  {},
                  {
                    onSuccess: () => router.replace("/inicio"),
                    onError: (e) => {
                      if (e instanceof FalhaApi && e.status === 401) {
                        qc.removeQueries({ queryKey: ["sessao"] });
                      }
                    },
                  },
                )
              }
            >
              {aceitar.isPending ? "Aceitando…" : "Aceitar convite"}
            </button>
            <button
              type="button"
              className="btn"
              disabled={sair.isPending}
              onClick={() => {
                aceitar.reset();
                sair.mutate();
              }}
            >
              {emailDiferente ? "Sair e entrar com o e-mail convidado" : "Entrar com outra conta"}
            </button>
          </div>
        ) : (
          <form className="stack" onSubmit={entrarInline} noValidate>
            <p className="small muted" style={{ margin: 0 }}>
              Já existe uma conta com este e-mail. Entre com ela para aceitar o convite.
            </p>
            {aceitar.isError && (
              <div className="alerta t-danger" role="alert">
                {mensagemErroAceite(aceitar.error)}
              </div>
            )}
            <label className="campo">
              E-mail
              <input
                type="email"
                autoComplete="username"
                placeholder="voce@exemplo.com"
                aria-invalid={!!formLogin.formState.errors.email}
                {...formLogin.register("email")}
              />
              {formLogin.formState.errors.email && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {formLogin.formState.errors.email.message}
                </span>
              )}
            </label>
            <label className="campo">
              Senha
              <input
                type="password"
                autoComplete="current-password"
                aria-invalid={!!formLogin.formState.errors.senha}
                {...formLogin.register("senha")}
              />
              {formLogin.formState.errors.senha && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {formLogin.formState.errors.senha.message}
                </span>
              )}
            </label>
            {entrar.isError && (
              <div className="alerta t-danger" role="alert">
                {entrar.error instanceof FalhaApi
                  ? entrar.error.message
                  : "Não foi possível entrar. Tente de novo."}
              </div>
            )}
            <button type="submit" className="btn primario" disabled={entrar.isPending}>
              {entrar.isPending ? "Entrando…" : "Entrar para aceitar"}
            </button>
            <Link
              href={`/entrar?proximo=${encodeURIComponent(`/convite/${token}`)}`}
              className="small"
            >
              Ir para a tela de login
            </Link>
          </form>
        )}
      </div>
    </div>
  );
}
