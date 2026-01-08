FROM golang:1.25.2-alpine AS builder

RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /app/go

COPY go/go.mod go/go.sum ./

RUN go mod download

COPY go/ ./

#前端
WORKDIR /app/solid-project

RUN apk add --no-cache nodejs npm

RUN node -v && npm -v

COPY solid-project/package*.json ./

RUN npm install

COPY solid-project/ ./

RUN npm run build

#构建
WORKDIR /app

RUN cd go && go build -gcflags="all=-N -l" -o ../server_binary .

FROM alpine:latest AS runner

RUN apk add --no-cache libc6-compat

WORKDIR /app

COPY --from=builder /go/bin/dlv /usr/local/bin/dlv
COPY --from=builder /app/server_binary .

EXPOSE 80 2345

# CMD [ "./server_binary" ]
CMD ["dlv", "--listen=:2345", "--headless=true", "--api-version=2", "exec", "--continue", "--accept-multiclient", "./server_binary"]