package com.onyxirc.client.config;

import java.io.IOException;
import java.io.InputStream;
import java.util.Properties;

/**
 * Client configuration loader
 */
public class ClientConfig {
    private final Properties properties;

    private ClientConfig(Properties properties) {
        this.properties = properties;
    }

    public static ClientConfig load(String resourceName) throws IOException {
        Properties props = new Properties();

        try (InputStream input = ClientConfig.class.getClassLoader().getResourceAsStream(resourceName)) {
            if (input == null) {
                throw new IOException("Unable to find " + resourceName);
            }
            props.load(input);
        }

        return new ClientConfig(props);
    }

    public String getServerHost() {
        return properties.getProperty("server.host", "localhost");
    }

    public int getServerPort() {
        return Integer.parseInt(properties.getProperty("server.port", "6667"));
    }

    public int getServerTimeout() {
        return Integer.parseInt(properties.getProperty("server.timeout", "30000"));
    }

    public int getRSAKeySize() {
        return Integer.parseInt(properties.getProperty("security.rsa_key_size", "2048"));
    }

    public int getAESKeySize() {
        return Integer.parseInt(properties.getProperty("security.aes_key_size", "256"));
    }

    public boolean isAutoReconnect() {
        return Boolean.parseBoolean(properties.getProperty("client.auto_reconnect", "true"));
    }

    public int getReconnectDelay() {
        return Integer.parseInt(properties.getProperty("client.reconnect_delay", "5000"));
    }

    public int getMaxReconnectAttempts() {
        return Integer.parseInt(properties.getProperty("client.max_reconnect_attempts", "10"));
    }

    public boolean showTimestamps() {
        return Boolean.parseBoolean(properties.getProperty("ui.show_timestamps", "true"));
    }

    public String getTimestampFormat() {
        return properties.getProperty("ui.timestamp_format", "HH:mm:ss");
    }

    public String getProperty(String key) {
        return properties.getProperty(key);
    }

    public String getProperty(String key, String defaultValue) {
        return properties.getProperty(key, defaultValue);
    }
}
