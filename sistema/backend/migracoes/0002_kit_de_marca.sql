-- Etapa 2: kit de marca (MKT-01, MKT-03, MKT-04).
-- Um kit por organização. Tudo é lido e gravado pela organização ativa da sessão.

CREATE TABLE kits_marca (
  organizacao_id  uuid PRIMARY KEY REFERENCES organizacoes (id) ON DELETE CASCADE,
  site            text NOT NULL DEFAULT '',
  perfis          text NOT NULL DEFAULT '',
  tom_de_voz      text NOT NULL DEFAULT '',
  publico         text NOT NULL DEFAULT '',
  temas           text NOT NULL DEFAULT '',
  nunca_dizer     text NOT NULL DEFAULT '',
  fontes          text NOT NULL DEFAULT '',
  exemplos        text NOT NULL DEFAULT '',
  -- MKT-04: de 3 a 5 assuntos fixos (o limite de 5 é garantido aqui; o mínimo é um aviso na tela).
  pilares         text[] NOT NULL DEFAULT '{}' CHECK (cardinality(pilares) <= 5),
  cores           text[] NOT NULL DEFAULT '{}' CHECK (cardinality(cores) <= 5),
  atualizado_em   timestamptz NOT NULL DEFAULT now(),
  atualizado_por  uuid REFERENCES usuarios (id) ON DELETE SET NULL
);

-- Logo e arquivos de referência (manual da marca, exemplos de posts).
-- Guardados no próprio banco: poucos arquivos, pequenos, e o backup do banco leva tudo junto.
CREATE TABLE arquivos_marca (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organizacao_id  uuid NOT NULL REFERENCES organizacoes (id) ON DELETE CASCADE,
  tipo            text NOT NULL CHECK (tipo IN ('logo', 'referencia')),
  nome            text NOT NULL,
  mime            text NOT NULL,
  tamanho         integer NOT NULL,
  conteudo        bytea NOT NULL,
  criado_em       timestamptz NOT NULL DEFAULT now(),
  criado_por      uuid REFERENCES usuarios (id) ON DELETE SET NULL
);
CREATE INDEX arquivos_marca_org ON arquivos_marca (organizacao_id, criado_em);
-- No máximo um logo por organização.
CREATE UNIQUE INDEX arquivos_marca_um_logo ON arquivos_marca (organizacao_id) WHERE tipo = 'logo';
