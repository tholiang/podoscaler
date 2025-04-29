FROM golang:alpine

ARG SRC_DIR

RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go build -o dummy .
CMD ["/app/dummy"]