SELECT * FROM users WHERE id IN (SELECT user_id FROM payments) LIMIT 20;
