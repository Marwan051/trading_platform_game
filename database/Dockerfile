# Custom TimescaleDB image with pre-configured extensions
# Only use this if you need custom initialization scripts in init/ directory
FROM timescale/timescaledb:2.25.0-pg18

# Install additional tools if needed
RUN apt-get update && apt-get install -y \
    postgresql-contrib \
    && rm -rf /var/lib/apt/lists/*

# Copy initialization scripts directory
# Any .sql files in init/ will run automatically on first container start
COPY --chown=postgres:postgres init/ /docker-entrypoint-initdb.d/

# Set default environment variables
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres
ENV POSTGRES_DB=trading_platform

# Expose PostgreSQL port
EXPOSE 5432
