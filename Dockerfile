FROM golang:1.22-alpine

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .
EXPOSE 8080
ENV DB_STR "postgresql://postgres@host.docker.internal:5432/haven"

CMD ["./main"]