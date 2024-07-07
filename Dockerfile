FROM golang:1.21.11 as build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w"  -o /tgmon

FROM alpine as app
COPY --from=build /tgmon /tgmon
ENTRYPOINT [ "/tgmon" ]