FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w \
      -X github.com/vinaycharlie01/sh-mcp-go/pkg/version.Version=${VERSION} \
      -X github.com/vinaycharlie01/sh-mcp-go/pkg/version.Commit=${COMMIT} \
      -X github.com/vinaycharlie01/sh-mcp-go/pkg/version.BuildDate=${BUILD_DATE}" \
    -o /bin/sh-mcp-go \
    ./cmd/sh-mcp-go

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/sh-mcp-go /sh-mcp-go
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080 8081

ENTRYPOINT ["/sh-mcp-go"]
