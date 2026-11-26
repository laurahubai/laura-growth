-- Etapa 2B: acessos por pessoa e por alçada (PRM-01 a PRM-05).
-- Migração apenas aditiva.
--
-- Como desfazer (somente após pg_dump do banco):
--   DROP TABLE IF EXISTS registro_acessos;
--   DROP TABLE IF EXISTS convites;
--   DROP TABLE IF EXISTS alcadas_membro;

CREATE TABLE alcadas_membro (
  organizacao_id  uuid NOT NULL,
  usuario_id      uuid NOT NULL,
  alcada          text NOT NULL CHECK (alcada IN (
                    'ver_marca_posts',
                    'cmo_dna',
                    'aprovar_plano',
                    'editar_posts',
                    'aprovar_posts',
                    'conectar_redes',
                    'ver_creditos',
                    'gerenciar_pessoas'
                  )),
  concedida_por   uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  concedida_em    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organizacao_id, usuario_id, alcada),
  FOREIGN KEY (organizacao_id, usuario_id) REFERENCES membros (organizacao_id, usuario_id) ON DELETE CASCADE
);

CREATE TABLE convites (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organizacao_id  uuid NOT NULL REFERENCES organizacoes (id) ON DELETE CASCADE,
  email           text NOT NULL,
  alcadas         text[] NOT NULL DEFAULT '{}' CHECK (alcadas <@ ARRAY[
                    'ver_marca_posts',
                    'cmo_dna',
                    'aprovar_plano',
                    'editar_posts',
                    'aprovar_posts',
                    'conectar_redes',
                    'ver_creditos',
                    'gerenciar_pessoas'
                  ]::text[]),
  token_hash      bytea UNIQUE,
  criado_por      uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  criado_em       timestamptz NOT NULL DEFAULT now(),
  expira_em       timestamptz NOT NULL,
  aceito_em       timestamptz,
  aceito_por      uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  revogado_em     timestamptz,
  envio_estado    text NOT NULL DEFAULT 'pendente' CHECK (envio_estado IN ('enviado', 'falhou', 'pendente'))
);
CREATE INDEX convites_org_criado ON convites (organizacao_id, criado_em DESC);
CREATE INDEX convites_org_email ON convites (organizacao_id, lower(email));

CREATE TABLE registro_acessos (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organizacao_id   uuid REFERENCES organizacoes (id) ON DELETE CASCADE,
  ator_id          uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  alvo_usuario_id  uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  alvo_email       text,
  acao             text NOT NULL,
  detalhe          jsonb NOT NULL DEFAULT '{}'::jsonb,
  em               timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registro_acessos_org_em ON registro_acessos (organizacao_id, em DESC);
CREATE INDEX registro_acessos_em ON registro_acessos (em DESC);
