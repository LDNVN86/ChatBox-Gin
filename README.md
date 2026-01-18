# ChatBox Backend API

A high-performance, scalable chat platform backend built with **Go** and **Gin** framework. Supports multi-channel messaging (Facebook Messenger, Zalo, Web), automated bot responses, and real-time communication.

## 🚀 Features

### Core Features

- **Multi-Channel Support**: Facebook Messenger, Zalo, Web Widget (extensible)
- **Real-time Messaging**: WebSocket via Centrifugo for instant message delivery
- **Automated Bot**: Rule-based auto-responses with keyword matching
- **Multi-Workspace**: Isolated workspaces for different businesses/teams
- **Agent Management**: Role-based access control (Owner, Admin, Agent)

### Technical Features

- **JWT Authentication**: Secure httpOnly cookie-based auth with refresh tokens
- **CSRF Protection**: Double-submit cookie pattern
- **Pagination**: Efficient data loading for large datasets
- **Soft Delete**: Safe data recovery for rules and conversations
- **Structured Logging**: Production-ready logging with Zap

## 🛠️ Tech Stack

| Component | Technology           |
| --------- | -------------------- |
| Language  | Go 1.21+             |
| Framework | Gin                  |
| Database  | PostgreSQL 15+       |
| ORM       | GORM                 |
| Real-time | Centrifugo           |
| Logging   | Zap                  |
| Auth      | JWT (golang-jwt/jwt) |
| Password  | bcrypt               |

## 📁 Project Structure

```
BACKEND-GIN/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── auth/                # JWT token management
│   ├── bot/                 # Rule engine & auto-responder
│   ├── channel/             # Channel implementations (Facebook, Mock)
│   ├── config/              # Configuration management
│   ├── database/            # Database connection
│   ├── dto/                 # Data Transfer Objects
│   ├── errors/              # Custom error types
│   ├── handlers/            # HTTP handlers (controllers)
│   ├── middleware/          # Auth, CORS, Logging, CSRF
│   ├── models/              # Database models (GORM)
│   ├── realtime/            # Centrifugo client
│   ├── repositories/        # Data access layer
│   └── services/            # Business logic
├── .env.example             # Environment variables template
├── go.mod                   # Go module definition
└── go.sum                   # Dependencies lock file
```

## 🔧 Installation

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Centrifugo (for real-time features)

### Setup

1. **Clone repository**

```bash
git clone https://github.com/yourusername/chatbox.git
cd chatbox/ChatBox-Gin/BACKEND-GIN
```

2. **Copy environment file**

```bash
cp .env.example .env
```

3. **Configure environment variables**

```env
# Server
PORT=8081
GIN_MODE=debug

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=chatbox-gin

# JWT
JWT_SECRET=your-super-secret-key
JWT_EXPIRES_IN=3600

# Centrifugo
CENTRIFUGO_URL=http://localhost:8000
CENTRIFUGO_API_KEY=your-api-key

# Facebook (optional)
FB_VERIFY_TOKEN=your-verify-token
FB_APP_SECRET=your-app-secret
```

4. **Run database migrations**

```bash
# Migrations run automatically on startup via GORM AutoMigrate
```

5. **Start server**

```bash
go run ./cmd/server
```

## 📚 API Documentation

### Authentication

| Method | Endpoint               | Description          |
| ------ | ---------------------- | -------------------- |
| POST   | `/api/v1/auth/login`   | User login           |
| POST   | `/api/v1/auth/refresh` | Refresh access token |
| GET    | `/api/v1/auth/me`      | Get current user     |
| POST   | `/api/v1/auth/logout`  | Logout               |

### Conversations

| Method | Endpoint                             | Description                    |
| ------ | ------------------------------------ | ------------------------------ |
| GET    | `/api/v1/conversations`              | List conversations (paginated) |
| GET    | `/api/v1/conversations/:id`          | Get conversation details       |
| PATCH  | `/api/v1/conversations/:id`          | Update conversation status     |
| GET    | `/api/v1/conversations/:id/messages` | List messages (paginated)      |
| POST   | `/api/v1/conversations/:id/messages` | Send message as agent          |
| POST   | `/api/v1/conversations/:id/bot`      | Toggle bot on/off              |

### Rules (Bot Automation)

| Method | Endpoint                    | Description                 |
| ------ | --------------------------- | --------------------------- |
| GET    | `/api/v1/rules`             | List active rules           |
| GET    | `/api/v1/rules/trash`       | List deleted rules          |
| GET    | `/api/v1/rules/:id`         | Get rule details            |
| POST   | `/api/v1/rules`             | Create rule                 |
| PUT    | `/api/v1/rules/:id`         | Update rule                 |
| DELETE | `/api/v1/rules/:id`         | Soft delete rule            |
| PATCH  | `/api/v1/rules/:id/toggle`  | Toggle rule active/inactive |
| POST   | `/api/v1/rules/:id/restore` | Restore deleted rule        |

### Webhooks

| Method | Endpoint                   | Description              |
| ------ | -------------------------- | ------------------------ |
| GET    | `/api/v1/webhook/facebook` | Facebook verification    |
| POST   | `/api/v1/webhook/facebook` | Facebook message webhook |

