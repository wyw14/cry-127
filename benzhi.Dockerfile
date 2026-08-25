FROM golang:1.26.2
ENV GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local CGO_ENABLED=1
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
RUN go build -mod=vendor ./...
CMD ["go", "run", "-mod=vendor", "./cmd/forgevac", "-addr", "0.0.0.0:21227", "-data", "/tmp/forgevac"]
