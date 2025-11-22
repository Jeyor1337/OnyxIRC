# OnyxIRC Deployment Guide

This guide covers deploying OnyxIRC server and client in various environments.

## Prerequisites

- Docker & Docker Compose (recommended)
- OR: Go 1.21+, MySQL 8.0+, Java 17+ (for manual installation)

## Quick Start with Docker Compose

### 1. Clone the Repository

```bash
git clone <repository-url>
cd OnyxIRC
```

### 2. Configure Environment

Edit `server/configs/server.yaml` and update:
- Database credentials (or use environment variables)
- Server host/port settings
- Security parameters

### 3. Start All Services

```bash
docker-compose up -d
```

This will start:
- MySQL database (port 3306)
- OnyxIRC server (port 6667)

### 4. View Logs

```bash
# Server logs
docker-compose logs -f server

# Database logs
docker-compose logs -f mysql
```

### 5. Create First Admin User

Connect with a client and register:
```
/register admin yourpassword
/login admin yourpassword
```

Then manually grant admin privileges in MySQL:
```bash
docker-compose exec mysql mysql -u root -p
```

```sql
USE onyxirc;
UPDATE users SET is_admin = TRUE WHERE username = 'admin';
```

### 6. Test with Client (Optional)

```bash
docker-compose --profile client up client
```

## Manual Installation

### Server Installation

#### 1. Install Dependencies

```bash
cd server
go mod download
```

#### 2. Set Up MySQL

```bash
mysql -u root -p < ../sql/schema.sql
```

#### 3. Configure Server

Edit `server/configs/server.yaml`:
```yaml
database:
  host: "localhost"
  port: 3306
  name: "onyxirc"
  user: "irc_user"
  password: "yourpassword"

server:
  host: "0.0.0.0"
  port: 6667

security:
  rsa_key_size: 2048
  max_ip_suspicion: 3
```

#### 4. Build and Run

```bash
go build -o server cmd/server/main.go
./server -config configs/server.yaml
```

### Client Installation

#### 1. Build with Maven

```bash
cd client-java
mvn clean package
```

#### 2. Configure Client

Edit `src/main/resources/client.properties`:
```properties
server.host=localhost
server.port=6667
```

#### 3. Run Client

```bash
java -jar target/onyxirc-client.jar
```

## Production Deployment

### Security Hardening

1. **Use Strong Passwords**
   ```yaml
   database:
     password: "${DB_PASSWORD}"  # Use environment variable
   ```

2. **Enable TLS/SSL**
   - Configure TLS certificates
   - Use port 6697 for TLS connections

3. **Firewall Configuration**
   ```bash
   # Allow only IRC port
   ufw allow 6667/tcp
   ufw enable
   ```

4. **RSA Key Management**
   - Generate strong RSA keys (4096-bit)
   - Protect private keys (chmod 600)
   - Never commit keys to version control

### Scaling

#### Horizontal Scaling

For multiple server instances:
1. Use load balancer (nginx, HAProxy)
2. Shared database for all instances
3. Redis for session storage (future enhancement)

#### Database Optimization

```sql
-- Add indexes for performance
CREATE INDEX idx_user_active ON users(is_active, last_login_time);
CREATE INDEX idx_channel_active ON channels(is_private);
```

### Monitoring

#### Health Checks

```bash
# Check server is running
nc -zv localhost 6667

# Check database connection
docker-compose exec mysql mysqladmin ping
```

#### Log Monitoring

```bash
# Server logs
tail -f server/logs/server.log

# Database slow queries
docker-compose exec mysql mysql -e "SHOW VARIABLES LIKE 'slow_query_log';"
```

### Backup & Recovery

#### Database Backup

```bash
# Backup
docker-compose exec mysql mysqldump -u root -p onyxirc > backup.sql

# Restore
docker-compose exec -T mysql mysql -u root -p onyxirc < backup.sql
```

#### Configuration Backup

```bash
# Backup server keys and config
tar -czf onyxirc-backup.tar.gz \
    server/configs/ \
    server/keys/ \
    sql/schema.sql
```

## Troubleshooting

### Server won't start

```bash
# Check logs
docker-compose logs server

# Common issues:
# 1. Database not ready - wait for MySQL healthcheck
# 2. Port already in use - check with: lsof -i :6667
# 3. Permission issues with keys directory
```

### Database connection errors

```bash
# Test database connection
docker-compose exec mysql mysql -u irc_user -p onyxirc

# Check database is healthy
docker-compose ps
```

### Client can't connect

```bash
# Test server is listening
telnet localhost 6667

# Check firewall
ufw status

# Verify server config
cat server/configs/server.yaml
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| DB_PASSWORD | Database password | changeme |
| DB_HOST | Database host | localhost |
| DB_PORT | Database port | 3306 |

## Updating

### Rolling Update

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose build
docker-compose up -d

# Run database migrations if needed
docker-compose exec server ./server -migrate
```

## Performance Tuning

### MySQL Optimization

```ini
# /etc/mysql/my.cnf
[mysqld]
max_connections = 500
innodb_buffer_pool_size = 1G
query_cache_size = 64M
```

### Server Configuration

```yaml
threadpool:
  worker_count: 20      # Adjust based on CPU cores
  max_workers: 100
  queue_size: 2000

server:
  max_connections: 1000
  read_timeout: 60s
  write_timeout: 60s
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/onyxirc/issues
- Documentation: /docs
