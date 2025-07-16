FROM golang:1.22.6 AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w"  -o /tgmon-server ./cmd/server &&\
    CGO_ENABLED=0 go build -ldflags="-s -w"  -o /tgmon-bot ./cmd/bot &&\
    CGO_ENABLED=0 go build -ldflags="-s -w"  -o /tgmon-cli ./cmd/cli

FROM alpine AS app
ENV SESSION_DIR="/TGMon/session"
ENV ACCESS_LOG="/TGMon/log/access.log"
ENV GIN_MODE=release
RUN mkdir -p /TGMon/session &&\
    mkdir -p /TGMon/log &&\
    alias tgmon-server=/tgmon-server &&\
    alias tgmon-bot=/tgmon-bot &&\
    alias tgmon-cli=/tgmon-cli
COPY --from=build /tgmon-server /bin/tgmon-server
COPY --from=build /tgmon-bot /bin/tgmon-bot
COPY --from=build /tgmon-cli /bin/tgmon-cli
