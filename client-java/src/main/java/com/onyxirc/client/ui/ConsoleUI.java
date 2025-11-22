package com.onyxirc.client.ui;

import com.onyxirc.client.config.ClientConfig;
import com.onyxirc.client.network.Connection;
import com.onyxirc.client.network.MessageHandler;
import com.onyxirc.client.security.Encryption;
import com.onyxirc.client.security.Hashing;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.crypto.SecretKey;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.security.PublicKey;
import java.text.SimpleDateFormat;
import java.util.Date;

/**
 * Console-based user interface
 */
public class ConsoleUI implements MessageHandler {
    private static final Logger logger = LoggerFactory.getLogger(ConsoleUI.class);

    private final ClientConfig config;
    private Connection connection;
    private String username;
    private boolean running;
    private final SimpleDateFormat timeFormat;

    public ConsoleUI(ClientConfig config) {
        this.config = config;
        this.running = true;
        this.timeFormat = new SimpleDateFormat(config.getTimestampFormat());
    }

    public void start() {
        printBanner();

        // Create connection
        connection = new Connection(config, this);

        try {
            connection.connect();
            onConnected();

            // Start input loop
            runInputLoop();

        } catch (IOException e) {
            logger.error("Failed to connect to server", e);
            System.err.println("Error: Failed to connect to server: " + e.getMessage());
        } finally {
            connection.disconnect();
        }
    }

    @Override
    public void onConnected() {
        println("Connected to " + config.getServerHost() + ":" + config.getServerPort());
        println("Type /help for available commands");
    }

    @Override
    public void onDisconnected(String reason) {
        println("Disconnected: " + reason);
        running = false;
    }

    @Override
    public void handleMessage(String message) {
        String[] parts = message.split(" ", 3);

        if (parts.length == 0) {
            return;
        }

        String command = parts[0];

        switch (command) {
            case "PUBKEY":
                handlePublicKey(parts);
                break;

            case "SESSIONKEY":
                handleSessionKey(parts);
                break;

            case "PING":
                connection.sendRaw("PONG :" + (parts.length > 1 ? parts[1] : ""));
                break;

            case "ERROR":
                println("[ERROR] " + (parts.length > 1 ? String.join(" ", parts).substring(parts[0].length() + 1) : "Unknown error"));
                break;

            default:
                // Display the message
                println(message);
                break;
        }
    }

    private void handlePublicKey(String[] parts) {
        if (parts.length < 2) {
            return;
        }

        try {
            String pemKey = String.join(" ", parts).substring(parts[0].length() + 1);
            if (pemKey.startsWith(":")) {
                pemKey = pemKey.substring(1);
            }

            PublicKey publicKey = Encryption.loadPublicKeyFromPEM(pemKey);
            connection.setServerPublicKey(publicKey);
            println("[INFO] Server public key received");
        } catch (Exception e) {
            logger.error("Failed to load server public key", e);
            println("[ERROR] Failed to process server public key");
        }
    }

    private void handleSessionKey(String[] parts) {
        if (parts.length < 2) {
            return;
        }

        try {
            String keyB64 = parts[1];
            if (keyB64.startsWith(":")) {
                keyB64 = keyB64.substring(1);
            }

            byte[] keyBytes = java.util.Base64.getDecoder().decode(keyB64);
            SecretKey sessionKey = Encryption.bytesToKey(keyBytes);
            connection.setSessionKey(sessionKey);
            println("[INFO] Session key received. Encryption enabled.");
        } catch (Exception e) {
            logger.error("Failed to process session key", e);
            println("[ERROR] Failed to process session key");
        }
    }

    private void runInputLoop() {
        BufferedReader console = new BufferedReader(new InputStreamReader(System.in));

        try {
            while (running && connection.isConnected()) {
                System.out.print("> ");
                String line = console.readLine();

                if (line == null || line.trim().isEmpty()) {
                    continue;
                }

                processInput(line.trim());
            }
        } catch (IOException e) {
            logger.error("Error reading console input", e);
        }
    }

    private void processInput(String input) {
        if (input.startsWith("/")) {
            processCommand(input);
        } else {
            println("Error: Use commands starting with /");
            println("Type /help for available commands");
        }
    }

    private void processCommand(String input) {
        String[] parts = input.substring(1).split("\\s+", 3);
        String command = parts[0].toLowerCase();

        switch (command) {
            case "help":
                showHelp();
                break;

            case "register":
                handleRegisterCommand(parts);
                break;

            case "login":
                handleLoginCommand(parts);
                break;

            case "join":
                if (parts.length < 2) {
                    println("Usage: /join <channel>");
                } else {
                    connection.sendRaw("JOIN " + parts[1]);
                }
                break;

            case "part":
                if (parts.length < 2) {
                    println("Usage: /part <channel>");
                } else {
                    connection.sendRaw("PART " + parts[1]);
                }
                break;

            case "msg":
                if (parts.length < 3) {
                    println("Usage: /msg <target> <message>");
                } else {
                    connection.sendRaw("PRIVMSG " + parts[1] + " :" + parts[2]);
                }
                break;

            case "quit":
                connection.sendRaw("QUIT :User quit");
                running = false;
                break;

            default:
                println("Unknown command: " + command);
                println("Type /help for available commands");
                break;
        }
    }

    private void handleRegisterCommand(String[] parts) {
        if (parts.length < 3) {
            println("Usage: /register <username> <password>");
            return;
        }

        username = parts[1];
        String password = parts[2];
        String passwordHash = Hashing.hashPassword(password);

        connection.sendRaw("REGISTER " + username + " " + passwordHash);
    }

    private void handleLoginCommand(String[] parts) {
        if (parts.length < 3) {
            println("Usage: /login <username> <password>");
            return;
        }

        username = parts[1];
        String password = parts[2];
        String passwordHash = Hashing.hashPassword(password);

        connection.sendRaw("LOGIN " + username + " " + passwordHash);
    }

    private void showHelp() {
        println("Available commands:");
        println("  /register <username> <password> - Register a new account");
        println("  /login <username> <password>    - Login to your account");
        println("  /join <channel>                 - Join a channel");
        println("  /part <channel>                 - Leave a channel");
        println("  /msg <target> <message>         - Send a message");
        println("  /quit                           - Disconnect and exit");
        println("  /help                           - Show this help");
    }

    private void printBanner() {
        println("========================================");
        println("   OnyxIRC Client v1.0.0");
        println("   Secure IRC with RSA/AES Encryption");
        println("========================================");
        println("");
    }

    private void println(String message) {
        if (config.showTimestamps()) {
            System.out.println("[" + timeFormat.format(new Date()) + "] " + message);
        } else {
            System.out.println(message);
        }
    }
}
