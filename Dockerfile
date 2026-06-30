FROM golang:1.26-alpine AS build

WORKDIR /src
ENV GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/finance-provider ./cmd/server/

FROM alpine:3.21
COPY --from=build /bin/finance-provider /bin/finance-provider
EXPOSE 8082
ENTRYPOINT ["/bin/finance-provider"]
