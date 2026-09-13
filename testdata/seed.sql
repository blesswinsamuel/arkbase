-- Seed schema and mock data for arkbase local testing

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    title VARCHAR(150) NOT NULL,
    price_cents INTEGER NOT NULL,
    inventory_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    total_cents INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'completed',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample users
INSERT INTO users (name, email) VALUES
('Alice Johnson', 'alice@example.com'),
('Bob Smith', 'bob@example.com'),
('Charlie Brown', 'charlie@example.com'),
('Diana Prince', 'diana@example.com'),
('Evan Wright', 'evan@example.com')
ON CONFLICT (email) DO NOTHING;

-- Insert sample products
INSERT INTO products (title, price_cents, inventory_count) VALUES
('Ergonomic Mechanical Keyboard', 12900, 45),
('Ultra-Wide Monitor 34"', 49900, 20),
('USB-C Docking Station', 8900, 60),
('Noise-Cancelling Headphones', 19900, 35),
('Standing Desk Mat', 3900, 100);

-- Insert sample orders
INSERT INTO orders (user_id, total_cents, status) VALUES
(1, 12900, 'completed'),
(1, 3900, 'completed'),
(2, 49900, 'completed'),
(3, 8900, 'shipped'),
(4, 19900, 'completed'),
(5, 16800, 'processing');
