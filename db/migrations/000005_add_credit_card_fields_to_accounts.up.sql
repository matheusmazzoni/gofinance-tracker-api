-- These columns store the day of the month for closing and payment due.
-- They are NULLable because they only apply to accounts of type 'credit_card'.
ALTER TABLE accounts
ADD COLUMN statement_closing_day INT,
ADD COLUMN payment_due_day INT;