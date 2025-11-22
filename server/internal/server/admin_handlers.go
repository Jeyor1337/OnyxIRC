package server

import (
    "fmt"
    "log"
    "strings"

    "github.com/onyxirc/server/internal/admin"
)

func (c *Client) handleAdminCommand(parts []string) error {
    if err := c.requireAuth(); err != nil {
        return err
    }

    if len(parts) < 2 {
        return fmt.Errorf("usage: ADMIN <subcommand> [args...]")
    }

    subcommand := strings.ToLower(parts[1])

    switch subcommand {
    case "kick":
        return c.handleAdminKick(parts[2:])
    case "ban":
        return c.handleAdminBan(parts[2:])
    case "unban":
        return c.handleAdminUnban(parts[2:])
    case "unlock":
        return c.handleAdminUnlock(parts[2:])
    case "makeadmin":
        return c.handleAdminMakeAdmin(parts[2:])
    case "removeadmin":
        return c.handleAdminRemoveAdmin(parts[2:])
    case "broadcast":
        return c.handleAdminBroadcast(parts[2:])
    case "stats":
        return c.handleAdminStats(parts[2:])
    case "log":
        return c.handleAdminLog(parts[2:])
    default:
        return fmt.Errorf("unknown admin command: %s", subcommand)
    }
}

func (c *Client) handleAdminKick(args []string) error {
    if len(args) < 2 {
        return fmt.Errorf("usage: ADMIN kick <username> <reason>")
    }

    username := args[0]
    reason := strings.Join(args[1:], " ")

    if err := c.server.adminService.KickUser(c.user.UserID, username, reason); err != nil {
        return err
    }

    c.server.clientsMu.RLock()
    for _, client := range c.server.clients {
        if client.user != nil && client.user.Username == username {
            client.Send(fmt.Sprintf("ERROR :Kicked by admin: %s", reason))
            go client.Disconnect()
            break
        }
    }
    c.server.clientsMu.RUnlock()

    c.Send(fmt.Sprintf(":%s NOTICE %s :User %s has been kicked", c.server.config.Server.ServerName, c.user.Username, username))
    log.Printf("Admin %s kicked user %s: %s", c.user.Username, username, reason)

    return nil
}

func (c *Client) handleAdminBan(args []string) error {
    if len(args) < 3 {
        return fmt.Errorf("usage: ADMIN ban <username> <duration_seconds> <reason>")
    }

    username := args[0]
    durationStr := args[1]
    reason := strings.Join(args[2:], " ")

    durationSeconds, err := admin.ParseDuration(durationStr)
    if err != nil {
        return err
    }

    if err := c.server.adminService.BanUser(c.user.UserID, username, reason, durationSeconds); err != nil {
        return err
    }

    c.server.clientsMu.RLock()
    for _, client := range c.server.clients {
        if client.user != nil && client.user.Username == username {
            client.Send(fmt.Sprintf("ERROR :Banned by admin: %s", reason))
            go client.Disconnect()
            break
        }
    }
    c.server.clientsMu.RUnlock()

    banType := "permanently"
    if durationSeconds > 0 {
        banType = fmt.Sprintf("for %d seconds", durationSeconds)
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :User %s has been banned %s", c.server.config.Server.ServerName, c.user.Username, username, banType))
    log.Printf("Admin %s banned user %s %s: %s", c.user.Username, username, banType, reason)

    return nil
}

func (c *Client) handleAdminUnban(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: ADMIN unban <username>")
    }

    username := args[0]

    if err := c.server.adminService.UnbanUser(c.user.UserID, username); err != nil {
        return err
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :User %s has been unbanned", c.server.config.Server.ServerName, c.user.Username, username))
    log.Printf("Admin %s unbanned user %s", c.user.Username, username)

    return nil
}

func (c *Client) handleAdminUnlock(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: ADMIN unlock <username>")
    }

    username := args[0]

    if err := c.server.adminService.UnlockAccount(c.user.UserID, username); err != nil {
        return err
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :Account unlocked for user %s", c.server.config.Server.ServerName, c.user.Username, username))
    log.Printf("Admin %s unlocked account for user %s", c.user.Username, username)

    return nil
}

func (c *Client) handleAdminMakeAdmin(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: ADMIN makeadmin <username>")
    }

    username := args[0]

    targetUser, err := c.server.authService.GetUserByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if err := c.server.adminService.MakeAdmin(c.user.UserID, targetUser.UserID); err != nil {
        return err
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :Admin privileges granted to %s", c.server.config.Server.ServerName, c.user.Username, username))
    log.Printf("Admin %s granted admin privileges to user %s", c.user.Username, username)

    return nil
}

func (c *Client) handleAdminRemoveAdmin(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: ADMIN removeadmin <username>")
    }

    username := args[0]

    targetUser, err := c.server.authService.GetUserByUsername(username)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if err := c.server.adminService.RemoveAdmin(c.user.UserID, targetUser.UserID); err != nil {
        return err
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :Admin privileges revoked from %s", c.server.config.Server.ServerName, c.user.Username, username))
    log.Printf("Admin %s revoked admin privileges from user %s", c.user.Username, username)

    return nil
}

func (c *Client) handleAdminBroadcast(args []string) error {
    if len(args) < 1 {
        return fmt.Errorf("usage: ADMIN broadcast <message>")
    }

    message := strings.Join(args, " ")

    if err := c.server.adminService.BroadcastMessage(c.user.UserID, message); err != nil {
        return err
    }

    broadcastMsg := fmt.Sprintf(":%s NOTICE * :[BROADCAST] %s", c.server.config.Server.ServerName, message)

    c.server.clientsMu.RLock()
    for _, client := range c.server.clients {
        client.Send(broadcastMsg)
    }
    c.server.clientsMu.RUnlock()

    log.Printf("Admin %s broadcast message: %s", c.user.Username, message)

    return nil
}

func (c *Client) handleAdminStats(args []string) error {
    stats, err := c.server.adminService.GetServerStats(c.user.UserID)
    if err != nil {
        return err
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :=== Server Statistics ===", c.server.config.Server.ServerName, c.user.Username))

    for key, value := range stats {
        c.Send(fmt.Sprintf(":%s NOTICE %s :%s: %v", c.server.config.Server.ServerName, c.user.Username, key, value))
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :active_connections: %d", c.server.config.Server.ServerName, c.user.Username, c.server.GetActiveClientCount()))
    c.Send(fmt.Sprintf(":%s NOTICE %s :active_sessions: %d", c.server.config.Server.ServerName, c.user.Username, c.server.sessionManager.GetActiveSessionCount()))

    return nil
}

func (c *Client) handleAdminLog(args []string) error {
    limit := 10
    if len(args) > 0 {
        fmt.Sscanf(args[0], "%d", &limit)
    }

    logs, err := c.server.adminService.GetAdminLog(c.user.UserID, limit, 0)
    if err != nil {
        return err
    }

    c.Send(fmt.Sprintf(":%s NOTICE %s :=== Admin Action Log (last %d) ===", c.server.config.Server.ServerName, c.user.Username, limit))

    for _, log := range logs {
        c.Send(fmt.Sprintf(":%s NOTICE %s :[%s] Admin ID %d: %s - %s",
            c.server.config.Server.ServerName,
            c.user.Username,
            log.PerformedAt.Format("2006-01-02 15:04:05"),
            log.AdminID,
            log.ActionType,
            *log.ActionDetails))
    }

    return nil
}
