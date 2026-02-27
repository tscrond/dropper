ARG TARGETOS
ARG TARGETARCH

FROM golang:1.24.3-alpine3.21 AS builder

WORKDIR /dropper

COPY . .

RUN go mod download

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /dropper/dropper /dropper/cmd

EXPOSE 3000

FROM golang:1.24.3-alpine3.21

WORKDIR /dropper

COPY --from=builder /dropper/dropper /dropper/dropper
COPY --from=builder /dropper/internal/repo /dropper/internal/repo

CMD [ "/dropper/dropper" ]
