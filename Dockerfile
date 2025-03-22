FROM golang:1.24.1-alpine AS builder

WORKDIR /dropper

COPY . .

RUN go mod download

RUN go build -o /dropper

EXPOSE 3000

FROM golang:1.24.1-alpine

WORKDIR /dropper

COPY --from=builder /dropper /dropper
CMD [ "/dropper/dropper" ]
