package com.onyxirc.client;

import com.onyxirc.client.config.ClientConfig;
import com.onyxirc.client.ui.ConsoleUI;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

/**
 * OnyxIRC Client Main Entry Point
 */
public class Main {
    private static final Logger logger = LoggerFactory.getLogger(Main.class);

    public static void main(String[] args) {
        logger.info("Starting OnyxIRC Client v1.0.0");

        try {
            // Load client configuration
            ClientConfig config = ClientConfig.load("client.properties");
            logger.info("Configuration loaded successfully");

            // Start console UI
            ConsoleUI ui = new ConsoleUI(config);
            ui.start();

        } catch (IOException e) {
            logger.error("Failed to load configuration: {}", e.getMessage(), e);
            System.err.println("Error: Failed to load configuration");
            System.err.println("Please ensure client.properties exists in the resources directory");
            System.exit(1);
        } catch (Exception e) {
            logger.error("Unexpected error: {}", e.getMessage(), e);
            System.err.println("Error: " + e.getMessage());
            System.exit(1);
        }
    }
}
