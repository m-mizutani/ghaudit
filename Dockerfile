FROM golang:1.17 AS build-go
COPY . /src
WORKDIR /src
RUN go build -o ghaudit .

FROM gcr.io/distroless/base
COPY --from=build-go /src/ghaudit /ghaudit
WORKDIR /
ENTRYPOINT ["/ghaudit"]
