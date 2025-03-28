FROM golang:1.21-alpine as builder
WORKDIR /app
COPY . .
RUN go build -o telemetry-tracker

FROM alpine
COPY --from=builder /app/telemetry-tracker /telemetry-tracker
EXPOSE 8080
ENTRYPOINT ["/telemetry-tracker"]
