UPDATE payments SET amount = 0 FROM users WHERE payments.user_id = users.id;
