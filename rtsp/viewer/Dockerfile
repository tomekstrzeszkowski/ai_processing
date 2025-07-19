# Use Node.js base image
FROM node:22

WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 8081 19000 19001 19002
RUN npm install -g expo-cli
CMD ["npx", "expo", "start", "--web"]
