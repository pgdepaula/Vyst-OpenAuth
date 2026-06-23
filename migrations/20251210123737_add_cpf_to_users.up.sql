ALTER TABLE users ADD COLUMN cpf VARCHAR(11);
CREATE UNIQUE INDEX users_cpf_idx ON users (cpf) WHERE cpf IS NOT NULL AND cpf != '';
