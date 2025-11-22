package server

import (
    "fmt"
    "log"
    "net"
    "sync"
    "time"

    "github.com/onyxirc/server/internal/admin"
    "github.com/onyxirc/server/internal/auth"
    "github.com/onyxirc/server/internal/config"
    "github.com/onyxirc/server/internal/database"
    "github.com/onyxirc/server/internal/security"
)

type Server struct {
    config           *config.Config
    db               *database.DB
    listener         net.Listener
    clients          map[string]*Client 
    clientsMu        sync.RWMutex
    authService      *auth.AuthService
    adminService     *admin.AdminService
    ipTrackingService *security.IPTrackingService
    sessionManager   *security.SessionManager
    cryptoManager    *auth.CryptoManager
    shutdown         chan struct{}
    wg               sync.WaitGroup
}

func New(cfg *config.Config, db *database.DB) (*Server, error) {
    
    userRepo := database.NewUserRepository(db)
    securityRepo := database.NewSecurityRepository(db)
    adminRepo := database.NewAdminRepository(db)

    authService := auth.NewAuthService(
        userRepo,
        securityRepo,
        cfg.Security.PasswordMinLength,
        cfg.Security.PasswordRequireSpecial,
    )

    adminService := admin.NewAdminService(
        userRepo,
        adminRepo,
        securityRepo,
    )

    ipTrackingService := security.NewIPTrackingService(
        securityRepo,
        cfg.Security.MaxIPSuspicion,
        cfg.Security.EnableIPTracking,
    )

    sessionManager := security.NewSessionManager(
        time.Duration(cfg.Security.SessionTimeout) * time.Second,
    )

    cryptoManager, err := initializeCrypto(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize crypto: %w", err)
    }

    return &Server{
        config:            cfg,
        db:                db,
        clients:           make(map[string]*Client),
        authService:       authService,
        adminService:      adminService,
        ipTrackingService: ipTrackingService,
        sessionManager:    sessionManager,
        cryptoManager:     cryptoManager,
        shutdown:          make(chan struct{}),
    }, nil
}

func (s *Server) Start() error {
    address := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

    listener, err := net.Listen("tcp", address)
    if err != nil {
        return fmt.Errorf("failed to start listener: %w", err)
    }

    s.listener = listener
    log.Printf("Server listening on %s", address)

    for {
        select {
        case <-s.shutdown:
            return nil
        default:
            conn, err := s.listener.Accept()
            if err != nil {
                select {
                case <-s.shutdown:
                    return nil
                default:
                    log.Printf("Failed to accept connection: %v", err)
                    continue
                }
            }

            s.wg.Add(1)
            go s.handleConnection(conn)
        }
    }
}

func (s *Server) handleConnection(conn net.Conn) {
    defer s.wg.Done()

    client := NewClient(conn, s)

    log.Printf("New connection from %s", conn.RemoteAddr().String())

    client.Handle()

    log.Printf("Connection closed from %s", conn.RemoteAddr().String())
}

func (s *Server) AddClient(client *Client) {
    s.clientsMu.Lock()
    defer s.clientsMu.Unlock()

    s.clients[client.SessionID] = client
}

func (s *Server) RemoveClient(sessionID string) {
    s.clientsMu.Lock()
    defer s.clientsMu.Unlock()

    delete(s.clients, sessionID)
}

func (s *Server) GetClient(sessionID string) (*Client, bool) {
    s.clientsMu.RLock()
    defer s.clientsMu.RUnlock()

    client, exists := s.clients[sessionID]
    return client, exists
}

func (s *Server) BroadcastToChannel(channelID int64, message string, excludeSessionID string) {
    s.clientsMu.RLock()
    defer s.clientsMu.RUnlock()

    for _, client := range s.clients {
        if client.SessionID == excludeSessionID {
            continue
        }

        client.channelsMu.RLock()
        inChannel := false
        for _, cid := range client.channels {
            if cid == channelID {
                inChannel = true
                break
            }
        }
        client.channelsMu.RUnlock()

        if inChannel {
            client.Send(message)
        }
    }
}

func (s *Server) GetActiveClientCount() int {
    s.clientsMu.RLock()
    defer s.clientsMu.RUnlock()

    return len(s.clients)
}

func (s *Server) Shutdown() error {
    log.Println("Initiating graceful shutdown...")

    close(s.shutdown)

    if s.listener != nil {
        s.listener.Close()
    }

    s.clientsMu.Lock()
    for _, client := range s.clients {
        client.Send("ERROR :Server shutting down")
        client.Disconnect()
    }
    s.clientsMu.Unlock()

    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("All connections closed gracefully")
    case <-time.After(10 * time.Second):
        log.Println("Shutdown timeout reached, forcing exit")
    }

    if err := s.db.Close(); err != nil {
        log.Printf("Error closing database: %v", err)
    }

    log.Println("Server shutdown complete")
    return nil
}

func initializeCrypto(cfg *config.Config) (*auth.CryptoManager, error) {
    var keyPair *auth.RSAKeyPair

    privateKey, err := auth.LoadPrivateKeyFromFile(cfg.Security.RSAPrivateKeyPath)
    if err != nil {
        
        log.Printf("Generating new RSA key pair (%d bits)...", cfg.Security.RSAKeySize)
        keyPair, err = auth.GenerateRSAKeyPair(cfg.Security.RSAKeySize)
        if err != nil {
            return nil, fmt.Errorf("failed to generate RSA keys: %w", err)
        }

        if err := keyPair.SavePrivateKeyToFile(cfg.Security.RSAPrivateKeyPath); err != nil {
            return nil, fmt.Errorf("failed to save private key: %w", err)
        }

        if err := keyPair.SavePublicKeyToFile(cfg.Security.RSAPublicKeyPath); err != nil {
            return nil, fmt.Errorf("failed to save public key: %w", err)
        }

        log.Println("RSA keys generated and saved successfully")
    } else {
        
        publicKey, err := auth.LoadPublicKeyFromFile(cfg.Security.RSAPublicKeyPath)
        if err != nil {
            return nil, fmt.Errorf("failed to load public key: %w", err)
        }

        keyPair = &auth.RSAKeyPair{
            PrivateKey: privateKey,
            PublicKey:  publicKey,
        }

        log.Println("RSA keys loaded successfully")
    }

    cryptoManager := auth.NewCryptoManager(keyPair, cfg.Security.AESMode)

    return cryptoManager, nil
}
