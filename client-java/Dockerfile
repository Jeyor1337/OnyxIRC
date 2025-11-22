# OnyxIRC Java Client Dockerfile
FROM maven:3.9-eclipse-temurin-17 AS builder

# Set working directory
WORKDIR /app

# Copy pom.xml
COPY pom.xml .

# Download dependencies
RUN mvn dependency:go-offline

# Copy source code
COPY src ./src

# Build the application
RUN mvn clean package -DskipTests

# Final stage
FROM eclipse-temurin:17-jre-alpine

# Create app directory
WORKDIR /app

# Copy JAR from builder
COPY --from=builder /app/target/onyxirc-client.jar .

# Copy resources
COPY --from=builder /app/src/main/resources ./resources

# Run the client
CMD ["java", "-jar", "onyxirc-client.jar"]
