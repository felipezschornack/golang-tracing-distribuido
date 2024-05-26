FROM golang:1.22.3-alpine3.20 AS BUILDER
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -o weather-by-cep main.go

FROM scratch
WORKDIR /app
COPY --from=BUILDER /app/weather-by-cep /app/weather-by-cep
ENTRYPOINT ["./weather-by-cep"]