-- Should allow: UPDATE with WHERE
UPDATE users SET last_login = NOW() WHERE id = 42;
