FROM node:20-alpine
WORKDIR /app
COPY --chown=10001:10001 . .
USER 10001:10001
EXPOSE 8080
CMD ["node", "server.js"]
