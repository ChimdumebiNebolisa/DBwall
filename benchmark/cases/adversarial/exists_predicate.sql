DELETE FROM users WHERE EXISTS (SELECT 1 FROM approved_users WHERE approved_users.id = users.id);
