FROM golang:1.18 as builder
WORKDIR /app/
COPY . .
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://athens.chuhlomin.com \
    go build -mod=readonly -a -installsuffix cgo \
    -o server .

FROM scratch
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
