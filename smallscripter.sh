#!/usr/bin/env bash
#Smallscripter
cloud=""
if [ "$1" != "" ]
then
  cdir="$1"
  cloud="-$1"
fi
DOCKER_REGISTRY=registry-gitlab.teuto.net
DOCKER_IMAGE=teutonetes
DOCKER_IMAGE_TAG=latest
IMAGENAME=$DOCKER_REGISTRY/technik/$DOCKER_IMAGE:$DOCKER_IMAGE_TAG
DIR=/home/$(whoami)/teutonetes/cluster

## Name of Alias
ALIASNAME=teutonetes$cloud

echo "If you don't have the image: $IMAGENAME - do a docker pull image command: \"docker pull $DOCKER_REGISTRY/technik/$DOCKER_IMAGE:$DOCKER_IMAGE_TAG\"."
echo "Setting alias $ALIASNAME."
alias $ALIASNAME="docker run -ti --rm -v $DIR:/root/cluster/ -v ~/.kube:/root/.kube/ -w /root/ -e UID=$(id -u) -e GID=$(id -g) --env-file $DIR/$cdir/.config $IMAGENAME"

echo "#########################################################################"
echo "Script done. To continue the installation:"
echo "Use these commands in order (be sure to use quotation marks if your cluster name contains whitespaces:"
echo " \" $ALIASNAME deploy $cdir \" "
