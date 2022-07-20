FROM golang:1.17-alpine

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY *.go ./

RUN go build -o /filesaver-storage

EXPOSE 33333

CMD [ "/filesaver-storage" ]