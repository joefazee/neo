DROP TABLE IF EXISTS token_blacklist;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS settlements;
DROP TABLE IF EXISTS bets;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS market_outcomes;
DROP TABLE IF EXISTS markets;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS countries;

-- Drop UUID extension
DROP EXTENSION IF EXISTS "uuid-ossp";
