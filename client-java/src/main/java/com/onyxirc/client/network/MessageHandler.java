package com.onyxirc.client.network;

public interface MessageHandler {
    
    void handleMessage(String message);

    void onDisconnected(String reason);

    void onConnected();
}
