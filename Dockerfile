FROM golang:1.13
EXPOSE 80

ARG DEPLOY_ENV=production

ADD src /opt/joonas.ninja-chat/src
ADD go.mod /opt/joonas.ninja-chat/src/go.mod
ADD go.sum /opt/joonas.ninja-chat/src/go.sum
ADD env/${DEPLOY_ENV}.env /opt/joonas.ninja-chat/src/app.env

WORKDIR /opt/joonas.ninja-chat/src
RUN go build -o chat
RUN mv ./chat /opt/joonas.ninja-chat
WORKDIR /opt/joonas.ninja-chat
RUN rm -R -rf ./src


CMD ["./chat "]


