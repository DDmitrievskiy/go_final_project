FROM golang:1.22.5 AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o scheduler


FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/scheduler .
COPY --from=builder /app/web ./web

ENV TODO_PORT=7540
ENV TODO_DBFILE=/db/scheduler.db
EXPOSE $TODO_PORT

CMD ["./scheduler"]