FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git make curl
ENV GOOS=linux
ENV CGO_ENABLED=0
COPY . /src
WORKDIR /src
RUN make test
RUN make alpine

FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata
RUN export PATH=$PATH:/app
WORKDIR /app
COPY --from=builder /src/bin/gemini /app/gemini
CMD ["/app/gemini"]
