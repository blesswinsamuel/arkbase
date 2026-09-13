-- Sample database seed for arkbase local development & testing

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    price_cents INTEGER NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total_cents INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed users
INSERT INTO users (name, email) VALUES
    ('Alice Johnson', 'alice@example.com'),
    ('Bob Smith', 'bob@example.com'),
    ('Charlie Brown', 'charlie@example.com'),
    ('Diana Prince', 'diana@example.com'),
    ('Evan Wright', 'evan@example.com'),
    ('Fiona Gallagher', 'fiona@example.com'),
    ('George Clark', 'george@example.com'),
    ('Hannah Abbott', 'hannah@example.com'),
    ('Ian Malcolm', 'ian@example.com'),
    ('Julia Roberts', 'julia@example.com')
ON CONFLICT (email) DO NOTHING;

-- Seed products
INSERT INTO products (sku, name, price_cents, stock) VALUES
    ('PROD-001', 'Ergonomic Mechanical Keyboard', 12999, 45),
    ('PROD-002', 'Ultra-Wide 34-inch Monitor', 49999, 12),
    ('PROD-003', 'Wireless Noise-Cancelling Headphones', 19999, 80),
    ('PROD-004', 'USB-C Aluminum Multiport Dock', 7999, 120),
    ('PROD-005', 'Vertical Ergonomic Mouse', 5999, 65),
    ('PROD-006', 'High-Speed NVMe SSD 2TB', 14999, 30),
    ('PROD-007', 'Desk Mat Felt & Rubber Base', 2499, 200),
    ('PROD-008', '4K High-Res Streaming Webcam', 11999, 40)
ON CONFLICT (sku) DO NOTHING;

-- Seed orders
INSERT INTO orders (user_id, total_cents, status) VALUES
    (1, 12999, 'completed'),
    (1, 5999, 'completed'),
    (2, 49999, 'shipped'),
    (3, 19999, 'completed'),
    (4, 7999, 'processing'),
    (5, 14999, 'completed'),
    (6, 2499, 'completed'),
    (7, 11999, 'shipped'),
    (8, 12999, 'completed'),
    (9, 19999, 'completed');
