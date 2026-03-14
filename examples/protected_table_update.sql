-- With policy that protects "users", this should trigger writes_to_protected_tables (warn)
UPDATE users SET role = 'viewer' WHERE id = 1;
