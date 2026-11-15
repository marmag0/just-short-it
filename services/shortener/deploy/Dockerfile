FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY services/shortener ./services/shortener
RUN CGO_ENABLED=0 go build -o /shortener ./services/shortener/cmd

FROM scratch
ARG SHORTENER_PORT
ENV SHORTENER_PORT=${SHORTENER_PORT}
COPY --from=build /shortener /shortener
USER 65532:65532
EXPOSE ${SHORTENER_PORT}
ENTRYPOINT ["/shortener"]
