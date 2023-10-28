FROM golang:1.21 AS builder

WORKDIR /src
COPY ./ ./
RUN GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o dnshortcut

FROM scratch

COPY --from=builder /src/dnshortcut /
ENTRYPOINT [ "/dnshortcut" ]
