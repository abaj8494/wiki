#!/bin/bash

# Make script executable with: chmod +x deploy.sh

# Install Docker and Docker Compose if not already installed
if ! [ -x "$(command -v docker)" ]; then
  echo 'Installing Docker...'
  curl -fsSL https://get.docker.com -o get-docker.sh
  sh get-docker.sh
  rm get-docker.sh
fi

if ! [ -x "$(command -v docker-compose)" ]; then
  echo 'Installing Docker Compose...'
  curl -L "https://github.com/docker/compose/releases/download/v2.20.3/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
fi

# Create persistent directory if it doesn't exist
echo 'Ensuring persistent storage directory exists...'
mkdir -p /var/www/wiki/persistence
chmod 755 /var/www/wiki/persistence

# Build and start the container
echo 'Building and starting wiki container...'
docker-compose down
docker-compose build --no-cache
docker-compose up -d

echo 'Deployment complete!'
echo 'Your wiki should be running at http://<your-vultr-ip>:21313'
echo 'Files are now persisted in /var/www/wiki/persistence' 