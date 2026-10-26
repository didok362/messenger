-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(50),
    type VARCHAR(20) NOT NULL CHECK (type IN ('direct', 'group')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE chat_members(
    chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(chat_id, user_id)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID REFERENCES users(id) ON DELETE SET NULL,
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    content VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_members_user_id ON chat_members(user_id);
CREATE INDEX idx_messages_chat_id_created_at ON messages(chat_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS messages CASCADE; 
DROP TABLE IF EXISTS chat_members CASCADE;
DROP TABLE IF EXISTS chats CASCADE;
DROP TABLE IF EXISTS users CASCADE;

