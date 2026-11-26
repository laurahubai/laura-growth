"use client";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Check, Copy, MailPlus, RefreshCw, Shield, Trash2, UserMinus, X } from "lucide-react";
import {
  PedidoCriarConvite,
  rotuloAlcada,
  type Alcada,
  type ConjuntoAlcadas,
  type PessoaMembro,
  type RespostaConviteCriado,
} from "@laura-growth/contratos";
import { SeletorAlcadas } from "@/components/SeletorAlcadas";
import { SemAcesso } from "@/components/SemAcesso";
import { TabelaRegistroAcessos } from "@/components/TabelaRegistroAcessos";
import {
  erroSemPermissao,
  FalhaApi,
  useCancelarConvite,
  useCriarConvite,
  usePessoas,
  useReenviarConvite,
  useRegistroAcessos,
  useRemoverPessoa,
  useSalvarAlcadasPessoa,
  useSessao,
} from "@/lib/api";
import { etiquetaEnvioConvite, formatarDataHora } from "@/lib/formato";

export default function PaginaPessoas() {
  const { data: sessao } = useSessao();
  const orgId = sessao?.organizacaoAtiva.id ?? "";
  const pessoas = usePessoas(orgId);
  const registro = useRegistroAcessos(orgId);
  const criarConvite = useCriarConvite(orgId);
  const reenviarConvite = useReenviarConvite(orgId);
  const cancelarConvite = useCancelarConvite(orgId);
  const removerPessoa = useRemoverPessoa(orgId);

  const [editando, setEditando] = useState<PessoaMembro | null>(null);
  const [alcadasConvite, setAlcadasConvite] = useState<Alcada[]>([]);
  const [versaoFormConvite, setVersaoFormConvite] = useState(0);
  const [conviteRecente, setConviteRecente] = useState<RespostaConviteCriado | null>(null);
  const [copiado, setCopiado] = useState(false);
  const [erroAcao, setErroAcao] = useState("");

  const {
    register,
    handleSubmit,
    setError,
    reset: resetarFormConvite,
    formState: { errors },
  } = useForm<{ email: string }>({ defaultValues: { email: "" } });

  if (!sessao) return null;

  if (erroSemPermissao(pessoas.error) || erroSemPermissao(registro.error)) {
    const msg =
      (pessoas.error instanceof FalhaApi && pessoas.error.message) ||
      (registro.error instanceof FalhaApi && registro.error.message) ||
      "Você não tem permissão para gerenciar pessoas nesta organização.";
    return <SemAcesso mensagem={msg} />;
  }

  if (pessoas.isPending) return <p className="muted">Carregando…</p>;
  if (pessoas.isError) {
    return (
      <div className="alerta t-danger" role="alert">
        {pessoas.error.message}
      </div>
    );
  }

  const dados = pessoas.data;

  const enviarConvite = handleSubmit((campos) => {
    setErroAcao("");
    criarConvite.reset();
    const emailLimpo = campos.email.trim().toLowerCase();
    const validado = PedidoCriarConvite.safeParse({
      email: emailLimpo,
      alcadas: alcadasConvite,
    });
    if (!validado.success) {
      const erroEmail = validado.error.issues.find((i) => i.path[0] === "email");
      setError("email", { message: erroEmail ? "Informe um e-mail válido." : "Confira os dados do convite." });
      return;
    }

    criarConvite.mutate(validado.data, {
      onSuccess: (res) => {
        setConviteRecente(res);
        setCopiado(false);
        resetarFormConvite();
        setAlcadasConvite([]);
        setVersaoFormConvite((v) => v + 1);
      },
    });
  });

  const copiarLink = async (link: string) => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(link);
      }
      setCopiado(true);
    } catch {
      setCopiado(false);
    }
  };

  const reenviar = (id: string) => {
    setErroAcao("");
    reenviarConvite.mutate(id, {
      onSuccess: (res) => {
        setConviteRecente(res);
        setCopiado(false);
      },
      onError: (e) => {
        setErroAcao(e instanceof FalhaApi ? e.message : "Não foi possível reenviar o convite.");
      },
    });
  };

  const cancelar = (id: string, email: string) => {
    if (!confirm(`Cancelar o convite para "${email}"?`)) return;
    setErroAcao("");
    cancelarConvite.mutate(id, {
      onSuccess: () => {
        if (conviteRecente?.convite.id === id) setConviteRecente(null);
      },
      onError: (e) => {
        setErroAcao(e instanceof FalhaApi ? e.message : "Não foi possível cancelar o convite.");
      },
    });
  };

  const remover = (m: PessoaMembro) => {
    if (!confirm(`Remover "${m.nome}" (${m.email}) da organização?`)) return;
    setErroAcao("");
    removerPessoa.mutate(m.usuarioId, {
      onError: (e) => {
        setErroAcao(e instanceof FalhaApi ? e.message : "Não foi possível remover a pessoa.");
      },
    });
  };

  return (
    <div className="stack" style={{ gap: 20 }}>
      <div className="page-header" style={{ marginBottom: 0 }}>
        <div>
          <h1>Pessoas</h1>
          <p>
            Quem tem acesso e o que cada pessoa pode fazer na organização <b>{sessao.organizacaoAtiva.nome}</b>.
          </p>
        </div>
      </div>

      {erroAcao && (
        <div className="alerta t-danger" role="alert">
          {erroAcao}
        </div>
      )}

      {conviteRecente && (
        <section className="card stack" data-testid="convite-link-gerado" aria-live="polite">
          <div className="row" style={{ justifyContent: "space-between" }}>
            <h2 style={{ margin: 0 }}>Link de convite pronto</h2>
            <button
              type="button"
              className="btn peq"
              aria-label="Fechar aviso do link"
              onClick={() => setConviteRecente(null)}
            >
              <X size={14} aria-hidden />
            </button>
          </div>
          {(conviteRecente.mensagemEnvio || conviteRecente.convite.envioEstado === "falhou") && (
            <div className="alerta t-warning" role="alert">
              {conviteRecente.mensagemEnvio ||
                "Não foi possível enviar o e-mail do convite. Copie o link abaixo e envie manualmente."}
            </div>
          )}
          <p className="small muted" style={{ margin: 0 }}>
            Convite para <b>{conviteRecente.convite.email}</b>. Copie o link agora — por segurança ele não fica visível depois que você sair desta página.
          </p>
          <div className="row" style={{ flexWrap: "nowrap" }}>
            <input
              type="text"
              readOnly
              value={conviteRecente.link}
              aria-label="Link do convite"
              style={{ flex: 1, minWidth: 0 }}
              onClick={(e) => e.currentTarget.select()}
            />
            <button
              type="button"
              className="btn primario"
              onClick={() => copiarLink(conviteRecente.link)}
            >
              {copiado ? <Check size={16} aria-hidden /> : <Copy size={16} aria-hidden />}
              {copiado ? "Link copiado" : "Copiar link"}
            </button>
          </div>
        </section>
      )}

      <div className="grid g2" style={{ alignItems: "start" }}>
        <section className="card stack" data-testid="lista-membros">
          <h2 style={{ margin: 0 }}>Pessoas com acesso</h2>
          {dados.membros.length === 0 ? (
            <p className="muted" style={{ margin: 0 }}>Nenhuma pessoa cadastrada nesta organização.</p>
          ) : (
            <div className="rolagem">
              <table>
                <thead>
                  <tr>
                    <th>Pessoa</th>
                    <th>Papel</th>
                    <th>Alçadas</th>
                    <th>Ações</th>
                  </tr>
                </thead>
                <tbody>
                  {dados.membros.map((m) => {
                    const ehDono = m.papel === "dono";
                    const ehVoce = m.usuarioId === sessao.usuario.id;
                    const bloqueado = ehDono || ehVoce;
                    return (
                      <tr key={m.usuarioId} data-testid={`membro-${m.email}`}>
                        <td>
                          <div>
                            <b>{m.nome}</b> {ehVoce && <span className="small muted">(você)</span>}
                          </div>
                          <div className="small muted">{m.email}</div>
                        </td>
                        <td>
                          <span className={`badge ${ehDono ? "t-brand" : "t-neutral"}`}>
                            {ehDono ? "Dono" : "Membro"}
                          </span>
                        </td>
                        <td>
                          {ehDono ? (
                            <span className="badge t-brand">Todas as alçadas</span>
                          ) : m.alcadas.length === 0 ? (
                            <span className="small muted">Nenhuma alçada</span>
                          ) : (
                            <div className="row" style={{ gap: 4 }}>
                              {m.alcadas.map((a) => (
                                <span key={a} className="badge t-neutral">
                                  {rotuloAlcada[a]}
                                </span>
                              ))}
                            </div>
                          )}
                        </td>
                        <td>
                          {bloqueado ? (
                            <span className="small muted">
                              {ehDono ? "Dono não editável" : "Sua conta"}
                            </span>
                          ) : (
                            <div className="row" style={{ flexWrap: "nowrap" }}>
                              <button
                                type="button"
                                className="btn peq"
                                onClick={() => setEditando(m)}
                              >
                                <Shield size={14} aria-hidden /> Editar alçadas
                              </button>
                              <button
                                type="button"
                                className="btn peq perigo"
                                disabled={removerPessoa.isPending}
                                onClick={() => remover(m)}
                              >
                                <UserMinus size={14} aria-hidden /> Remover
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="card stack" data-testid="form-convite">
          <h2 style={{ margin: 0 }}>Convidar pessoa</h2>
          <form className="stack" onSubmit={enviarConvite} noValidate>
            <label className="campo">
              E-mail
              <input
                type="email"
                placeholder="pessoa@empresa.com"
                aria-invalid={!!errors.email}
                {...register("email")}
              />
              {errors.email && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.email.message}
                </span>
              )}
            </label>

            <SeletorAlcadas
              key={versaoFormConvite}
              alcadas={alcadasConvite}
              aoMudar={setAlcadasConvite}
              conjuntos={dados.conjuntos}
            />

            {criarConvite.isError && (
              <div className="alerta t-danger" role="alert">
                {criarConvite.error instanceof FalhaApi
                  ? criarConvite.error.message
                  : "Não foi possível enviar o convite."}
              </div>
            )}

            <button type="submit" className="btn primario" disabled={criarConvite.isPending}>
              <MailPlus size={16} aria-hidden />
              {criarConvite.isPending ? "Enviando convite…" : "Enviar convite"}
            </button>
          </form>
        </section>
      </div>

      <section className="card stack" data-testid="lista-convites">
        <h2 style={{ margin: 0 }}>Convites pendentes</h2>
        {dados.convites.length === 0 ? (
          <p className="muted" style={{ margin: 0 }}>Nenhum convite pendente no momento.</p>
        ) : (
          <div className="rolagem">
            <table>
              <thead>
                <tr>
                  <th>E-mail</th>
                  <th>Alçadas</th>
                  <th>Envio</th>
                  <th>Expira em</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {dados.convites.map((c) => {
                  const et = etiquetaEnvioConvite(c.envioEstado);
                  return (
                    <tr key={c.id} data-testid={`convite-${c.email}`}>
                      <td>
                        <div><b>{c.email}</b></div>
                        {c.criadoPorNome && (
                          <div className="small muted">Convidado por {c.criadoPorNome}</div>
                        )}
                      </td>
                      <td>
                        {c.alcadas.length === 0 ? (
                          <span className="small muted">Nenhuma alçada</span>
                        ) : (
                          <div className="row" style={{ gap: 4 }}>
                            {c.alcadas.map((a) => (
                              <span key={a} className="badge t-neutral">
                                {rotuloAlcada[a]}
                              </span>
                            ))}
                          </div>
                        )}
                      </td>
                      <td>
                        <span className={`badge ${et.classe}`}>{et.texto}</span>
                      </td>
                      <td className="num small">{formatarDataHora(c.expiraEm)}</td>
                      <td>
                        <div className="row" style={{ flexWrap: "nowrap" }}>
                          <button
                            type="button"
                            className="btn peq"
                            disabled={reenviarConvite.isPending}
                            onClick={() => reenviar(c.id)}
                          >
                            <RefreshCw size={14} aria-hidden /> Reenviar
                          </button>
                          <button
                            type="button"
                            className="btn peq perigo"
                            disabled={cancelarConvite.isPending}
                            onClick={() => cancelar(c.id, c.email)}
                          >
                            <Trash2 size={14} aria-hidden /> Cancelar
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="card stack" data-testid="historico-acessos" aria-busy={registro.isPending}>
        <h2 style={{ margin: 0 }}>Histórico de acessos</h2>
        {registro.isPending && <p className="muted" style={{ margin: 0 }}>Carregando histórico…</p>}
        {registro.isError && (
          <div className="alerta t-danger" role="alert">
            {registro.error.message}
          </div>
        )}
        {registro.data && (
          <TabelaRegistroAcessos
            registros={registro.data.registros}
            vazio="Nenhuma alteração de acesso registrada nesta organização."
          />
        )}
      </section>

      {editando && (
        <GavetaEditarAlcadas
          orgId={orgId}
          pessoa={editando}
          conjuntos={dados.conjuntos}
          aoFechar={() => setEditando(null)}
        />
      )}
    </div>
  );
}

function GavetaEditarAlcadas({
  orgId,
  pessoa,
  conjuntos,
  aoFechar,
}: {
  orgId: string;
  pessoa: PessoaMembro;
  conjuntos: ConjuntoAlcadas[];
  aoFechar: () => void;
}) {
  const salvar = useSalvarAlcadasPessoa(orgId);
  const [alcadas, setAlcadas] = useState<Alcada[]>(pessoa.alcadas);

  useEffect(() => {
    const aoTeclar = (e: KeyboardEvent) => {
      if (e.key === "Escape") aoFechar();
    };
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [aoFechar]);

  const enviar = (e: React.FormEvent) => {
    e.preventDefault();
    salvar.mutate(
      { usuarioId: pessoa.usuarioId, alcadas },
      { onSuccess: () => aoFechar() },
    );
  };

  return (
    <div className="drawer-fundo" onClick={aoFechar}>
      <div
        className="drawer stack"
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-editar-alcadas"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="row" style={{ justifyContent: "space-between" }}>
          <h2 id="titulo-editar-alcadas" style={{ margin: 0 }}>Editar alçadas</h2>
          <button type="button" className="btn peq" aria-label="Fechar" onClick={aoFechar}>
            <X size={16} aria-hidden />
          </button>
        </div>

        <p className="small muted" style={{ margin: 0 }}>
          <b>{pessoa.nome}</b> ({pessoa.email})
        </p>

        <form className="stack" onSubmit={enviar}>
          <SeletorAlcadas
            alcadas={alcadas}
            aoMudar={setAlcadas}
            conjuntos={conjuntos}
          />

          {salvar.isError && (
            <div className="alerta t-danger" role="alert">
              {salvar.error instanceof FalhaApi ? salvar.error.message : "Não foi possível salvar as alçadas."}
            </div>
          )}

          <div className="row" style={{ justifyContent: "flex-end", marginTop: 8 }}>
            <button type="button" className="btn" onClick={aoFechar}>
              Cancelar
            </button>
            <button type="submit" className="btn primario" disabled={salvar.isPending}>
              {salvar.isPending ? "Salvando…" : "Salvar alçadas"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
