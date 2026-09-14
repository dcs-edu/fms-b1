-- UNSET admin role for dev account
UPDATE users SET role = 'teacher', WHERE email = 'hajileayinomba@gmail.com'
