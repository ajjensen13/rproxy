FROM golang:alpine as build
ARG GCFLAGS="-c 1"
ENV CGO_ENABLED=0
ENV GO111MODULE=on
WORKDIR /src/app
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN go test -json ./...
RUN go build -i -gcflags "$GCFLAGS" -o /bin/rproxy ./cmd/rproxy

FROM alpine as run
ARG MAIN=rproxy
COPY --from=build /bin/rproxy /bin/rproxy
CMD [ "/bin/rproxy" ]