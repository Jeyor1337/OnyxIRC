package server

import (
    "fmt"
    "log"

    "github.com/onyxirc/server/internal/database"
)

func (c *Client) handleJoinComplete(channelName string) error {
    channelRepo := database.NewChannelRepository(c.server.db)

    channel, err := channelRepo.GetByName(channelName)
    if err != nil {
        
        channel, err = channelRepo.Create(channelName, c.user.UserID, false)
        if err != nil {
            return fmt.Errorf("failed to create channel: %w", err)
        }
        log.Printf("Channel %s created by user %s", channelName, c.user.Username)
    }

    isMember, err := channelRepo.IsMember(channel.ChannelID, c.user.UserID)
    if err != nil {
        return fmt.Errorf("failed to check membership: %w", err)
    }

    if !isMember {
        
        if err := channelRepo.AddMember(channel.ChannelID, c.user.UserID, "member"); err != nil {
            return fmt.Errorf("failed to join channel: %w", err)
        }
    }

    c.JoinChannel(channel.ChannelID)

    c.Send(fmt.Sprintf(":%s!%s@%s JOIN :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName))

    if channel.Topic != nil {
        c.Send(fmt.Sprintf(":%s 332 %s %s :%s",
            c.server.config.Server.ServerName, c.user.Username, channelName, *channel.Topic))
    }

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

        if len(usernames) > 0 {
            c.Send(fmt.Sprintf(":%s 353 %s = %s :%s",
                c.server.config.Server.ServerName, c.user.Username, channelName,
                joinStrings(usernames, " ")))
        }

        c.Send(fmt.Sprintf(":%s 366 %s %s :End of NAMES list",
            c.server.config.Server.ServerName, c.user.Username, channelName))
    }

    joinMsg := fmt.Sprintf(":%s!%s@%s JOIN :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName)
    c.server.BroadcastToChannel(channel.ChannelID, joinMsg, c.SessionID)

    log.Printf("User %s joined channel %s", c.user.Username, channelName)

    return nil
}

func (c *Client) handlePartComplete(channelName string) error {
    channelRepo := database.NewChannelRepository(c.server.db)

    channel, err := channelRepo.GetByName(channelName)
    if err != nil {
        return fmt.Errorf("channel not found: %s", channelName)
    }

    isMember, err := channelRepo.IsMember(channel.ChannelID, c.user.UserID)
    if err != nil {
        return fmt.Errorf("failed to check membership: %w", err)
    }

    if !isMember {
        return fmt.Errorf("you are not in channel %s", channelName)
    }

    partMsg := fmt.Sprintf(":%s!%s@%s PART :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName)
    c.server.BroadcastToChannel(channel.ChannelID, partMsg, "")

    if err := channelRepo.RemoveMember(channel.ChannelID, c.user.UserID); err != nil {
        return fmt.Errorf("failed to leave channel: %w", err)
    }

    c.LeaveChannel(channel.ChannelID)

    c.Send(partMsg)

    log.Printf("User %s left channel %s", c.user.Username, channelName)

    return nil
}

func (c *Client) handlePrivMsgComplete(target, message string) error {
    
    if target[0] == '#' {
        return c.sendChannelMessage(target, message)
    }

    return c.sendDirectMessage(target, message)
}

func (c *Client) sendChannelMessage(channelName, message string) error {
    channelRepo := database.NewChannelRepository(c.server.db)

    channel, err := channelRepo.GetByName(channelName)
    if err != nil {
        return fmt.Errorf("channel not found: %s", channelName)
    }

    isMember, err := channelRepo.IsMember(channel.ChannelID, c.user.UserID)
    if err != nil {
        return fmt.Errorf("failed to check membership: %w", err)
    }

    if !isMember {
        return fmt.Errorf("cannot send to channel %s: not a member", channelName)
    }

    msg := fmt.Sprintf(":%s!%s@%s PRIVMSG %s :%s",
        c.user.Username, c.user.Username, c.GetIPAddress(), channelName, message)

    c.server.BroadcastToChannel(channel.ChannelID, msg, c.SessionID)

    c.Send(msg)

    log.Printf("User %s sent message to channel %s: %s", c.user.Username, channelName, message)

    return nil
}

func (c *Client) sendDirectMessage(targetUsername, message string) error {
    
    targetUser, err := c.server.authService.GetUserByUsername(targetUsername)
    if err != nil {
        return fmt.Errorf("user not found: %s", targetUsername)
    }

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
        
        targetClient.Send(msg)
        log.Printf("User %s sent DM to %s: %s", c.user.Username, targetUsername, message)
    } else {
        
        log.Printf("User %s sent DM to offline user %s: %s", c.user.Username, targetUsername, message)
        return fmt.Errorf("user %s is offline (message not delivered)", targetUsername)
    }

    return nil
}

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
