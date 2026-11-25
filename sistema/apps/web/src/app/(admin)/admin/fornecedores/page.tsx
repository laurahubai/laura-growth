"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Plus, X } from "lucide-react";
import {
  enderecoPadraoFornecedor,
  nomeTipoFornecedor,
  PedidoEditarFornecedor,
  PedidoNovoFornecedor,
  TipoFornecedor,
  type Fornecedor,
  type UsoPermitido,
} from "@laura-growth/contratos";
import { useRedirecionarSemPermissao } from "@/components/ShellAdmin";
import { FalhaApi, useCriarFornecedor, useEditarFornecedor, useFornecedoresAdmin } from "@/lib/api";
import { etiquetaSituacao, formatarUsd } from "@/lib/formato";

type ModoFormulario = { tipo: "novo" } | { tipo: "editar"; fornecedor: Fornecedor } | null;

export default function PaginaFornecedores() {
  const lista = useFornecedoresAdmin();
  const editar = useEditarFornecedor();
  const [modo, setModo] = useState<ModoFormulario>(null);
  const [erroAcao, setErroAcao] = useState<string>("");

  const bloqueado = useRedirecionarSemPermissao(lista.error || editar.error);
  if (bloqueado) return <div className="vazio" role="alert">{bloqueado}</div>;

  const alternarAtivo = (f: Fornecedor) => {
    setErroAcao("");
    editar.mutate(
      { id: f.id, dados: { ativo: !f.ativo } },
      { onError: (e) => setErroAcao(e instanceof FalhaApi ? e.message : "Não foi possível atualizar o fornecedor.") },
    );
  };

  return (
    <>
      <div className="page-header">
        <div>
          <h1>Fornecedores de IA</h1>
          <p>Chaves de API, limites mensais de gasto e uso permitido na plataforma.</p>
        </div>
        <button type="button" className="btn primario" onClick={() => setModo({ tipo: "novo" })}>
          <Plus size={16} aria-hidden /> Novo fornecedor
        </button>
      </div>

      {erroAcao && (
        <div className="alerta t-danger" role="alert" style={{ marginBottom: 16 }}>
          {erroAcao}
        </div>
      )}

      <section className="card" data-testid="lista-fornecedores" aria-busy={lista.isPending}>
        {lista.isPending && <p className="vazio">Carregando fornecedores…</p>}
        {lista.isError && (
          <div className="alerta t-danger" role="alert">
            {lista.error.message}
          </div>
        )}
        {lista.data && lista.data.fornecedores.length === 0 && (
          <p className="vazio">Nenhum fornecedor cadastrado ainda.</p>
        )}
        {lista.data && lista.data.fornecedores.length > 0 && (
          <div className="rolagem">
            <table>
              <thead>
                <tr>
                  <th>Nome</th>
                  <th>Tipo</th>
                  <th>Endereço</th>
                  <th>Chave</th>
                  <th>Uso</th>
                  <th className="d">Limite mensal</th>
                  <th className="d">Gasto do mês</th>
                  <th>Situação</th>
                  <th>Ativo</th>
                  <th>Ações</th>
                </tr>
              </thead>
              <tbody>
                {lista.data.fornecedores.map((f) => {
                  const sit = etiquetaSituacao(f.situacao);
                  return (
                    <tr key={f.id}>
                      <td>
                        <div className="row" style={{ gap: 6 }}>
                          <Link href={`/admin/fornecedores/${f.id}`} style={{ fontWeight: 600 }}>
                            {f.nome}
                          </Link>
                          {f.simulado && <span className="badge t-ai">Simulado</span>}
                        </div>
                      </td>
                      <td>{nomeTipoFornecedor[f.tipo]}</td>
                      <td className="small muted">{f.endereco || "—"}</td>
                      <td className="num small">{f.chaveFinal ? `••••${f.chaveFinal}` : "sem chave"}</td>
                      <td className="small">{f.usoPermitido === "casa" ? "Só a casa" : "Todas"}</td>
                      <td className="d">{formatarUsd(f.limiteMensalUsd)}</td>
                      <td className="d">{formatarUsd(f.gastoMesUsd)}</td>
                      <td>
                        <span className={`badge ${sit.classe}`}>{sit.texto}</span>
                      </td>
                      <td>
                        <span className={`badge ${f.ativo ? "t-success" : "t-neutral"}`}>
                          {f.ativo ? "Ativo" : "Inativo"}
                        </span>
                      </td>
                      <td>
                        <div className="row" style={{ flexWrap: "nowrap" }}>
                          <Link href={`/admin/fornecedores/${f.id}`} className="btn peq">
                            Modelos
                          </Link>
                          <button
                            type="button"
                            className="btn peq"
                            onClick={() => setModo({ tipo: "editar", fornecedor: f })}
                          >
                            Editar
                          </button>
                          <button
                            type="button"
                            className="btn peq"
                            disabled={editar.isPending}
                            onClick={() => alternarAtivo(f)}
                          >
                            {f.ativo ? "Desativar" : "Reativar"}
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

      {modo?.tipo === "novo" && <GavetaNovoFornecedor aoFechar={() => setModo(null)} />}
      {modo?.tipo === "editar" && (
        <GavetaEditarFornecedor fornecedor={modo.fornecedor} aoFechar={() => setModo(null)} />
      )}
    </>
  );
}

type CamposNovoFornecedor = {
  tipo: TipoFornecedor;
  nome: string;
  endereco: string;
  chave: string;
  limiteMensalUsd: string;
  avisoPercentual: string;
  usoPermitido: UsoPermitido;
};

function GavetaNovoFornecedor({ aoFechar }: { aoFechar: () => void }) {
  const criar = useCriarFornecedor();
  useRedirecionarSemPermissao(criar.error);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    setError,
    reset,
    formState: { errors },
  } = useForm<CamposNovoFornecedor>({
    defaultValues: {
      tipo: "openai",
      nome: "",
      endereco: enderecoPadraoFornecedor.openai,
      chave: "",
      limiteMensalUsd: "50",
      avisoPercentual: "80",
      usoPermitido: "todas",
    },
  });

  const tipoAtual = watch("tipo");
  const ehSimulado = tipoAtual === "simulado";

  useEffect(() => {
    const aoTeclar = (e: KeyboardEvent) => {
      if (e.key === "Escape") aoFechar();
    };
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [aoFechar]);

  const trocarTipo = (novoTipo: TipoFornecedor) => {
    setValue("tipo", novoTipo);
    setValue("endereco", enderecoPadraoFornecedor[novoTipo]);
    if (novoTipo === "simulado") {
      setValue("chave", "");
    }
  };

  const enviar = handleSubmit((campos) => {
    criar.reset();
    const limite = Number(campos.limiteMensalUsd);
    const aviso = Number(campos.avisoPercentual);
    if (!Number.isFinite(limite) || limite <= 0) {
      setError("limiteMensalUsd", { message: "Informe um limite maior que zero." });
      return;
    }
    if (!Number.isInteger(aviso) || aviso < 1 || aviso > 100) {
      setError("avisoPercentual", { message: "Informe um percentual inteiro entre 1 e 100." });
      return;
    }
    if (!ehSimulado && !campos.chave.trim()) {
      setError("chave", { message: "Informe a chave de API." });
      return;
    }

    const candidato = {
      nome: campos.nome,
      tipo: campos.tipo,
      endereco: ehSimulado ? undefined : campos.endereco,
      chave: ehSimulado ? undefined : campos.chave,
      usoPermitido: campos.usoPermitido,
      limiteMensalUsd: limite,
      avisoPercentual: aviso,
    };

    const validado = PedidoNovoFornecedor.safeParse(candidato);
    if (!validado.success) {
      for (const issue of validado.error.issues) {
        const campo = issue.path[0] as keyof CamposNovoFornecedor | undefined;
        if (campo) setError(campo, { message: issue.message });
      }
      return;
    }

    criar.mutate(validado.data, {
      onSuccess: () => {
        setValue("chave", "");
        reset();
        criar.reset();
        aoFechar();
      },
      onError: (e) => {
        if (e instanceof FalhaApi) {
          if (e.codigo === "nome_repetido") setError("nome", { message: e.message });
          if (e.codigo === "endereco_invalido") setError("endereco", { message: e.message });
          if (e.codigo === "chave_obrigatoria") setError("chave", { message: e.message });
        }
      },
    });
  });

  const erroGeral = criar.error instanceof FalhaApi ? criar.error.message : criar.error ? "Não foi possível salvar." : "";

  return (
    <div className="drawer-fundo" onClick={aoFechar}>
      <div
        className="drawer stack"
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-novo-fornecedor"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="row" style={{ justifyContent: "space-between" }}>
          <h2 id="titulo-novo-fornecedor" style={{ margin: 0 }}>Novo fornecedor</h2>
          <button type="button" className="btn peq" aria-label="Fechar" onClick={aoFechar}>
            <X size={16} aria-hidden />
          </button>
        </div>

        <form className="stack" onSubmit={enviar} noValidate>
          <label className="campo">
            Tipo
            <select
              value={tipoAtual}
              onChange={(e) => trocarTipo(e.target.value as TipoFornecedor)}
            >
              {TipoFornecedor.options.map((t) => (
                <option key={t} value={t}>
                  {nomeTipoFornecedor[t]}
                </option>
              ))}
            </select>
          </label>

          <label className="campo">
            Nome
            <input type="text" aria-invalid={!!errors.nome} {...register("nome")} />
            {errors.nome && <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.nome.message}</span>}
          </label>

          {!ehSimulado && (
            <label className="campo">
              Endereço (URL)
              <input
                type="text"
                placeholder="https://..."
                aria-invalid={!!errors.endereco}
                {...register("endereco")}
              />
              {errors.endereco && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.endereco.message}</span>
              )}
            </label>
          )}

          {!ehSimulado && (
            <label className="campo">
              Chave de API
              <input
                type="password"
                autoComplete="new-password"
                data-1p-ignore
                data-lpignore="true"
                data-form-type="other"
                aria-invalid={!!errors.chave}
                {...register("chave")}
              />
              {errors.chave && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.chave.message}</span>
              )}
            </label>
          )}

          <div className="grid g2" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <label className="campo">
              Limite mensal (US$)
              <input
                type="number"
                step="0.01"
                min="0.01"
                aria-invalid={!!errors.limiteMensalUsd}
                {...register("limiteMensalUsd")}
              />
              {errors.limiteMensalUsd && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.limiteMensalUsd.message}
                </span>
              )}
            </label>

            <label className="campo">
              Aviso (%)
              <input
                type="number"
                step="1"
                min="1"
                max="100"
                aria-invalid={!!errors.avisoPercentual}
                {...register("avisoPercentual")}
              />
              {errors.avisoPercentual && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.avisoPercentual.message}
                </span>
              )}
            </label>
          </div>

          <label className="campo">
            Uso permitido
            <select {...register("usoPermitido")}>
              <option value="todas">Todas as organizações</option>
              <option value="casa">Só a casa</option>
            </select>
          </label>

          {erroGeral && (
            <p role="alert" style={{ color: "var(--laura-danger)", margin: 0 }}>
              {erroGeral}
            </p>
          )}

          <div className="row" style={{ justifyContent: "flex-end", marginTop: 8 }}>
            <button type="button" className="btn" onClick={aoFechar}>
              Cancelar
            </button>
            <button type="submit" className="btn primario" disabled={criar.isPending}>
              {criar.isPending ? "Salvando…" : "Salvar fornecedor"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

type CamposEditarFornecedor = {
  nome: string;
  endereco: string;
  chave: string;
  limiteMensalUsd: string;
  avisoPercentual: string;
  usoPermitido: UsoPermitido;
  ativo: boolean;
};

function GavetaEditarFornecedor({ fornecedor, aoFechar }: { fornecedor: Fornecedor; aoFechar: () => void }) {
  const editar = useEditarFornecedor();
  useRedirecionarSemPermissao(editar.error);
  const ehSimulado = fornecedor.tipo === "simulado";

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    reset,
    formState: { errors },
  } = useForm<CamposEditarFornecedor>({
    defaultValues: {
      nome: fornecedor.nome,
      endereco: fornecedor.endereco,
      chave: "",
      limiteMensalUsd: String(fornecedor.limiteMensalUsd),
      avisoPercentual: String(fornecedor.avisoPercentual),
      usoPermitido: fornecedor.usoPermitido,
      ativo: fornecedor.ativo,
    },
  });

  useEffect(() => {
    const aoTeclar = (e: KeyboardEvent) => {
      if (e.key === "Escape") aoFechar();
    };
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [aoFechar]);

  const enviar = handleSubmit((campos) => {
    editar.reset();
    const limite = Number(campos.limiteMensalUsd);
    const aviso = Number(campos.avisoPercentual);
    if (!Number.isFinite(limite) || limite <= 0) {
      setError("limiteMensalUsd", { message: "Informe um limite maior que zero." });
      return;
    }
    if (!Number.isInteger(aviso) || aviso < 1 || aviso > 100) {
      setError("avisoPercentual", { message: "Informe um percentual inteiro entre 1 e 100." });
      return;
    }

    const candidato: PedidoEditarFornecedor = {};
    if (campos.nome.trim() !== fornecedor.nome) candidato.nome = campos.nome;
    if (!ehSimulado && campos.endereco.trim() !== fornecedor.endereco) candidato.endereco = campos.endereco;
    if (!ehSimulado && campos.chave.trim() !== "") candidato.chave = campos.chave;
    if (campos.usoPermitido !== fornecedor.usoPermitido) candidato.usoPermitido = campos.usoPermitido;
    if (limite !== fornecedor.limiteMensalUsd) candidato.limiteMensalUsd = limite;
    if (aviso !== fornecedor.avisoPercentual) candidato.avisoPercentual = aviso;
    if (campos.ativo !== fornecedor.ativo) candidato.ativo = campos.ativo;

    if (Object.keys(candidato).length === 0) {
      setValue("chave", "");
      reset();
      aoFechar();
      return;
    }

    const validado = PedidoEditarFornecedor.safeParse(candidato);
    if (!validado.success) {
      for (const issue of validado.error.issues) {
        const campo = issue.path[0] as keyof CamposEditarFornecedor | undefined;
        if (campo) setError(campo, { message: issue.message });
      }
      return;
    }

    editar.mutate(
      { id: fornecedor.id, dados: validado.data },
      {
        onSuccess: () => {
          setValue("chave", "");
          reset();
          editar.reset();
          aoFechar();
        },
        onError: (e) => {
          if (e instanceof FalhaApi) {
            if (e.codigo === "nome_repetido") setError("nome", { message: e.message });
            if (e.codigo === "endereco_invalido") setError("endereco", { message: e.message });
          }
        },
      },
    );
  });

  const erroGeral =
    editar.error instanceof FalhaApi ? editar.error.message : editar.error ? "Não foi possível salvar." : "";

  return (
    <div className="drawer-fundo" onClick={aoFechar}>
      <div
        className="drawer stack"
        role="dialog"
        aria-modal="true"
        aria-labelledby="titulo-editar-fornecedor"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="row" style={{ justifyContent: "space-between" }}>
          <h2 id="titulo-editar-fornecedor" style={{ margin: 0 }}>Editar fornecedor</h2>
          <button type="button" className="btn peq" aria-label="Fechar" onClick={aoFechar}>
            <X size={16} aria-hidden />
          </button>
        </div>

        <p className="small muted" style={{ margin: 0 }}>
          Tipo: <b>{nomeTipoFornecedor[fornecedor.tipo]}</b>
          {fornecedor.chaveFinal ? ` · Chave atual: ••••${fornecedor.chaveFinal}` : " · Sem chave"}
        </p>

        <form className="stack" onSubmit={enviar} noValidate>
          <label className="campo">
            Nome
            <input type="text" aria-invalid={!!errors.nome} {...register("nome")} />
            {errors.nome && <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.nome.message}</span>}
          </label>

          {!ehSimulado && (
            <label className="campo">
              Endereço (URL)
              <input type="text" aria-invalid={!!errors.endereco} {...register("endereco")} />
              {errors.endereco && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.endereco.message}</span>
              )}
            </label>
          )}

          {!ehSimulado && (
            <label className="campo">
              Trocar chave
              <input
                type="password"
                autoComplete="new-password"
                data-1p-ignore
                data-lpignore="true"
                data-form-type="other"
                placeholder="Deixe vazio para manter a chave atual"
                aria-invalid={!!errors.chave}
                {...register("chave")}
              />
              {errors.chave && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>{errors.chave.message}</span>
              )}
            </label>
          )}

          <div className="grid g2" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <label className="campo">
              Limite mensal (US$)
              <input
                type="number"
                step="0.01"
                min="0.01"
                aria-invalid={!!errors.limiteMensalUsd}
                {...register("limiteMensalUsd")}
              />
              {errors.limiteMensalUsd && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.limiteMensalUsd.message}
                </span>
              )}
            </label>

            <label className="campo">
              Aviso (%)
              <input
                type="number"
                step="1"
                min="1"
                max="100"
                aria-invalid={!!errors.avisoPercentual}
                {...register("avisoPercentual")}
              />
              {errors.avisoPercentual && (
                <span className="small" style={{ color: "var(--laura-danger)" }}>
                  {errors.avisoPercentual.message}
                </span>
              )}
            </label>
          </div>

          <label className="campo">
            Uso permitido
            <select {...register("usoPermitido")}>
              <option value="todas">Todas as organizações</option>
              <option value="casa">Só a casa</option>
            </select>
          </label>

          <label className="row" style={{ fontWeight: 500, fontSize: 13 }}>
            <input type="checkbox" {...register("ativo")} />
            Fornecedor ativo
          </label>

          {erroGeral && (
            <p role="alert" style={{ color: "var(--laura-danger)", margin: 0 }}>
              {erroGeral}
            </p>
          )}

          <div className="row" style={{ justifyContent: "flex-end", marginTop: 8 }}>
            <button type="button" className="btn" onClick={aoFechar}>
              Cancelar
            </button>
            <button type="submit" className="btn primario" disabled={editar.isPending}>
              {editar.isPending ? "Salvando…" : "Salvar alterações"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
