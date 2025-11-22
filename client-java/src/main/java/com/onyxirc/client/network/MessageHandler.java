package com.onyxirc.client.network;

/**
 * Interface for handling messages from server
 */
public interface MessageHandler {
    /**
     * Handles a message received from the server
     */
    void handleMessage(String message);

    /**
     * Called when disconnected from server
     */
    void onDisconnected(String reason);

    /**
     * Called when connection is established
     */
    void onConnected();
}
