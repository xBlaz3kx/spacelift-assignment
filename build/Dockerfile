FROM golang:1.21.1 as build

WORKDIR /mnt/homework

# Set environment variables for the build
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-build
ENV GO111MODULE=on
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOOS=linux

# Leverage build cache by copying go.mod and go.sum first
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
RUN go mod verify

# Copy the rest of the source code
COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" go build -o ./gateway main.go

# Debug stage using Delve
FROM build as debug

COPY --from=build /mnt/homework /mnt/homework

WORKDIR /mnt/homework

# Install Delve and build the application
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go build -gcflags="all=-N -l" -o ./gateway ./cmd

CMD ["/go/bin/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./gateway"]

FROM alpine as app

# Add curl for health checks
RUN apk --update --no-cache add curl

SHELL ["/bin/sh", "-l", "-c"]

# Copy the binary from the build stage
COPY --chown=$user:$group --from=build /mnt/homework/gateway /usr/local/bin/gateway
RUN chmod +x /usr/local/bin/gateway
WORKDIR /usr/local/gateway

# Create a non-root user
ARG user=app
ARG group=docker
RUN addgroup $group
RUN adduser \
    --disabled-password \
    --gecos "" \
    --ingroup $group \
    $user
USER $user

# Expose the default http port
EXPOSE 3000

# Add healthcheck
HEALTHCHECK --interval=5s --timeout=1s --start-period=3s --retries=3 CMD curl -f http://localhost:3000/live || exit 1

CMD ["/usr/local/bin/gateway"]