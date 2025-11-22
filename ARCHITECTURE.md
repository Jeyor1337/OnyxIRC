# OnyxIRC Architecture Documentation

## System Overview

OnyxIRC is a secure IRC (Internet Relay Chat) system with advanced security features including RSA/AES-256 encryption, IP-based anomaly detection, and comprehensive admin controls.

## High-Level Architecture

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────┐
│  Java Client    │ ◄─────► │  Golang Server   │ ◄─────► │   MySQL DB  │
│  (Multi-lang)   │   TCP   │  (Port 6667)     │  JDBC   │             │
└─────────────────┘  6667   └──────────────────┘         └─────────────┘
       │                             │
       │ RSA/AES                     │
       │ Encryption                  │
       └─────────────────────────────┘
```

## Server Architecture

### Layer Structure

```
┌──────────────────────────────────────────────────────┐
│                  Network Layer                        │
│  - TCP Server (port 6667)                            │
│  - Connection Handler                                 │
│  - Client Manager                                     │
└──────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────┐
│                 Protocol Layer                        │
│  - IRC Protocol Parser                               │
│  - Command Router                                     │
│  - Message Handler                                    │
└──────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────┐
│                 Business Logic Layer                  │
│  - Auth Service    - Admin Service                   │
│  - Channel Manager - Security Service                │
│  - IP Tracking     - Session Manager                 │
└──────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────┐
│                  Data Access Layer                    │
│  - User Repository      - Channel Repository         │
│  - Security Repository  - Admin Repository           │
└──────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────┐
│                   MySQL Database                      │
│  - Users   - Channels   - Messages   - Admin Logs    │
└──────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Network Layer (`server/internal/server/`)

**Files:**
- `server.go` - Main server struct, connection acceptance
- `client.go` - Client connection handling
- `handlers.go` - Command handlers

**Responsibilities:**
- Accept TCP connections
- Manage client lifecycle
- Handle graceful shutdown
- Broadcast messages

#### 2. Security Layer (`server/internal/auth/`, `server/internal/security/`)

**Files:**
- `auth/hashing.go` - SHA-256 password hashing
- `auth/rsa.go` - RSA key generation and encryption
- `auth/aes.go` - AES-256 encryption (GCM/CBC)
- `auth/encryption.go` - Hybrid encryption manager
- `security/ip_tracking.go` - IP-based anomaly detection
- `security/session.go` - Session management

**Key Features:**
- **Password Security:** SHA-256 with per-user salts, constant-time comparison
- **Encryption:** RSA-2048/4096 for key exchange, AES-256-GCM for data
- **IP Tracking:** Automatic suspicion counter, account locking after 3 IP changes
- **Sessions:** In-memory session store with auto-cleanup

#### 3. Business Logic Layer

##### Authentication Service (`server/internal/auth/auth.go`)

```go
type AuthService struct {
    userRepo     *UserRepository
    securityRepo *SecurityRepository
    // ...
}

// Core methods:
- Register(username, password) -> User
- Login(username, password, ip) -> User
- VerifyPassword(password, salt, hash) -> bool
```

##### Admin Service (`server/internal/admin/commands.go`)

```go
type AdminService struct {
    userRepo     *UserRepository
    adminRepo    *AdminRepository
    securityRepo *SecurityRepository
}

// Admin operations:
- KickUser(adminID, username, reason)
- BanUser(adminID, username, reason, duration)
- UnlockAccount(adminID, username)
- MakeAdmin(adminID, targetUserID)
- GetServerStats(adminID)
```

#### 4. Data Access Layer (`server/internal/database/`)

**Repositories:**
- `user_repository.go` - User CRUD operations
- `security_repository.go` - IP tracking, account locks
- `channel_repository.go` - Channel management
- `admin_repository.go` - Admin actions, bans

**Connection Pool:**
- Max Open Connections: 100 (configurable)
- Max Idle Connections: 10
- Connection Lifetime: 3600s
- Health checks every 5s

## Database Schema

### Core Tables

