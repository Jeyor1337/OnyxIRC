# OnyxIRC

A secure IRC (Internet Relay Chat) client/server system with advanced security features.

## Features

- **Secure Communication**: RSA/AES-256 hybrid encryption for all data transmission
- **Strong Authentication**: SHA-256 password hashing with salts
- **Anti-Abuse System**: IP tracking and anomaly detection (blocks after 3 IP changes)
- **Admin Commands**: Comprehensive administrator command system
- **Decentralized Management**: Channel owners and moderators with delegated permissions
- **Intelligent Multithreading**: Efficient worker pool implementation
- **Multi-Language Support**: Server in Golang, reference client in Java

## Project Structure

```
OnyxIRC/
├── server/                 # Golang server
│   ├── cmd/
│   │   └── server/        # Main entry point
│   ├── internal/          # Internal packages
│   │   ├── config/        # Configuration management
│   │   ├── database/      # Database layer
│   │   ├── models/        # Data models
│   │   ├── auth/          # Authentication & encryption
│   │   ├── security/      # IP tracking, session management
│   │   ├── server/        # TCP server implementation
│   │   ├── protocol/      # IRC protocol
│   │   ├── admin/         # Admin commands
│   │   └── threadpool/    # Worker pool
│   └── configs/           # Configuration files
├── client-java/           # Java client
│   └── src/main/java/com/onyxirc/client/
│       ├── network/       # Connection handling
│       ├── security/      # Encryption
│       ├── ui/            # User interface
│       ├── protocol/      # Protocol implementation
│       └── models/        # Data models
├── sql/                   # Database schema
└── docs/                  # Documentation
```

## Architecture

### Security Design

1. **Password Security**
   - SHA-256 hashing with per-user random salts
   - Minimum password length enforcement
   - Optional special character requirements

2. **Data Encryption**
   - RSA (2048/4096-bit) for initial key exchange
   - AES-256-GCM for bulk data encryption
   - Per-session symmetric keys

3. **IP Tracking System**
   - Records all login attempts with IP addresses
   - Compares current IP with last known IP
   - Increments suspicion counter on mismatch
   - Locks account when suspicion > 3
   - Admin unlock capability

### Database Schema

- **users**: User accounts and credentials
- **user_ip_tracking**: Login history with IP addresses
- **user_security_status**: IP suspicion tracking and account locks
- **channels**: Chat channels
- **channel_members**: Channel membership and roles
- **messages**: Encrypted message storage
- **direct_messages**: Private messages
- **server_config**: Server configuration
- **admin_action_log**: Audit log for admin actions
- **user_bans**: Ban management
- **session_tokens**: Session management

## Getting Started

### Prerequisites

**Server:**
- Go 1.21 or higher
- MySQL 8.0 or higher

**Client:**
- Java 17 or higher
- Maven 3.8 or higher

### Installation

1. **Database Setup**
   ```bash
   mysql -u root -p < sql/schema.sql
   ```

2. **Server Setup**
   ```bash
   cd server
   go mod download
   go build -o server cmd/server/main.go
   ./server -config configs/server.yaml
   ```

3. **Client Setup**
   ```bash
   cd client-java
   mvn clean package
   java -jar target/onyxirc-client.jar
   ```

## Configuration

### Server Configuration

Edit `server/configs/server.yaml`:
- Server host/port settings
- Database connection details
- Security parameters (RSA/AES settings, IP tracking)
- Thread pool configuration
- Logging settings

### Client Configuration

Edit `client-java/src/main/resources/client.properties`:
- Server connection details
- Security settings
- Auto-reconnect behavior
- UI preferences

## Usage

### Client Commands

```
/register <username> <password>  - Register new account
/login <username> <password>     - Login to server
/join <channel>                  - Join a channel
/part <channel>                  - Leave a channel
/msg <user> <message>            - Send private message
/quit                            - Disconnect from server
```

### Admin Commands

```
/admin kick <username>           - Kick user from server
/admin ban <username> <duration> - Ban user (duration in seconds, 0 = permanent)
/admin unban <username>          - Remove ban
/admin unlock <username>         - Reset IP suspicion counter
/admin makeadmin <username>      - Grant admin privileges
/admin removeadmin <username>    - Revoke admin privileges
/admin broadcast <message>       - Send message to all users
/admin stats                     - Show server statistics
/admin shutdown                  - Graceful server shutdown
```

## Security Considerations

- Never commit private keys or certificates to version control
- Use environment variables for sensitive configuration
- Enable TLS/SSL in production
- Regularly update dependencies
- Implement rate limiting on authentication endpoints
- Monitor admin action logs for suspicious activity
- Regular security audits recommended

## Development

### Running Tests

**Server:**
```bash
cd server
go test ./...
```

**Client:**
```bash
cd client-java
mvn test
```

### Building

**Server:**
```bash
cd server
go build -o server cmd/server/main.go
```

**Client:**
```bash
cd client-java
mvn clean package
```

## License

MIT

## Contribution

no one

## Support

For issues and questions, please open an issue on the project repository.
