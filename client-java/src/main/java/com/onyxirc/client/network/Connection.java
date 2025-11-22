package com.onyxirc.client.network;

import com.onyxirc.client.config.ClientConfig;
import com.onyxirc.client.security.Encryption;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.crypto.SecretKey;
import java.io.*;
import java.net.Socket;
import java.security.PublicKey;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingQueue;

/**
 * Manages connection to IRC server
 */
public class Connection {
    private static final Logger logger = LoggerFactory.getLogger(Connection.class);

    private final ClientConfig config;
    private Socket socket;
    private BufferedReader reader;
    private PrintWriter writer;
    private MessageHandler messageHandler;
    private Thread receiverThread;
    private Thread senderThread;
    private BlockingQueue<String> sendQueue;
    private volatile boolean connected;
    private PublicKey serverPublicKey;
    private SecretKey sessionKey;

    public Connection(ClientConfig config, MessageHandler messageHandler) {
        this.config = config;
        this.messageHandler = messageHandler;
        this.sendQueue = new LinkedBlockingQueue<>();
        this.connected = false;
    }

    /**
     * Connects to the server
     */
    public void connect() throws IOException {
        logger.info("Connecting to {}:{}", config.getServerHost(), config.getServerPort());

        socket = new Socket(config.getServerHost(), config.getServerPort());
        socket.setSoTimeout(config.getServerTimeout());

        reader = new BufferedReader(new InputStreamReader(socket.getInputStream()));
        writer = new PrintWriter(socket.getOutputStream(), true);

        connected = true;

        // Start receiver thread
        receiverThread = new Thread(this::receiveMessages, "Receiver");
        receiverThread.start();

        // Start sender thread
        senderThread = new Thread(this::sendMessages, "Sender");
        senderThread.start();

        logger.info("Connected to server");
    }

    /**
     * Disconnects from the server
     */
    public void disconnect() {
        if (!connected) {
            return;
        }

        logger.info("Disconnecting from server");
        connected = false;

        try {
            if (writer != null) {
                writer.println("QUIT :Client disconnecting");
            }

            if (socket != null && !socket.isClosed()) {
                socket.close();
            }

            if (receiverThread != null) {
                receiverThread.interrupt();
            }

            if (senderThread != null) {
                senderThread.interrupt();
            }
        } catch (IOException e) {
            logger.error("Error during disconnect", e);
        }

        logger.info("Disconnected");
    }

    /**
     * Sends a raw message to the server
     */
    public void sendRaw(String message) {
        if (!connected) {
            logger.warn("Cannot send message: not connected");
            return;
        }

        try {
            sendQueue.put(message);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            logger.error("Interrupted while queueing message", e);
        }
    }

    /**
     * Receiver thread - reads messages from server
     */
    private void receiveMessages() {
        try {
            String line;
            while (connected && (line = reader.readLine()) != null) {
                logger.debug("Received: {}", line);
                messageHandler.handleMessage(line);
            }
        } catch (IOException e) {
            if (connected) {
                logger.error("Error receiving messages", e);
                messageHandler.onDisconnected("Connection lost");
            }
        } finally {
            connected = false;
        }
    }

    /**
     * Sender thread - sends messages to server
     */
    private void sendMessages() {
        try {
            while (connected) {
                String message = sendQueue.take();
                logger.debug("Sending: {}", message);
                writer.println(message);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Sets the server's public key
     */
    public void setServerPublicKey(PublicKey key) {
        this.serverPublicKey = key;
    }

    /**
     * Sets the session key
     */
    public void setSessionKey(SecretKey key) {
        this.sessionKey = key;
    }

    /**
     * Gets the session key
     */
    public SecretKey getSessionKey() {
        return sessionKey;
    }

    /**
     * Checks if connected
     */
    public boolean isConnected() {
        return connected;
    }

    /**
     * Encrypts and sends a message
     */
    public void sendEncrypted(String message) throws Exception {
        if (sessionKey == null) {
            throw new IllegalStateException("No session key established");
        }

        String encrypted = Encryption.encryptAESGCM(sessionKey, message);
        sendRaw("ENCRYPTED :" + encrypted);
    }
}
