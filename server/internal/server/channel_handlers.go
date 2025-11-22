package server

import (
    "fmt"
    "log"

    "github.com/onyxirc/server/internal/database"
)

// handleJoinComplete implements full channel joining logic
func (c *Client) handleJoinComplete(channelName string) error {
    channelRepo := database.NewChannelRepository(c.server.db)

    // Try to get existing channel
    channel, err := channelRepo.GetByName(channelName)
    if err != nil {
        // Channel doesn't exist, create it
        channel, err = channelRepo.Create(channelName, c.user.UserID, false)
        if err != nil {
            return fmt.Errorf("failed to create channel: %w", err)
        }
        log.Printf("Channel %s created by user %s", channelName, c.user.Username)
    }

    // Check if already a member
    isMember, err := channelRepo.IsMember(channel.ChannelID, c.user.UserID)
    if err != nil {
        return fmt.Errorf("failed to check membership: %w", err)
    }

    if !isMember {
        // Add user to channel
        if err := channelRepo.AddMember(channel.ChannelID, c.user.UserID, "member"); err != nil {
            return fmt.Errorf("failed to join channel: %w", err)
        }
    }

    // Add to client's channel list
    c.JoinChannel(channel.ChannelID)

    // Notify user
    c.Send(fmt.Sprintf(":%s!%s@%s JOIN :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName))

    // Send channel topic if exists
    if channel.Topic != nil {
        c.Send(fmt.Sprintf(":%s 332 %s %s :%s",
            c.server.config.Server.ServerName, c.user.Username, channelName, *channel.Topic))
    }

    // Send member list
    members, err := channelRepo.GetMembers(channel.ChannelID)
    if err == nil {
        usernames := []string{}
        for _, member := range members {
            user, err := c.server.authService.GetUserByID(member.UserID)
            if err == nil {
                prefix := ""
                if member.Role == "owner" {
                    prefix = "@"
                } else if member.Role == "moderator" {
                    prefix = "+"
                }
                usernames = append(usernames, prefix+user.Username)
            }
        }

        // Send NAMES reply
        if len(usernames) > 0 {
            c.Send(fmt.Sprintf(":%s 353 %s = %s :%s",
                c.server.config.Server.ServerName, c.user.Username, channelName,
                joinStrings(usernames, " ")))
        }

        c.Send(fmt.Sprintf(":%s 366 %s %s :End of NAMES list",
            c.server.config.Server.ServerName, c.user.Username, channelName))
    }

    // Notify other channel members
    joinMsg := fmt.Sprintf(":%s!%s@%s JOIN :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName)
    c.server.BroadcastToChannel(channel.ChannelID, joinMsg, c.SessionID)

    log.Printf("User %s joined channel %s", c.user.Username, channelName)

    return nil
}

// handlePartComplete implements full channel leaving logic
func (c *Client) handlePartComplete(channelName string) error {
    channelRepo := database.NewChannelRepository(c.server.db)

    // Get channel
    channel, err := channelRepo.GetByName(channelName)
    if err != nil {
        return fmt.Errorf("channel not found: %s", channelName)
    }

    // Check if member
    isMember, err := channelRepo.IsMember(channel.ChannelID, c.user.UserID)
    if err != nil {
        return fmt.Errorf("failed to check membership: %w", err)
    }

    if !isMember {
        return fmt.Errorf("you are not in channel %s", channelName)
    }

    // Notify channel members
    partMsg := fmt.Sprintf(":%s!%s@%s PART :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName)
    c.server.BroadcastToChannel(channel.ChannelID, partMsg, "")

    // Remove from channel
    if err := channelRepo.RemoveMember(channel.ChannelID, c.user.UserID); err != nil {
        return fmt.Errorf("failed to leave channel: %w", err)
    }

    // Remove from client's channel list
    c.LeaveChannel(channel.ChannelID)

    // Notify user
    c.Send(partMsg)

    log.Printf("User %s left channel %s", c.user.Username, channelName)

    return nil
}

// handlePrivMsgComplete implements full message sending logic
func (c *Client) handlePrivMsgComplete(target, message string) error {
    // Check if target is a channel (starts with #)
    if target[0] == '#' {
        return c.sendChannelMessage(target, message)
    }

    // Otherwise it's a direct message
    return c.sendDirectMessage(target, message)
}

// sendChannelMessage sends a message to a channel
func (c *Client) sendChannelMessage(channelName, message string) error {
    channelRepo := database.NewChannelRepository(c.server.db)

    // Get channel
    channel, err := channelRepo.GetByName(channelName)
    if err != nil {
        return fmt.Errorf("channel not found: %s", channelName)
    }

    // Check if member
    isMember, err := channelRepo.IsMember(channel.ChannelID, c.user.UserID)
    if err != nil {
        return fmt.Errorf("failed to check membership: %w", err)
    }

    if !isMember {
        return fmt.Errorf("cannot send to channel %s: not a member", channelName)
    }

    // TODO: Encrypt message before storing
    // For now, store plaintext
    // messageRepo.StoreMessage(channel.ChannelID, c.user.UserID, message)

    // Broadcast to channel
    msg := fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName, message)

    c.server.BroadcastToChannel(channel.ChannelID, msg, c.SessionID)

    // Echo back to sender
    c.Send(msg)

    log.Printf("User %s sent message to channel %s: %s", c.user.Username, channelName, message)

    return nil
}

// sendDirectMessage sends a direct message to a user
func (c *Client) sendDirectMessage(targetUsername, message string) error {
    // Get target user
    targetUser, err := c.server.authService.GetUserByUsername(targetUsername)
    if err != nil {
        return fmt.Errorf("user not found: %s", targetUsername)
    }

    // Find target client if online
    var targetClient *Client
    c.server.clientsMu.RLock()
    for _, client := range c.server.clients {
        if client.user != nil && client.user.UserID == targetUser.UserID {
            targetClient = client
            break
        }
    }
    c.server.clientsMu.RUnlock()

    msg := fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), targetUsername, message)

    if targetClient != nil {
        // User is online, deliver immediately
        targetClient.Send(msg)
        log.Printf("User %s sent DM to %s: %s", c.user.Username, targetUsername, message)
    } else {
        // User is offline, store for later
        // TODO: Implement offline message storage
        log.Printf("User %s sent DM to offline user %s: %s", c.user.Username, targetUsername, message)
        return fmt.Errorf("user %s is offline (message not delivered)", targetUsername)
    }

    return nil
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
    if len(strs) == 0 {
        return ""
    }
    result := strs[0]
    for i := 1; i < len(strs); i++ {
        result += sep + strs[i]
    }
    return result
}
