-- 004_create_roles_permissions_table.down.sql
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TYPE IF EXISTS permission_action;
DROP TYPE IF EXISTS role_name;
