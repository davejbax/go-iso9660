FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y xorriso p7zip-full tar