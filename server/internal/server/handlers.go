package server

import (
    "encoding/base64"
    "fmt"
    "log"
    "strings"
)

func (c *Client) handleRegister(parts []string) error {
    if len(parts) < 3 {
        return fmt.Errorf("usage: REGISTER <username> <password_hash>")
    }

    username := parts[1]
    passwordHash := parts[2]

    user, err := c.server.authService.Register(username, passwordHash)
    if err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }

    c.Send(fmt.Sprintf(":%s NOTICE * :Registration successful. Please login.", c.server.config.Server.ServerName))
    log.Printf("User registered: %s (ID: %d)", user.Username, user.UserID)

    return nil
}

func (c *Client) handleLogin(parts []string) error {
    if len(parts) < 3 {
        return fmt.Errorf("usage: LOGIN <username> <password_hash>")
    }

    username := parts[1]
    passwordHash := parts[2]
    ipAddress := c.GetIPAddress()

    user, err := c.server.authService.Login(username, passwordHash, ipAddress)
    if err != nil {
        return fmt.Errorf("login failed: %w", err)
    }

    if err := c.server.ipTrackingService.CheckIPAndTrack(user.UserID, ipAddress); err != nil {
        return fmt.Errorf("login blocked: %w", err)
    }

    sessionKey, err := c.server.cryptoManager.GenerateSessionKey(c.server.config.Security.AESKeySize)
    if err != nil {
        return fmt.Errorf("failed to generate session key: %w", err)
    }

    session, err := c.server.sessionManager.CreateSession(user, ipAddress, sessionKey)
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }

    c.user = user
    c.authenticated = true
    c.session = session
    c.SessionID = session.SessionID
    c.sessionKey = sessionKey

    c.server.AddClient(c)

    c.Send(fmt.Sprintf(":%s NOTICE %s :Login successful. Session ID: %s", c.server.config.Server.ServerName, username, session.SessionID))
    c.Send(fmt.Sprintf(":%s NOTICE %s :Please exchange encryption keys using KEYEXCHANGE", c.server.config.Server.ServerName, username))

    log.Printf("User logged in: %s (ID: %d) from %s", user.Username, user.UserID, ipAddress)

    return nil
}

func (c *Client) handleKeyExchange(parts []string) error {
    if err := c.requireAuth(); err != nil {
        return err
    }

    if len(parts) < 2 {
        return fmt.Errorf("usage: KEYEXCHANGE <encrypted_session_key>")
    }

    sessionKeyB64 := base64.StdEncoding.EncodeToString(c.sessionKey)

    c.Send(fmt.Sprintf("SESSIONKEY :%s", sessionKeyB64))
    c.Send(fmt.Sprintf(":%s NOTICE %s :Key exchange complete. All messages will be encrypted.", c.server.config.Server.ServerName, c.user.Username))

    return nil
}

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

func (c *Client) handlePing(parts []string) error {
    if len(parts) < 2 {
        c.Send(fmt.Sprintf("PONG :%s", c.server.config.Server.ServerName))
    } else {
        c.Send(fmt.Sprintf("PONG :%s", parts[1]))
    }

    return nil
}
