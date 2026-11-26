-- Ajustes 2A-1 (J4): permite excluir fornecedores preservando o histórico de execuções (ia_execucoes).
-- Migração aditiva: torna fornecedor_id e modelo_id opcionais e altera as chaves estrangeiras para ON DELETE SET NULL.
--
-- Como desfazer (somente após pg_dump do banco; requer ausência de nulos em fornecedor_id e modelo_id):
--   ALTER TABLE ia_execucoes DROP CONSTRAINT IF EXISTS ia_execucoes_fornecedor_id_fkey;
--   ALTER TABLE ia_execucoes DROP CONSTRAINT IF EXISTS ia_execucoes_modelo_id_fkey;
--   ALTER TABLE ia_execucoes
--     ADD CONSTRAINT ia_execucoes_fornecedor_id_fkey FOREIGN KEY (fornecedor_id) REFERENCES ia_fornecedores (id),
--     ADD CONSTRAINT ia_execucoes_modelo_id_fkey FOREIGN KEY (modelo_id) REFERENCES ia_modelos (id);
--   ALTER TABLE ia_execucoes ALTER COLUMN modelo_id SET NOT NULL;
--   ALTER TABLE ia_execucoes ALTER COLUMN fornecedor_id SET NOT NULL;

ALTER TABLE ia_execucoes ALTER COLUMN fornecedor_id DROP NOT NULL;
ALTER TABLE ia_execucoes ALTER COLUMN modelo_id DROP NOT NULL;

ALTER TABLE ia_execucoes DROP CONSTRAINT IF EXISTS ia_execucoes_fornecedor_id_fkey;
ALTER TABLE ia_execucoes DROP CONSTRAINT IF EXISTS ia_execucoes_modelo_id_fkey;

ALTER TABLE ia_execucoes
  ADD CONSTRAINT ia_execucoes_fornecedor_id_fkey FOREIGN KEY (fornecedor_id) REFERENCES ia_fornecedores (id) ON DELETE SET NULL,
  ADD CONSTRAINT ia_execucoes_modelo_id_fkey FOREIGN KEY (modelo_id) REFERENCES ia_modelos (id) ON DELETE SET NULL;
