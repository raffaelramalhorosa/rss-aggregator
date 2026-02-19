# ---------- Build stage ----------
FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# ---------- Runtime stage ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /server /server
COPY --from=build /app/static /static

EXPOSE 8080

ENTRYPOINT ["/server"]