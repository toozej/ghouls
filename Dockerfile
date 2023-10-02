# setup project and deps
FROM golang:1.21-bullseye AS init

WORKDIR /go/ghouls/

COPY go.mod* go.sum* ./
RUN go mod download

COPY . ./

FROM init as vet
RUN go vet ./...

# run tests
FROM init as test
RUN go test -coverprofile c.out -v ./...

# build binary
FROM init as build
ARG LDFLAGS

RUN CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" ./cmd/ghouls/

# runtime image
FROM scratch
# Copy our static executable.
COPY --from=build /go/ghouls/ghouls /go/bin/ghouls
# Expose port for publishing as web service
EXPOSE 8080
# Run the binary.
ENTRYPOINT ["/go/bin/ghouls"]
