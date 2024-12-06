#!/bin/sh
PROJECT_NAME="sshesame-mongo-env"
case "$1" in
  "start"|"up")
    docker-compose -p $PROJECT_NAME up -d;;
  "stop"|"down")
    docker-compose down;;
  "restart")
    docker-compose down
    docker-compose up -d
   ;;
  "ps")
    docker-compose ps;;
#  "exec")
#    docker-compose exec server $2:@;;
#  "bash")
#    docker-compose exec server bash;;
  "default")
    echo "Hello World!";;
esac