```sql
users
├── user_id (PK)
├── username (UNIQUE)
├── password_hash (SHA-256)
├── password_salt
├── is_active
├── is_admin
└── last_login_time

user_security_status
├── user_id (PK, FK)
├── last_known_ip
├── ip_suspicion_count
├── account_locked
├── lock_reason
└── locked_by (FK)

user_ip_tracking
├── tracking_id (PK)
├── user_id (FK)
├── ip_address
├── login_timestamp
└── is_successful

channels
├── channel_id (PK)
├── channel_name (UNIQUE)
├── created_by (FK)
├── topic
└── is_private

channel_members
├── membership_id (PK)
├── channel_id (FK)
├── user_id (FK)
├── role (member/moderator/owner)
└── is_muted

admin_action_log
├── log_id (PK)
├── admin_id (FK)
├── action_type
├── target_user_id (FK)
└── performed_at
```

### Relationships

```
users (1) ────── (N) user_ip_tracking
  │
  ├─────── (1) user_security_status
  │
  ├─────── (N) channel_members
  │
  └─────── (N) admin_action_log

channels (1) ── (N) channel_members
         (1) ── (N) messages
```

## Security Architecture

### Encryption Flow

```
1. Server Startup:
   ├─ Generate/Load RSA key pair (2048/4096-bit)
   └─ Store private key securely (chmod 600)

2. Client Connection:
   ├─ Server sends RSA public key (PEM format)
   └─ Client receives and stores public key

3. Key Exchange:
   ├─ Server generates AES session key (256-bit)
   ├─ Server encrypts session key with client's expectation
   └─ Session key used for all subsequent messages

4. Message Encryption:
   ├─ AES-256-GCM for authenticated encryption
   ├─ Random IV/nonce per message
   └─ Integrity verification with GCM tag
```

### IP Tracking Algorithm

```python
def check_ip_and_track(user_id, current_ip):
    status = get_security_status(user_id)

    if status.account_locked:
        raise AccountLockedException()

    if status.last_known_ip is None:
        # First login
        update_last_known_ip(user_id, current_ip)
        return

    if status.last_known_ip != current_ip:
        # IP changed
        new_count = increment_suspicion(user_id)

        if new_count > MAX_SUSPICION (3):
            lock_account(user_id, "Too many IP changes")
            raise AccountLockedException()

        update_last_known_ip(user_id, current_ip)
```

## Client Architecture

### Java Client Structure

```
com.onyxirc.client
├── Main.java                    # Entry point
├── config/
│   └── ClientConfig.java        # Configuration loader
├── network/
│   ├── Connection.java          # Socket management
│   └── MessageHandler.java      # Message processing
├── security/
│   ├── Hashing.java            # SHA-256
│   └── Encryption.java         # RSA/AES
└── ui/
    └── ConsoleUI.java           # User interface
```

### Client-Server Protocol

```
1. Connection Established:
   SERVER → CLIENT: PUBKEY :<RSA_PUBLIC_KEY_PEM>

2. Registration:
   CLIENT → SERVER: REGISTER <username> <password_hash>
   SERVER → CLIENT: NOTICE :Registration successful

3. Login:
   CLIENT → SERVER: LOGIN <username> <password_hash>
   SERVER → CLIENT: NOTICE :Login successful. Session ID: <sid>

4. Key Exchange:
   SERVER → CLIENT: SESSIONKEY :<base64_aes_key>

5. Channel Operations:
   CLIENT → SERVER: JOIN #channel
   SERVER → CLIENT: :user!user@ip JOIN :#channel
   SERVER → CLIENT: :server 353 user = #channel :@owner +mod user

6. Messages:
   CLIENT → SERVER: PRIVMSG #channel :Hello world
   SERVER → CHANNEL: :user!user@ip PRIVMSG #channel :Hello world
```

## Concurrency & Threading

### Worker Pool Architecture

```
┌─────────────────────────────────────┐
│       Job Queue (Channel)           │
│     Capacity: 1000 (configurable)   │
└──────────────┬──────────────────────┘
               │
       ┌───────┴────────┐
       │                │
   ┌───▼───┐       ┌───▼───┐
   │Worker1│  ...  │WorkerN│
   └───────┘       └───────┘

Worker Scaling:
- Min Workers: 10
- Max Workers: 100
- Auto-scale when queue > 50% full
- Idle timeout: 60s
```

### Goroutine Management

```
Per Client:
├─ Connection Handler Goroutine
├─ Read Goroutine (scanner.Scan loop)
└─ Write Goroutine (message queue processor)

Server Level:
├─ Accept Loop Goroutine
├─ Session Cleanup Goroutine (every 1 min)
└─ Worker Pool Goroutines (10-100)
```

