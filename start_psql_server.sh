go build
docker build . -t pincher-api:latest
docker-compose down --remove-orphans
docker-compose up -d --build
