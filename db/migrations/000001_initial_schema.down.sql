-- #############################################################################
-- #  REVERSÃO DO SCHEMA INICIAL                                               #
-- #############################################################################
-- Este arquivo remove todas as estruturas criadas pela migration 'up'.
-- A ordem de remoção é a inversa da criação para respeitar as dependências.


-- Remove tabelas de junção e de suporte primeiro
DROP TABLE IF EXISTS transaction_tags;
DROP TABLE IF EXISTS budgets;

-- Remove a tabela central de transações
DROP TABLE IF EXISTS transactions;

-- Remove as tabelas primárias
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS accounts;

-- Remove a tabela fundamental por último
DROP TABLE IF EXISTS users;

-- OBS: Índices são removidos automaticamente quando as tabelas são deletadas.
-- Não é necessário um comando DROP INDEX explícito aqui.