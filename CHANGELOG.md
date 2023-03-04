# Kubernetes Change Context (k8s-context)

Customize Kubernetes Change Context (KUBECONFIG)

![goreport](https://goreportcard.com/badge/github.com/devopscorner/k8s-context)
![all contributors](https://img.shields.io/github/contributors/devopscorner/k8s-context)
![tags](https://img.shields.io/github/v/tag/devopscorner/k8s-context?sort=semver)
[![docker pulls](https://img.shields.io/docker/pulls/devopscorner/k8s-context.svg)](https://hub.docker.com/r/devopscorner/k8s-context/)
![download all](https://img.shields.io/github/downloads/devopscorner/k8s-context/total.svg)
![view](https://views.whatilearened.today/views/github/devopscorner/k8s-context.svg)
![clone](https://img.shields.io/badge/dynamic/json?color=success&label=clone&query=count&url=https://github.com/devopscorner/k8s-context/blob/master/clone.json?raw=True&logo=github)
![issues](https://img.shields.io/github/issues/devopscorner/k8s-context)
![pull requests](https://img.shields.io/github/issues-pr/devopscorner/k8s-context)
![forks](https://img.shields.io/github/forks/devopscorner/k8s-context)
![stars](https://img.shields.io/github/stars/devopscorner/k8s-context)
[![license](https://img.shields.io/github/license/devopscorner/k8s-context)](https://img.shields.io/github/license/devopscorner/k8s-context)

## Available Tags

### Alpine

| Image name | Size |
|------------|------|
| `devopscorner/k8s-context:latest` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/latest.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=latest) |
| `devopscorner/k8s-context:alpine` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/alpine.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=alpine) |
| `devopscorner/k8s-context:alpine-latest` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/alpine-latest.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=alpine-latest) |
| `devopscorner/k8s-context:alpine-3.15` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/alpine-3.15.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=alpine-3.15) |
| `devopscorner/k8s-context:go1.19-alpine3.15` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/go1.19-alpine3.15.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=go1.19-alpine3.15) |
| `devopscorner/k8s-context:go1.19.3-alpine3.15` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/go1.19.3-alpine3.15.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=go1.19.3-alpine3.15) |
| `devopscorner/k8s-context:alpine-3.16` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/alpine-3.16.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=alpine-3.16) |
| `devopscorner/k8s-context:go1.19-alpine3.16` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/go1.19-alpine3.16.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=go1.19-alpine3.16) |
| `devopscorner/k8s-context:go1.19.5-alpine3.16` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/go1.19.5-alpine3.16.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=go1.19.5-alpine3.16) |
| `devopscorner/k8s-context:alpine-3.17` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/alpine-3.17.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=alpine-3.17) |
| `devopscorner/k8s-context:go1.19-alpine3.17` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/go1.19-alpine3.17.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=go1.19-alpine3.17) |
| `devopscorner/k8s-context:go1.19.5-alpine3.17` | [![docker image size](https://img.shields.io/docker/image-size/devopscorner/k8s-context/go1.19.5-alpine3.17.svg?label=Image%20size&logo=docker)](https://hub.docker.com/repository/docker/devopscorner/k8s-context/tags?page=1&ordering=last_updated&name=go1.19.5-alpine3.17) |


---

### version 1.0

- First deployment GO Apps
- Script build image
- Script ecr-tag & ecr-push
- Helm deployment values
- Upgrade gomod, using GO `1.17`
- Dockerfile using `golang:1.17-alpine3.15`
