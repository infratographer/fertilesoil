FROM cgr.dev/chainguard/go:1.20 as build-env

WORKDIR /go/src/fertilesoil

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/treeman main.go 

FROM cgr.dev/chainguard/static:latest

# `nonroot` coming from distroless
USER 65532:65532

COPY --from=build-env /go/bin/treeman /
ENTRYPOINT ["/treeman"]
