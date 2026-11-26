-- Etapa 2C: DNA da marca (kit ampliado, origens por campo e versões).
-- Migração apenas aditiva; preserva dados e arquivos existentes em kits_marca e arquivos_marca.
--
-- Como desfazer (somente após pg_dump do banco):
--   DROP TABLE IF EXISTS kit_versoes;
--   DROP TABLE IF EXISTS kit_origens;
--   ALTER TABLE kits_marca
--     DROP COLUMN IF EXISTS restricoes_setor,
--     DROP COLUMN IF EXISTS canais_outro,
--     DROP COLUMN IF EXISTS canais,
--     DROP COLUMN IF EXISTS objetivo_90_dias,
--     DROP COLUMN IF EXISTS maior_desafio,
--     DROP COLUMN IF EXISTS como_compram,
--     DROP COLUMN IF EXISTS regioes,
--     DROP COLUMN IF EXISTS modelo_outro,
--     DROP COLUMN IF EXISTS modelo_negocio,
--     DROP COLUMN IF EXISTS diferencial,
--     DROP COLUMN IF EXISTS produtos;

ALTER TABLE kits_marca
  ADD COLUMN produtos         jsonb NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(produtos) = 'array' AND jsonb_array_length(produtos) <= 20),
  ADD COLUMN diferencial      text NOT NULL DEFAULT '',
  ADD COLUMN modelo_negocio   text NOT NULL DEFAULT ''
    CHECK (modelo_negocio IN ('', 'b2b', 'b2c', 'b2b2c', 'outro')),
  ADD COLUMN modelo_outro     text NOT NULL DEFAULT '',
  ADD COLUMN regioes          text NOT NULL DEFAULT '',
  ADD COLUMN como_compram     text NOT NULL DEFAULT '',
  ADD COLUMN maior_desafio    text NOT NULL DEFAULT '',
  ADD COLUMN objetivo_90_dias text NOT NULL DEFAULT '',
  ADD COLUMN canais           text[] NOT NULL DEFAULT '{}'
    CHECK (canais <@ ARRAY[
      'instagram', 'facebook', 'linkedin', 'tiktok', 'youtube',
      'google_meu_negocio', 'site_blog', 'whatsapp', 'email'
    ]::text[]),
  ADD COLUMN canais_outro     text NOT NULL DEFAULT '',
  ADD COLUMN restricoes_setor text NOT NULL DEFAULT '';

CREATE TABLE kit_origens (
  organizacao_id  uuid NOT NULL REFERENCES organizacoes (id) ON DELETE CASCADE,
  campo           text NOT NULL CHECK (campo IN (
                    'site', 'perfis', 'tomDeVoz', 'publico', 'temas', 'nuncaDizer',
                    'fontes', 'exemplos', 'pilares', 'cores', 'produtos', 'diferencial',
                    'modeloNegocio', 'regioes', 'comoCompram', 'maiorDesafio',
                    'objetivo90Dias', 'canais', 'restricoesSetor'
                  )),
  origem          text NOT NULL CHECK (origem IN ('confirmado', 'sugerido_ia')),
  definido_por    uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  definido_em     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organizacao_id, campo),
  CHECK (origem <> 'sugerido_ia' OR definido_por IS NULL)
);

-- Histórico de versões do kit (somente inserção pelo servidor).
CREATE TABLE kit_versoes (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organizacao_id    uuid NOT NULL REFERENCES organizacoes (id) ON DELETE CASCADE,
  numero            integer NOT NULL CHECK (numero >= 1),
  dados             jsonb NOT NULL,
  origens           jsonb NOT NULL DEFAULT '{}'::jsonb,
  campos_alterados  text[] NOT NULL DEFAULT '{}',
  autor_id          uuid REFERENCES usuarios (id) ON DELETE SET NULL,
  autor_tipo        text NOT NULL CHECK (autor_tipo IN ('pessoa', 'ia')),
  criado_em         timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organizacao_id, numero)
);
CREATE INDEX kit_versoes_org_numero ON kit_versoes (organizacao_id, numero DESC);

-- Preenche origem 'confirmado' para todo campo já preenchido antes da migração.
INSERT INTO kit_origens (organizacao_id, campo, origem, definido_por, definido_em)
SELECT k.organizacao_id, v.campo, 'confirmado', k.atualizado_por, k.atualizado_em
FROM kits_marca k
CROSS JOIN LATERAL (
  VALUES
    ('site',       k.site <> ''),
    ('perfis',     k.perfis <> ''),
    ('tomDeVoz',   k.tom_de_voz <> ''),
    ('publico',    k.publico <> ''),
    ('temas',      k.temas <> ''),
    ('nuncaDizer', k.nunca_dizer <> ''),
    ('fontes',     k.fontes <> ''),
    ('exemplos',   k.exemplos <> ''),
    ('pilares',    cardinality(k.pilares) > 0),
    ('cores',      cardinality(k.cores) > 0)
) AS v(campo, preenchido)
WHERE v.preenchido;

-- Cria a versão 1 para cada kit existente, preservando autor e data da última alteração.
INSERT INTO kit_versoes (organizacao_id, numero, dados, origens, campos_alterados, autor_id, autor_tipo, criado_em)
SELECT
  k.organizacao_id,
  1,
  jsonb_build_object(
    'site', k.site,
    'perfis', k.perfis,
    'tomDeVoz', k.tom_de_voz,
    'publico', k.publico,
    'temas', k.temas,
    'nuncaDizer', k.nunca_dizer,
    'fontes', k.fontes,
    'exemplos', k.exemplos,
    'pilares', to_jsonb(k.pilares),
    'cores', to_jsonb(k.cores),
    'produtos', k.produtos,
    'diferencial', k.diferencial,
    'modeloNegocio', k.modelo_negocio,
    'modeloOutro', k.modelo_outro,
    'regioes', k.regioes,
    'comoCompram', k.como_compram,
    'maiorDesafio', k.maior_desafio,
    'objetivo90Dias', k.objetivo_90_dias,
    'canais', to_jsonb(k.canais),
    'canaisOutro', k.canais_outro,
    'restricoesSetor', k.restricoes_setor,
    'atualizadoEm', to_jsonb(k.atualizado_em)
  ),
  COALESCE(
    (
      SELECT jsonb_object_agg(
        ko.campo,
        jsonb_build_object(
          'origem', ko.origem,
          'definidoPor', u.nome,
          'definidoEm', to_jsonb(ko.definido_em)
        )
      )
      FROM kit_origens ko
      LEFT JOIN usuarios u ON u.id = ko.definido_por
      WHERE ko.organizacao_id = k.organizacao_id
    ),
    '{}'::jsonb
  ),
  COALESCE(
    (
      SELECT array_agg(ko.campo ORDER BY ko.campo)
      FROM kit_origens ko
      WHERE ko.organizacao_id = k.organizacao_id
    ),
    '{}'::text[]
  ),
  k.atualizado_por,
  'pessoa',
  k.atualizado_em
FROM kits_marca k;