## Performance Characteristics

### Benchmarks (Estimated)

- **Connections**: 1000+ concurrent clients
- **Messages/sec**: 10,000+
- **Latency**: <10ms (local network)
- **Memory**: ~50MB base + ~1MB per 100 clients

### Scalability Limits

- **Single Instance**: ~10,000 clients
- **Database**: Bottleneck at ~50,000 concurrent ops/sec
- **Network**: Limited by bandwidth (assuming 1Gbps)

## Configuration Management

### Server Configuration Hierarchy

```
1. Default values (in code)
2. server.yaml file
3. Environment variables (highest priority)

Example:
database.password: "${DB_PASSWORD}"  # Reads from env
```

### Key Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| server.max_connections | 1000 | Max concurrent clients |
| security.max_ip_suspicion | 3 | IP changes before lock |
| security.session_timeout | 3600 | Session expiry (seconds) |
| threadpool.worker_count | 10 | Initial worker threads |
| database.max_open_conns | 100 | DB connection pool size |

## Deployment Topologies

### Single-Server Deployment

```
┌────────────────────────────┐
│       Load Balancer        │ (optional)
└────────────┬───────────────┘
             │
┌────────────▼───────────────┐
│     OnyxIRC Server         │
│  ┌──────────────────────┐  │
│  │  Application         │  │
│  └──────────┬───────────┘  │
│  ┌──────────▼───────────┐  │
│  │  MySQL Database      │  │
│  └──────────────────────┘  │
└────────────────────────────┘
```

### Multi-Server Deployment (Future)

```
     ┌────────────────┐
     │ Load Balancer  │
     └───┬────────┬───┘
         │        │
    ┌────▼──┐  ┌─▼─────┐
    │Server1│  │Server2│
    └───┬───┘  └───┬───┘
        │          │
    ┌───▼──────────▼────┐
    │ Shared MySQL DB   │
    └───────────────────┘
```

## Error Handling Strategy

### Error Levels

1. **Critical**: Server crash/restart
   - Database connection lost
   - Port binding failure

2. **Error**: Operation failed, logged
   - Authentication failure
   - Database query error

3. **Warning**: Unexpected but handled
   - Invalid command format
   - Client disconnect

### Recovery Mechanisms

- **Database**: Auto-reconnect with exponential backoff
- **Clients**: Graceful disconnect, session cleanup
- **Worker Pool**: Failed jobs logged, worker continues

## Future Enhancements

1. **Redis Integration**: Distributed session storage
2. **Message Persistence**: Long-term message history
3. **End-to-End Encryption**: Client-to-client encryption
4. **WebSocket Support**: Browser-based clients
5. **File Transfer**: Encrypted file sharing
6. **Voice/Video**: Real-time communication
7. **Mobile Clients**: iOS/Android apps
8. **Clustering**: Multi-server coordination
9. **Analytics**: Usage statistics and reporting
10. **API Gateway**: REST/GraphQL API

## Code Organization

```
OnyxIRC/
├── server/                  # Golang server
│   ├── cmd/server/         # Entry point
│   ├── internal/           # Private packages
│   │   ├── admin/         # Admin commands
│   │   ├── auth/          # Authentication
│   │   ├── config/        # Configuration
│   │   ├── database/      # Data access
│   │   ├── models/        # Data models
│   │   ├── security/      # Security services
│   │   ├── server/        # Server logic
│   │   └── threadpool/    # Worker pool
│   └── configs/           # Config files
├── client-java/           # Java client
│   └── src/main/java/     # Source code
├── sql/                   # Database schema
└── docs/                  # Documentation
```

## Testing Strategy

### Unit Tests
- Encryption/decryption
- Password hashing
- IP tracking logic
- Repository operations

### Integration Tests
- End-to-end auth flow
- Channel operations
- Admin commands

### Load Tests
- Concurrent connections
- Message throughput
- Database performance

## Monitoring & Observability

### Metrics to Track
- Active connections
- Messages per second
- Database query time
- Session count
- Error rate
- CPU/Memory usage

### Logging
- Structured logging (JSON)
- Log levels: DEBUG, INFO, WARN, ERROR
- Rotation: 100MB per file, 5 backups
