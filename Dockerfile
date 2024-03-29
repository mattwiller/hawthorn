FROM golang:1.21-alpine AS build

ARG UMLS_API_KEY

WORKDIR /app
COPY . .

RUN apk add --no-cache make curl
RUN make build

FROM scratch
WORKDIR /

COPY --from=build /app/umls.db /
COPY --from=build /app/hawthorn /

ENTRYPOINT [ "/hawthorn" ]
EXPOSE 29927
