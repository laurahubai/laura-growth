-- Etapa 1: acesso e organizações (ORG-01, ORG-02, ORG-03).

CREATE TABLE usuarios (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  nome        text NOT NULL,
  email       text NOT NULL,
  senha_hash  text NOT NULL,
  criado_em   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX usuarios_email_unico ON usuarios (lower(email));

-- ORG-02: o tipo define, no futuro, quais alertas de regras do setor se aplicam.
CREATE TABLE organizacoes (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  nome       text NOT NULL,
  tipo       text NOT NULL CHECK (tipo IN ('empresa', 'clinica_estetica')),
  criada_em  timestamptz NOT NULL DEFAULT now()
);

-- ORG-03: uma pessoa pode ter acesso a várias organizações.
CREATE TABLE membros (
  organizacao_id  uuid NOT NULL REFERENCES organizacoes (id) ON DELETE CASCADE,
  usuario_id      uuid NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  papel           text NOT NULL DEFAULT 'dono' CHECK (papel IN ('dono', 'membro')),
  criado_em       timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organizacao_id, usuario_id)
);
CREATE INDEX membros_usuario ON membros (usuario_id);

-- Sessão de login. Guardamos só o hash do token; o token fica no cookie do navegador.
-- organizacao_id é a organização ativa: tudo o que a tela vê é filtrado por ela (ORG-01).
CREATE TABLE sessoes (
  token_hash      bytea PRIMARY KEY,
  usuario_id      uuid NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  organizacao_id  uuid REFERENCES organizacoes (id) ON DELETE SET NULL,
  criada_em       timestamptz NOT NULL DEFAULT now(),
  expira_em       timestamptz NOT NULL
);
CREATE INDEX sessoes_usuario ON sessoes (usuario_id);
