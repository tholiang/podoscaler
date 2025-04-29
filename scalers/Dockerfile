FROM golang:1.24-alpine as builder

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Set build arg and working directory
ARG BUILD_TAG
WORKDIR /app

# Copy go.mod and go.sum first — maximize cache
COPY ./go.mod ./go.sum ./
RUN go mod download

# Copy only necessary source directories
COPY ./autoscaler ./autoscaler
COPY ./util ./util
COPY ./autoscalertest ./autoscalertest
COPY ./main ./main

# Build the binary
RUN go build -tags=${BUILD_TAG} -x -o exec ./main

# Final image — minimal
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/exec .
CMD ["/app/exec"]
