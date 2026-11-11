# ---- build ----
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/hako .

# ---- runtime ----
FROM alpine:3.22
RUN apk add --no-cache ffmpeg tzdata ca-certificates
COPY --from=build /out/hako /usr/local/bin/hako
ENV HAKO_DATA=/data \
    HAKO_ADDR=:8096 \
    TZ=Asia/Tokyo
VOLUME /data
EXPOSE 8096
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD wget -qO- http://127.0.0.1:8096/api/setup >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/hako"]
