FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w"  -o ./tgmon

FROM alpine AS app
ENV SESSION_DIR="/TGMon/session"
ENV WORKER_CACHE_ROOT="/TGMon/worker-cache"
ENV GIN_MODE=release
ENV LOG_LEVEL=WARNING
RUN mkdir -p $SESSION_DIR &&\
    mkdir -p $WORKER_CACHE_ROOT
COPY --from=build /app/tgmon /bin/tgmon
VOLUME /TGMon
CMD ["/bin/tgmon"]
