FROM golang:1-alpine
ENV GOBIN=/bin
WORKDIR /src/app
# [ineffectual] this allows skaffold to use buildArgs to compile without optimizations "all=-N -l" (https://skaffold.dev/docs/workflows/debug/)
ARG GCFLAGS="-c 1"

COPY ./go.mod go.sum ./
RUN go mod download
COPY . .

RUN go install -gcflags "$GCFLAGS"