FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/finance-provider ./cmd/server/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/finance-provider /bin/finance-provider
EXPOSE 8082
ENTRYPOINT ["/bin/finance-provider"]
