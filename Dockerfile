# Use debian as the base image
FROM debian:stable-slim

# COPY source destination
COPY pincher-api /bin/pincher-api

ENV PORT=8080

# Start the server process
CMD ["/bin/pincher-api"]

