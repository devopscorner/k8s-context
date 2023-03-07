### Builder Go ###
FROM golang:1.19.5-alpine3.17 as builder-go

WORKDIR /go/src/app
ENV GIN_MODE=release
ENV GOPATH=/go

RUN apk add --no-cache \
        build-base \
        git \
        curl \
        make \
        bash

COPY src /go/src/app

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    cd /go/src/app && \
        go build -mod=readonly -ldflags="-s -w" -o goapp


### Builder Python ###
FROM python:3.10.10-alpine${ALPINE_VERSION:-3.17} as builder-python

ARG AWS_CLI_VERSION=2.11.0
RUN apk add --no-cache git unzip groff build-base libffi-dev cmake
RUN git clone --single-branch --depth 1 -b ${AWS_CLI_VERSION} https://github.com/aws/aws-cli.git

WORKDIR aws-cli
RUN python -m venv venv
RUN . venv/bin/activate
RUN scripts/installers/make-exe
RUN unzip -q dist/awscli-exe.zip
RUN aws/install --bin-dir /aws-cli-bin
RUN /aws-cli-bin/aws --version

# reduce image size: remove autocomplete and examples
RUN rm -rf \
    /usr/local/aws-cli/v2/current/dist/aws_completer \
    /usr/local/aws-cli/v2/current/dist/awscli/data/ac.index \
    /usr/local/aws-cli/v2/current/dist/awscli/examples
RUN find /usr/local/aws-cli/v2/current/dist/awscli/data -name completions-1*.json -delete
RUN find /usr/local/aws-cli/v2/current/dist/awscli/botocore/data -name examples-1.json -delete


### Binary ###
# FROM golang:1.19.5-alpine3.17
FROM nginx:${NGINX_VERSION:-1.23-alpine}

ARG BUILD_DATE
ARG BUILD_VERSION
ARG GIT_COMMIT
ARG GIT_URL

ENV VENDOR="DevOpsCornerId"
ENV AUTHOR="DevOpsCorner.id <support@devopscorner.id>"
ENV IMG_NAME="alpine"
ENV IMG_VERSION="3.17"
ENV IMG_DESC="Docker GO App Alpine 3.17"
ENV IMG_ARCH="amd64/x86_64"

ENV ALPINE_VERSION="3.17"

LABEL maintainer="$AUTHOR" \
        architecture="$IMG_ARCH" \
        ubuntu-version="$ALPINE_VERSION" \
        org.label-schema.build-date="$BUILD_DATE" \
        org.label-schema.name="$IMG_NAME" \
        org.label-schema.description="$IMG_DESC" \
        org.label-schema.vcs-ref="$GIT_COMMIT" \
        org.label-schema.vcs-url="$GIT_URL" \
        org.label-schema.vendor="$VENDOR" \
        org.label-schema.version="$BUILD_VERSION" \
        org.label-schema.schema-version="$IMG_VERSION" \
        org.opencontainers.image.authors="$AUTHOR" \
        org.opencontainers.image.description="$IMG_DESC" \
        org.opencontainers.image.vendor="$VENDOR" \
        org.opencontainers.image.version="$IMG_VERSION" \
        org.opencontainers.image.revision="$GIT_COMMIT" \
        org.opencontainers.image.created="$BUILD_DATE" \
        fr.hbis.docker.base.build-date="$BUILD_DATE" \
        fr.hbis.docker.base.name="$IMG_NAME" \
        fr.hbis.docker.base.vendor="$VENDOR" \
        fr.hbis.docker.base.version="$BUILD_VERSION"

ENV GIN_MODE=release
COPY --from=alpine/k8s:1.26.2 /usr/bin/ /usr/local/bin/
COPY --from=builder-python /usr/local/aws-cli/ /usr/local/aws-cli/
COPY --from=builder-python /aws-cli-bin/ /usr/local/bin/
COPY --from=builder-go /go/src/app/goapp /usr/local/bin/k8s-context

EXPOSE 22 80 443

STOPSIGNAL SIGQUIT

CMD ["nginx", "-g", "daemon off;"]
