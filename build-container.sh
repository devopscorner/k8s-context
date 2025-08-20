#!/usr/bin/env sh
# -----------------------------------------------------------------------------
#  Build Container Image k8s-context
# -----------------------------------------------------------------------------
#  Author     : Dwi Fahni Denni
#  License    : Apache v2
# -----------------------------------------------------------------------------
set -e

export AWS_REGION="ap-southeast-3"
export AWS_ACCOUNT_ID="123456789012"
export CI_PROJECT_REGISTRY="docker.io"
export CI_PROJECT_REGISTRY_ECR="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/devopscorner/devops"
export CI_PROJECT_PATH="devopscorner"
export CI_PROJECT_NAME="k8s-context"

export IMAGE="$CI_PROJECT_REGISTRY/$CI_PROJECT_PATH/$CI_PROJECT_NAME"
export IMAGE_ECR="$CI_PROJECT_REGISTRY_ECR/$CI_PROJECT_NAME"

PATH_FOLDER=`pwd`

TAG_VERSION="1.24.5"
TAG_ID=`echo $(date '+%Y%m%d')`

LINE_PRINT="======================================================="

login_ecr() {
  echo "============="
  echo "  Login ECR  "
  echo "============="
  PASSWORD=`aws ecr get-login-password --region $AWS_REGION`
  echo $PASSWORD | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com
  echo '- DONE -'
  echo ''
}

docker_builder_go() {
   docker build --no-cache -f Dockerfile -t $IMAGE:latest .
}

docker_golang() {
  echo $LINE_PRINT
  echo " DOCKER BUILDER GOLANG $TAG_VERSION-$TAG_ID "
  echo $LINE_PRINT
  echo "docker build --no-cache -f Dockerfile -t $IMAGE:latest ."
  docker_builder_go
  echo ' - DONE -'
  echo ''
}

tag_alpine() {
   echo $LINE_PRINT
   echo " DOCKER TAG ALPINE $TAG_VERSION-$TAG_ID "
   echo $LINE_PRINT
   echo "docker tag $IMAGE:latest $IMAGE:latest-$TAG_ID
docker tag $IMAGE:latest $IMAGE:relase-$TAG_ID
docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-alpine
docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-alpine-$TAG_ID
docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-devopscorner
docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-devopscorner-alpine
docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-devopscorner-alpine-$TAG_ID"

   docker tag $IMAGE:latest $IMAGE:latest-$TAG_ID
   docker tag $IMAGE:latest $IMAGE:release-$TAG_ID
   docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-alpine
   docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-alpine-$TAG_ID
   docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-devopscorner
   docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-devopscorner-alpine
   docker tag $IMAGE:latest $IMAGE:$TAG_VERSION-devopscorner-alpine-$TAG_ID
   echo ' - DONE -'
   echo ''
}

tag_alpine_ecr() {
   echo $LINE_PRINT
   echo " ECR TAG ALPINE $TAG_VERSION-$TAG_ID "
   echo $LINE_PRINT
   echo "docker tag $IMAGE:latest $IMAGE_ECR:latest-$TAG_ID
docker tag $IMAGE:latest $IMAGE_ECR:release-$TAG_ID
docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-alpine
docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-alpine-$TAG_ID
docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-devopscorner
docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-devopscorner-alpine
docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-devopscorner-alpine-$TAG_ID"

   docker tag $IMAGE:latest $IMAGE_ECR:latest-$TAG_ID
   docker tag $IMAGE:latest $IMAGE_ECR:release-$TAG_ID
   docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-alpine
   docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-alpine-$TAG_ID
   docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-devopscorner
   docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-devopscorner-alpine
   docker tag $IMAGE:latest $IMAGE_ECR:$TAG_VERSION-devopscorner-alpine-$TAG_ID
   echo ' - DONE -'
   echo ''
}

push_alpine() {
  echo $LINE_PRINT
  echo " DOCKER PUSH ALPINE $TAG_VERSION-$TAG_ID "
  echo $LINE_PRINT
  PUSH_IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep alpine)
  for IMG in $PUSH_IMAGES; do
    echo "Docker Push => $IMG"
    echo ">> docker push $IMG"
    docker push $IMG
    echo '- DONE -'
    echo ''
  done
}

push_alpine_ecr() {
  echo $LINE_PRINT
  echo " ECR PUSH ALPINE $TAG_VERSION-$TAG_ID "
  echo $LINE_PRINT
  PUSH_IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep $AWS_ACCOUNT_ID)
  for IMG in $PUSH_IMAGES; do
    echo "Docker Push => $IMG"
    echo ">> docker push $IMG"
    docker push $IMG
    echo '- DONE -'
    echo ''
  done
}

push_devopscorner() {
  echo $LINE_PRINT
  echo " DOCKER PUSH DEVOPSCORNER $TAG_VERSION-$TAG_ID "
  echo $LINE_PRINT
  PUSH_IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "$CI_PROJECT_PATH/$CI_PROJECT_NAME")
  for IMG in $PUSH_IMAGES; do
    echo "Docker Push => $IMG"
    echo ">> docker push $IMG"
    docker push $IMG
    echo '- DONE -'
    echo ''
  done
}

docker_build() {
  ## Builder GO ##
  docker_golang
}

docker_tag() {
  ## ALPINE Tags ##
  tag_alpine
  tag_alpine_ecr
}

docker_push(){
  ## ALPINE Tags ##
  # push_alpine

  ## DevOpsCorner Tags ##
  push_devopscorner
}

ecr_push(){
  ## AWS_ACCOUNT_ID Tags ##
  login_ecr
  push_alpine_ecr
}

docker_clean() {
    echo "Cleanup Unknown Tags"
    echo "docker images -a | grep none | awk '{ print $3; }' | xargs docker rmi"
    docker images -a | grep none | awk '{ print $3; }' | xargs docker rmi
    echo '- DONE -'
    echo ''
}

main() {
  docker_build
  docker_tag
  docker_clean
  docker_push
  # ecr_push
  echo '-- ALL DONE --'
}

### START HERE ###
main