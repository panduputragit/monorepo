INSERT INTO admins (email, password_hash) VALUES
  ('admin@gym.com', '$2y$10$3cxXd3k6u3av847qmv1daur02Ki2u1tQEVR0zNwKWMN4VJWgdv5v2')
ON CONFLICT (email) DO NOTHING;
