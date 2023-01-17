FROM node:19-bullseye

RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

ARG LINK=no

COPY package.json .
COPY package-lock.json .

# Use force due to version conflict
RUN npm ci --fetch-timeout=600000 --force

COPY . /usr/src/app

RUN npm run build
ENTRYPOINT npm run start

EXPOSE 3000
