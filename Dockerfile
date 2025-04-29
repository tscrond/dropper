FROM golang:1.24.1-alpine AS builder

WORKDIR /dropper

COPY . .

RUN go mod download

RUN go build -o /dropper/dropper /dropper/cmd

EXPOSE 3000

FROM golang:1.24.1-alpine

WORKDIR /dropper

COPY --from=builder /dropper/dropper /dropper/dropper
COPY --from=builder /dropper/internal/repo /dropper/internal/repo

CMD [ "/dropper/dropper" ]