### Mock (Development)

| Method | Endpoint                | Description               |
| ------ | ----------------------- | ------------------------- |
| POST   | `/api/v1/mock/inbound`  | Simulate incoming message |
| POST   | `/api/v1/mock/outbound` | Simulate outgoing message |
| GET    | `/api/v1/mock/sent`     | Get sent messages         |
| DELETE | `/api/v1/mock/sent`     | Clear sent messages       |

## 🔐 Authentication Flow

```
1. Login: POST /auth/login
   ↓ Sets httpOnly cookies: access_token, refresh_token, csrf_token

2. API Request: Include cookies automatically
   ↓ For mutations: Include X-CSRF-Token header

3. Token Expired: 401 Unauthorized
   ↓ POST /auth/refresh (uses refresh_token cookie)

4. Logout: POST /auth/logout
   ↓ Clears all auth cookies
```

## 🤖 Bot Rule Engine

### Rule Types

| Type          | Description                          |
| ------------- | ------------------------------------ |
| `keyword`     | Match keywords in message            |
| `time_window` | Active during specific hours         |
| `fallback`    | Default response when no rules match |

### Example Rule

```json
{
  "name": "Greeting",
  "trigger_type": "keyword",
  "trigger_config": {
    "keywords": ["hello", "hi", "xin chào"],
    "match_type": "contains"
  },
  "response_type": "text",
  "response_config": {
    "text": "Xin chào! Tôi có thể giúp gì cho bạn?"
  },
  "priority": 10,
  "is_active": true
}
```

## 🌐 Real-time Events

Messages are pushed via Centrifugo to channel: `chat:workspace_{workspace_id}`

### Event Types

```typescript
// New message
{
  "type": "new_message",
  "message_id": "uuid",
  "conversation_id": "uuid",
  "direction": "in" | "out",
  "sender_type": "customer" | "bot" | "agent",
  "content": "Hello",
  "created_at": "2024-01-01T00:00:00Z"
}

// Conversation update
{
  "type": "conversation_update",
  "conversation_id": "uuid",
  "status": "open" | "pending" | "closed"
}
```

## 🗃️ Database Schema

### Core Tables

- **workspaces**: Business/team isolation
- **users**: Agents, admins, owners
- **channel_accounts**: Connected channels (Facebook, Zalo)
- **participants**: End customers
- **conversations**: Chat threads
- **messages**: Individual messages
- **rules**: Bot automation rules

## 🧪 Database Seeding

To quickly set up a development environment with a test workspace and user, you can run the following SQL commands in your PostgreSQL database:

```sql
-- 1. Create a new workspace
INSERT INTO workspaces (id, name, created_at, updated_at)
VALUES (
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  'Workspace 2',
  NOW(),
  NOW()
);

-- 2. Create an admin user for the new workspace
-- Password: 123456 (bcrypt hash)
INSERT INTO users (id, workspace_id, email, password_hash, name, role, is_active, created_at, updated_at)
VALUES (
  gen_random_uuid(),
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  'admin2@example.com',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.QKXK5TdLnBv1x4NYie',
  'Admin Workspace 2',
  'admin',
  true,
  NOW(),
  NOW()
);

-- 3. Connect a Facebook Channel Account
INSERT INTO channel_accounts (
  id,
  workspace_id,
  channel_type,
  name,
  channel_id,
  credentials,
  is_active,
  created_at,
  updated_at
) VALUES (
  gen_random_uuid(),
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
  'facebook',
  'FB Page Test',
  'ID FanPage',
  '{"page_access_token": "Key Access_Token From FB Dev"}',
  true,
  NOW(),
  NOW()
);
```

## 🧪 Development

### Run Tests

```bash
go test ./...
```

### Build Binary

```bash
go build -o chatbox-server ./cmd/server
```

### Docker (Optional)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/server /server
EXPOSE 8081
CMD ["/server"]
```

## 📝 Environment Variables

| Variable           | Required | Default   | Description                   |
| ------------------ | -------- | --------- | ----------------------------- |
| PORT               | No       | 8081      | Server port                   |
| GIN_MODE           | No       | debug     | debug/release                 |
| DB_HOST            | Yes      | localhost | PostgreSQL host               |
| DB_PORT            | No       | 5432      | PostgreSQL port               |
| DB_USER            | Yes      | -         | Database user                 |
| DB_PASSWORD        | Yes      | -         | Database password             |
| DB_NAME            | Yes      | -         | Database name                 |
| JWT_SECRET         | Yes      | -         | JWT signing key               |
| JWT_EXPIRES_IN     | No       | 3600      | Token expiry (seconds)        |
| CENTRIFUGO_URL     | Yes      | -         | Centrifugo server URL         |
| CENTRIFUGO_API_KEY | Yes      | -         | Centrifugo API key            |
| FB_VERIFY_TOKEN    | No       | -         | Facebook webhook verify token |
| FB_APP_SECRET      | No       | -         | Facebook app secret           |

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License.

---

Built with ❤️ using Go & Gin
