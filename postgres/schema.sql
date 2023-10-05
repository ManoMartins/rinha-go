CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE OR REPLACE FUNCTION generate_searchable(_nome VARCHAR, _apelido VARCHAR, _stack JSON)
  RETURNS TEXT AS $$
  BEGIN
  RETURN _nome || _apelido || _stack;
  END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE TABLE IF NOT EXISTS pessoas (
  id uuid UNIQUE NOT NULL,
  apelido TEXT UNIQUE NOT NULL,
  nome TEXT NOT NULL,
  nascimento DATE NOT NULL,
  stack JSON,
  searchable text GENERATED ALWAYS AS (generate_searchable(nome, apelido, stack)) STORED
);

CREATE INDEX IF NOT EXISTS idx_pessoas_searchable ON public.pessoas USING gist (searchable public.gist_trgm_ops (siglen='64'));

CREATE UNIQUE INDEX IF NOT EXISTS pessoas_apelido_index ON public.pessoas USING btree (apelido);