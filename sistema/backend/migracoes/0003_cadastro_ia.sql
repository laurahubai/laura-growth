-- Etapa 2A-1: cadastro de IA por chave de API (fornecedores, modelos, preços, rotas e execuções).
-- Migração apenas aditiva; nada é apagado (fornecedores e modelos usam ativo = false).
--
-- Como desfazer (somente após pg_dump do banco):
--   DROP TABLE IF EXISTS ia_execucoes;
--   DROP TABLE IF EXISTS ia_rotas;
--   DROP TABLE IF EXISTS ia_precos;
--   DROP TABLE IF EXISTS ia_modelos;
--   DROP TABLE IF EXISTS ia_fornecedores;
--   ALTER TABLE organizacoes DROP COLUMN IF EXISTS da_casa;
--   ALTER TABLE usuarios DROP COLUMN IF EXISTS admin_plataforma;

ALTER TABLE usuarios ADD COLUMN admin_plataforma boolean NOT NULL DEFAULT false;
ALTER TABLE organizacoes ADD COLUMN da_casa boolean NOT NULL DEFAULT false;

CREATE TABLE ia_fornecedores (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  nome               text NOT NULL,
  tipo               text NOT NULL CHECK (tipo IN ('openai', 'compativel_openai', 'anthropic', 'simulado')),
  cadastro           text NOT NULL DEFAULT 'chave_api' CHECK (cadastro IN ('chave_api', 'assinatura')),
  endereco           text NOT NULL DEFAULT '',
  chave_cifrada      bytea,
  chave_final        text,
  limite_mensal_usd  numeric(12,2) NOT NULL CHECK (limite_mensal_usd > 0),
  aviso_percentual   integer NOT NULL DEFAULT 80 CHECK (aviso_percentual BETWEEN 1 AND 100),
  uso_permitido      text NOT NULL DEFAULT 'todas' CHECK (uso_permitido IN ('todas', 'casa')),
  ativo              boolean NOT NULL DEFAULT true,
  criado_em          timestamptz NOT NULL DEFAULT now(),
  atualizado_em      timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ia_fornecedores_nome_unico ON ia_fornecedores (lower(nome));

CREATE TABLE ia_modelos (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  fornecedor_id      uuid NOT NULL REFERENCES ia_fornecedores (id),
  identificador      text NOT NULL,
  servico            text NOT NULL CHECK (servico IN ('texto', 'imagem')),
  origem             text NOT NULL CHECK (origem IN ('lido', 'manual')),
  limite_mensal_usd  numeric(12,2) CHECK (limite_mensal_usd IS NULL OR limite_mensal_usd > 0),
  aviso_percentual   integer CHECK (aviso_percentual IS NULL OR aviso_percentual BETWEEN 1 AND 100),
  ativo              boolean NOT NULL DEFAULT true,
  criado_em          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (fornecedor_id, identificador)
);

-- Tokens: US$ por 1 milhão de tokens; imagem: US$ por imagem.
CREATE TABLE ia_precos (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  modelo_id        uuid NOT NULL REFERENCES ia_modelos (id),
  unidade          text NOT NULL CHECK (unidade IN ('token_entrada', 'token_saida', 'imagem')),
  preco_usd        numeric(18,8) NOT NULL CHECK (preco_usd >= 0),
  vigencia_inicio  date NOT NULL,
  criado_em        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (modelo_id, unidade, vigencia_inicio)
);

CREATE TABLE ia_rotas (
  atividade            text PRIMARY KEY CHECK (atividade IN ('chat_cmo', 'plano', 'pauta', 'texto', 'imagem', 'revisao')),
  modelo_principal_id  uuid REFERENCES ia_modelos (id),
  modelo_reserva_id    uuid REFERENCES ia_modelos (id),
  atualizado_em        timestamptz NOT NULL DEFAULT now(),
  CHECK (modelo_principal_id IS NULL OR modelo_reserva_id IS NULL OR modelo_principal_id <> modelo_reserva_id)
);

CREATE TABLE ia_execucoes (
  id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  correlacao_id          uuid NOT NULL,
  atividade              text NOT NULL,
  organizacao_id         uuid REFERENCES organizacoes (id),
  usuario_id             uuid REFERENCES usuarios (id),
  fornecedor_id          uuid NOT NULL REFERENCES ia_fornecedores (id),
  modelo_id              uuid NOT NULL REFERENCES ia_modelos (id),
  fornecedor_nome        text NOT NULL,
  modelo_identificador   text NOT NULL,
  tentativa              integer NOT NULL DEFAULT 1,
  atendido_pela_reserva  boolean NOT NULL DEFAULT false,
  status                 text NOT NULL CHECK (status IN ('ok', 'falha', 'pulado_limite')),
  erro                   text,
  tokens_entrada         integer NOT NULL DEFAULT 0,
  tokens_saida           integer NOT NULL DEFAULT 0,
  imagens                integer NOT NULL DEFAULT 0,
  preco_token_entrada    numeric(18,8),
  preco_token_saida      numeric(18,8),
  preco_imagem           numeric(18,8),
  custo_usd              numeric(18,8) NOT NULL DEFAULT 0,
  custo_estimado         boolean NOT NULL DEFAULT false,
  simulado               boolean NOT NULL DEFAULT false,
  criado_em              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ia_execucoes_fornecedor_criado ON ia_execucoes (fornecedor_id, criado_em);
CREATE INDEX ia_execucoes_modelo_criado ON ia_execucoes (modelo_id, criado_em);
