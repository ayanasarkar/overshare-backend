FROM golang:1.22-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o overshare-backend .

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/overshare-backend .
COPY --from=builder /app/demo-assets ./demo-assets
EXPOSE 8080
ENTRYPOINT ["/app/overshare-backend"]
