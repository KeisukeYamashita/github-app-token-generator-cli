FROM alpine:latest
LABEL org.opencontainers.image.source https://github.com/KeisukeYamashita/github-app-token-generator-cli

COPY github-app-token-generator-cli /
ENTRYPOINT ["/github-app-token-generator-cli"]
