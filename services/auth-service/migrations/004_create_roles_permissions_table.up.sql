-- 004_create_roles_permissions_table.up.sql
CREATE TYPE role_name AS ENUM ('org_owner', 'org_admin', 'manager', 'member', 'viewer');
CREATE TYPE permission_action AS ENUM ('create', 'read', 'update', 'delete', 'manage');

CREATE TABLE roles (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name role_name NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uniq_roles_tenant_id_name ON roles (tenant_id, name);

CREATE TABLE permissions (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    resource VARCHAR(100) NOT NULL,
    action permission_action NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uniq_permissions_resource_action ON permissions (resource, action);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_roles FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permissions FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    granted_by UUID,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id, tenant_id),
    CONSTRAINT fk_user_roles_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_roles FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_roles_tenant_id ON user_roles (tenant_id);
CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);

-- Seed default permissions
INSERT INTO permissions (id, resource, action, description) VALUES
    (gen_random_uuid(), 'users', 'create', 'Create users'),
    (gen_random_uuid(), 'users', 'read', 'View users'),
    (gen_random_uuid(), 'users', 'update', 'Update users'),
    (gen_random_uuid(), 'users', 'delete', 'Delete users'),
    (gen_random_uuid(), 'users', 'manage', 'Full user management'),
    (gen_random_uuid(), 'organizations', 'create', 'Create organizations'),
    (gen_random_uuid(), 'organizations', 'read', 'View organizations'),
    (gen_random_uuid(), 'organizations', 'update', 'Update organizations'),
    (gen_random_uuid(), 'organizations', 'delete', 'Delete organizations'),
    (gen_random_uuid(), 'organizations', 'manage', 'Full organization management'),
    (gen_random_uuid(), 'roles', 'create', 'Create roles'),
    (gen_random_uuid(), 'roles', 'read', 'View roles'),
    (gen_random_uuid(), 'roles', 'update', 'Update roles'),
    (gen_random_uuid(), 'roles', 'delete', 'Delete roles'),
    (gen_random_uuid(), 'roles', 'manage', 'Full role management'),
    (gen_random_uuid(), 'billing', 'read', 'View billing'),
    (gen_random_uuid(), 'billing', 'manage', 'Manage billing'),
    (gen_random_uuid(), 'modules', 'read', 'View modules'),
    (gen_random_uuid(), 'modules', 'manage', 'Manage modules'),
    (gen_random_uuid(), 'audit', 'read', 'View audit logs')
ON CONFLICT DO NOTHING;
