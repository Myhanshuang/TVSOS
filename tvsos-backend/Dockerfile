# ==== Part1. 构建与编译阶段 ====
FROM golang:1.25.1-alpine AS builder

WORKDIR /usr/local/app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -v -o mover-backend

# ==== Part2. 运行阶段 ====
FROM alpine:latest AS runtime

WORKDIR /usr/local/app

COPY --from=builder /usr/local/app/mover-backend ./

RUN chmod +x mover-backend

EXPOSE 8088

CMD ["./mover-backend"]
