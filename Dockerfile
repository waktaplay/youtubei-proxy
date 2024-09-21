FROM golang:1.23.1-alpine
WORKDIR /usr/src/app

# Effectively tracks changes within your go.mod file
COPY src/go.mod .
COPY src/go.sum .
RUN go mod download

# Copies your source code into the app directory
COPY src/main.go .
RUN go build -o proxy

# Run the app
EXPOSE 8123/tcp

CMD ["./proxy"]
