FROM golang:1.18-alpine
RUN apk add --no-cache git make build-base
WORKDIR /go/app
COPY . ./
RUN go get .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/app ./
CMD ["./app"]