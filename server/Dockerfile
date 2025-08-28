FROM golang:1.22
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o otel-backend ./cmd/otel-backend
EXPOSE 8080
CMD ["./otel-backend"]
