package server

import (
    "encoding/base64"
    "fmt"
    "log"
    "strings"
)

// handleRegister handles user registration
// Format: REGISTER <username> <password_hash>
func (c *Client) handleRegister(parts []string) error {
    if len(parts) < 3 {
        return fmt.Errorf("usage: REGISTER <username> <password_hash>")
    }

    username := parts[1]
    passwordHash := parts[2]

    // Note: Client should send SHA-256 hash of password
    // For simplicity, we'll treat it as the password itself
    user, err := c.server.authService.Register(username, passwordHash)
    if err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }

    c.Send(fmt.Sprintf(":%s NOTICE * :Registration successful. Please login.", c.server.config.Server.ServerName))
    log.Printf("User registered: %s (ID: %d)", user.Username, user.UserID)

    return nil
}

// handleLogin handles user login
// Format: LOGIN <username> <password_hash>
func (c *Client) handleLogin(parts []string) error {
    if len(parts) < 3 {
        return fmt.Errorf("usage: LOGIN <username> <password_hash>")
    }

    username := parts[1]
    passwordHash := parts[2]
    ipAddress := c.GetIPAddress()

    // Authenticate user
    user, err := c.server.authService.Login(username, passwordHash, ipAddress)
    if err != nil {
        return fmt.Errorf("login failed: %w", err)
    }

    // Check IP and track
    if err := c.server.ipTrackingService.CheckIPAndTrack(user.UserID, ipAddress); err != nil {
        return fmt.Errorf("login blocked: %w", err)
    }

    // Generate session key
    sessionKey, err := c.server.cryptoManager.GenerateSessionKey(c.server.config.Security.AESKeySize)
    if err != nil {
        return fmt.Errorf("failed to generate session key: %w", err)
    }

    // Create session
    session, err := c.server.sessionManager.CreateSession(user, ipAddress, sessionKey)
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }

    // Update client
    c.user = user
    c.authenticated = true
    c.session = session
    c.SessionID = session.SessionID
    c.sessionKey = sessionKey

    // Add to server
    c.server.AddClient(c)

    // Send success with session ID
    c.Send(fmt.Sprintf(":%s NOTICE %s :Login successful. Session ID: %s", c.server.config.Server.ServerName, username, session.SessionID))
    c.Send(fmt.Sprintf(":%s NOTICE %s :Please exchange encryption keys using KEYEXCHANGE", c.server.config.Server.ServerName, username))

    log.Printf("User logged in: %s (ID: %d) from %s", user.Username, user.UserID, ipAddress)

    return nil
}

// handleKeyExchange handles session key exchange
// Format: KEYEXCHANGE <encrypted_session_key_base64>
func (c *Client) handleKeyExchange(parts []string) error {
    if err := c.requireAuth(); err != nil {
        return err
    }

    if len(parts) < 2 {
        return fmt.Errorf("usage: KEYEXCHANGE <encrypted_session_key>")
    }

    // For simplicity, server generates the key and sends it encrypted to client
    // In a real implementation, this would be more sophisticated

    // Encrypt session key with client's public key (if we had it)
    // For now, we'll just send it encoded
    sessionKeyB64 := base64.StdEncoding.EncodeToString(c.sessionKey)

    c.Send(fmt.Sprintf("SESSIONKEY :%s", sessionKeyB64))
    c.Send(fmt.Sprintf(":%s NOTICE %s :Key exchange complete. All messages will be encrypted.", c.server.config.Server.ServerName, c.user.Username))

    return nil
}

// handleJoin handles joining a channel
// Format: JOIN <channel_name>
func (c *Client) handleJoin(parts []string) error {
    if err := c.requireAuth(); err != nil {
        return err
    }

    if len(parts) < 2 {
        return fmt.Errorf("usage: JOIN <channel>")
    }

    channelName := parts[1]
    return c.handleJoinComplete(channelName)
}

// handlePart handles leaving a channel
// Format: PART <channel_name>
func (c *Client) handlePart(parts []string) error {
    if err := c.requireAuth(); err != nil {
        return err
    }

    if len(parts) < 2 {
        return fmt.Errorf("usage: PART <channel>")
    }

    channelName := parts[1]
    return c.handlePartComplete(channelName)
}

// handlePrivMsg handles sending a message
// Format: PRIVMSG <target> :<message>
func (c *Client) handlePrivMsg(parts []string) error {
    if err := c.requireAuth(); err != nil {
        return err
    }

    if len(parts) < 3 {
        return fmt.Errorf("usage: PRIVMSG <target> :<message>")
    }

    target := parts[1]
    message := strings.Join(parts[2:], " ")

    if strings.HasPrefix(message, ":") {
        message = message[1:]
    }

    return c.handlePrivMsgComplete(target, message)
}

// handleQuit handles client disconnect
// Format: QUIT :<message>
func (c *Client) handleQuit(parts []string) error {
    message := "Client quit"
    if len(parts) > 1 {
        message = strings.Join(parts[1:], " ")
        if strings.HasPrefix(message, ":") {
            message = message[1:]
        }
    }

    c.Send(fmt.Sprintf("ERROR :Closing connection: %s", message))

    if c.authenticated {
        log.Printf("User %s quit: %s", c.user.Username, message)
    }

    c.Disconnect()

    return nil
}

// handlePing handles PING requests
// Format: PING :<server>
func (c *Client) handlePing(parts []string) error {
    if len(parts) < 2 {
        c.Send(fmt.Sprintf("PONG :%s", c.server.config.Server.ServerName))
    } else {
        c.Send(fmt.Sprintf("PONG :%s", parts[1]))
    }

    return nil
}
