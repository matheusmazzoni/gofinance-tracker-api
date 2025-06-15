-- #############################################################################
-- #  SCHEMA INICIAL DO BANCO DE DADOS - PROJETO DE FINANÇAS                   #
-- #############################################################################
-- Este arquivo define a estrutura completa de todas as tabelas necessárias
-- para a versão inicial da aplicação.

-- ========= Bloco 1: Entidades Fundamentais (Sem dependências) ==========

-- Tabela de Usuários: A raiz de todos os dados do sistema.
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========= Bloco 2: Entidades Principais (Dependentes de 'users') ==========

-- Tabela de Contas: Onde o dinheiro está (bancos, carteiras, etc.).
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'checking', 'savings', 'credit_card', etc.
    initial_balance DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

-- Tabela de Categorias: Usada para classificar despesas e receitas.
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'income', 'expense', 'transfer'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    -- Garante que um usuário não pode ter categorias duplicadas.
    UNIQUE(user_id, name)
);

-- Tabela de Tags: Para classificação flexível e contextual.
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (user_id, name)
);


-- ========= Bloco 3: Entidade Central (Transações) ==========

-- Tabela de Transações: O coração do sistema, ligando tudo.
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    date TIMESTAMPTZ NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'income', 'expense', 'transfer'

    -- Chaves Estrangeiras
    account_id INT NOT NULL,            -- Conta de origem
    destination_account_id INT,         -- Conta de destino (para transferências)
    category_id INT,                    -- Categoria (nulo para transferências)

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Definição das Relações
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_account FOREIGN KEY(account_id) REFERENCES accounts(id),
    CONSTRAINT fk_destination_account FOREIGN KEY(destination_account_id) REFERENCES accounts(id),
    CONSTRAINT fk_category FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE SET NULL
);


-- ========= Bloco 4: Tabelas de Junção e Entidades de Suporte ==========

-- Tabela de Junção (Muitos-para-Muitos): Conecta Transações e Tags.
CREATE TABLE transaction_tags (
    transaction_id INT NOT NULL,
    tag_id INT NOT NULL,
    PRIMARY KEY (transaction_id, tag_id),
    CONSTRAINT fk_transaction FOREIGN KEY(transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_tag FOREIGN KEY(tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Tabela de Orçamentos (Budgets): Define metas de gastos.
CREATE TABLE budgets (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    category_id INT NOT NULL,
    amount DECIMAL(12, 2) NOT NULL CHECK (amount >= 0),
    month INT NOT NULL CHECK (month >= 1 AND month <= 12),
    year INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_category FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE,
    -- Impede orçamentos duplicados para a mesma categoria no mesmo mês/ano.
    UNIQUE (user_id, category_id, year, month)
);


-- ========= Bloco 5: Índices para Otimização de Consultas ==========

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_categories_user_id ON categories(user_id);
CREATE INDEX idx_tags_user_id ON tags(user_id);
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_budgets_user_id_category_id ON budgets(user_id, category_id);