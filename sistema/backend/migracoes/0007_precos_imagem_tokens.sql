-- Preços de imagem por tokens: permite preço por token de entrada de imagem e saída em modelos de imagem.
-- Migração apenas aditiva; linhas existentes em ia_precos e ia_execucoes permanecem intocadas.
--
-- Como desfazer (somente após pg_dump do banco; requer ausência de linhas com unidade = 'token_entrada_imagem'):
--   ALTER TABLE ia_execucoes DROP COLUMN IF EXISTS preco_token_entrada_imagem;
--   ALTER TABLE ia_execucoes DROP COLUMN IF EXISTS tokens_entrada_imagem;
--   ALTER TABLE ia_precos DROP CONSTRAINT IF EXISTS ia_precos_unidade_check;
--   ALTER TABLE ia_precos ADD CONSTRAINT ia_precos_unidade_check CHECK (unidade IN ('token_entrada', 'token_saida', 'imagem'));

ALTER TABLE ia_precos DROP CONSTRAINT IF EXISTS ia_precos_unidade_check;
ALTER TABLE ia_precos ADD CONSTRAINT ia_precos_unidade_check
  CHECK (unidade IN ('token_entrada', 'token_entrada_imagem', 'token_saida', 'imagem'));

ALTER TABLE ia_execucoes
  ADD COLUMN tokens_entrada_imagem integer,
  ADD COLUMN preco_token_entrada_imagem numeric(18,8);
