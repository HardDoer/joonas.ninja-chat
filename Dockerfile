FROM golang:alpine AS build

ARG DEPLOY_ENV=production
RUN apk add build-base

ADD src /opt/joonas.ninja-chat/src
ADD go.mod /opt/joonas.ninja-chat/src/go.mod
ADD go.sum /opt/joonas.ninja-chat/src/go.sum
ADD env/${DEPLOY_ENV}.env /opt/joonas.ninja-chat/app.env

WORKDIR /opt/joonas.ninja-chat/src
RUN go test
RUN go build -o chat

FROM alpine:latest
EXPOSE 80
WORKDIR /opt/joonas.ninja-chat
COPY --from=build /opt/joonas.ninja-chat/src/chat .
COPY --from=build /opt/joonas.ninja-chat/app.env .

CMD ["./chat"]


