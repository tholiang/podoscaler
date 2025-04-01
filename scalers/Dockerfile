FROM golang:alpine

ARG SRC_DIR

RUN mkdir /app
COPY ${SRC_DIR} /app/${SRC_DIR}
COPY ./util /app/util
COPY ./go.mod /app
COPY ./go.sum /app
WORKDIR /app
RUN ls
RUN go mod tidy
RUN go build -o exec ./${SRC_DIR}
CMD ["/app/exec"]