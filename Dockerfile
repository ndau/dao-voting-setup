FROM golang:1.19-alpine3.16 AS build

# Install tools required to build the project
# We will need to run `docker build --no-cache .` to update those dependencies
RUN apk update \
    && apk add --no-cache git make openssh

# Uncomment to load an SSH key as a build arg for private github repos
ARG SSH_PRIVATE_KEY

RUN mkdir -p /root/.ssh

COPY ${SSH_PRIVATE_KEY} /root/.ssh/id_rsa

RUN git config --global url."git@github.com:".insteadOf "https://github.com/"\
    && chmod 400 /root/.ssh/id_rsa \
    && touch /root/.ssh/known_hosts \
    && ssh-keyscan github.com >> /root/.ssh/known_hosts

COPY go.mod /go/src/project/
WORKDIR /go/src/project

# Copy all project and build it
COPY Makefile .

COPY . .

RUN make build


# This results in a single layer image
FROM alpine:3.12

WORKDIR /opt/app

# Add TLS certs for HTTPS requests
RUN apk update \
    && apk add ca-certificates --no-cache \
    && update-ca-certificates \
    && adduser ndau-user -s /bin/bash -D \
    && chown -R ndau-user: /opt/app
 
# Copy binaries over
COPY --from=build /go/src/project/bin/project /opt/app/project

RUN mkdir -p /opt/app/config

# Uncomment if this app will listen to HTTP requests
EXPOSE 8080

# Separate executable and arguments list
ENTRYPOINT ["/opt/app/project"]
