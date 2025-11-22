package server

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
    "sync"
    "time"

    "github.com/onyxirc/server/internal/models"
    "github.com/onyxirc/server/internal/security"
)

type Client struct {
    conn         net.Conn
    server       *Server
    session      *security.Session
    SessionID    string
    user         *models.User
    authenticated bool
    sessionKey   []byte 
    channels     []int64
    channelsMu   sync.RWMutex
    writer       *bufio.Writer
    writerMu     sync.Mutex
    disconnect   chan struct{}
    once         sync.Once
}

func NewClient(conn net.Conn, server *Server) *Client {
    return &Client{
        conn:          conn,
        server:        server,
        authenticated: false,
        channels:      []int64{},
        writer:        bufio.NewWriter(conn),
        disconnect:    make(chan struct{}),
    }
}

func (c *Client) Handle() {
    defer c.Disconnect()

    c.conn.SetReadDeadline(time.Now().Add(c.server.config.Server.ReadTimeout))

    c.Send(fmt.Sprintf(":%s NOTICE * :Welcome to %s", c.server.config.Server.ServerName, c.server.config.Server.ServerName))

    publicKeyPEM, err := c.server.cryptoManager.GetPublicKeyPEM()
    if err != nil {
        log.Printf("Failed to get public key: %v", err)
        return
    }
    c.Send(fmt.Sprintf("PUBKEY :%s", string(publicKeyPEM)))

    scanner := bufio.NewScanner(c.conn)
    for scanner.Scan() {
        line := scanner.Text()
        line = strings.TrimSpace(line)

        if line == "" {
            continue
        }

        c.conn.SetReadDeadline(time.Now().Add(c.server.config.Server.ReadTimeout))

        if err := c.processCommand(line); err != nil {
            log.Printf("Error processing command: %v", err)
            c.Send(fmt.Sprintf("ERROR :%v", err))

            if strings.Contains(err.Error(), "account locked") {
                return
            }
        }
    }

    if err := scanner.Err(); err != nil {
        log.Printf("Scanner error: %v", err)
    }
}

func (c *Client) processCommand(line string) error {
    parts := strings.Fields(line)
    if len(parts) == 0 {
        return nil
    }

    command := strings.ToUpper(parts[0])

    switch command {
    case "REGISTER":
        return c.handleRegister(parts)
    case "LOGIN":
        return c.handleLogin(parts)
    case "KEYEXCHANGE":
        return c.handleKeyExchange(parts)
    case "JOIN":
        return c.handleJoin(parts)
    case "PART":
        return c.handlePart(parts)
    case "PRIVMSG":
        return c.handlePrivMsg(parts)
    case "QUIT":
        return c.handleQuit(parts)
    case "PING":
        return c.handlePing(parts)
    case "PONG":
        return nil 
    case "ADMIN":
        return c.handleAdminCommand(parts)
    default:
        return fmt.Errorf("unknown command: %s", command)
    }
}

func (c *Client) Send(message string) {
    c.writerMu.Lock()
    defer c.writerMu.Unlock()

    if _, err := c.writer.WriteString(message + "\r\n"); err != nil {
        log.Printf("Failed to write to client: %v", err)
        return
    }

    if err := c.writer.Flush(); err != nil {
        log.Printf("Failed to flush writer: %v", err)
    }
}

func (c *Client) Disconnect() {
    c.once.Do(func() {
        close(c.disconnect)

        if c.authenticated && c.SessionID != "" {
            
            c.server.RemoveClient(c.SessionID)

            c.server.sessionManager.DestroySession(c.SessionID)
        }

        c.conn.Close()
    })
}

func (c *Client) GetIPAddress() string {
    addr := c.conn.RemoteAddr().String()
    
    if idx := strings.LastIndex(addr, ":"); idx != -1 {
        return addr[:idx]
    }
    return addr
}

func (c *Client) requireAuth() error {
    if !c.authenticated {
        return fmt.Errorf("not authenticated")
    }
    return nil
}

func (c *Client) JoinChannel(channelID int64) {
    c.channelsMu.Lock()
    defer c.channelsMu.Unlock()

    for _, cid := range c.channels {
        if cid == channelID {
            return
        }
    }

    c.channels = append(c.channels, channelID)
}

func (c *Client) LeaveChannel(channelID int64) {
    c.channelsMu.Lock()
    defer c.channelsMu.Unlock()

    for i, cid := range c.channels {
        if cid == channelID {
            c.channels = append(c.channels[:i], c.channels[i+1:]...)
            return
        }
    }
}

func (c *Client) IsInChannel(channelID int64) bool {
    c.channelsMu.RLock()
    defer c.channelsMu.RUnlock()

    for _, cid := range c.channels {
        if cid == channelID {
            return true
        }
    }
    return false
}
