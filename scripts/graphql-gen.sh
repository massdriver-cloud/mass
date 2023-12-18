#! /bin/bash
set -e

START=$(echo "$PWD")

cd ../massdriver || exit

if ! docker inspect massdriver-massdriver:latest > /dev/null 2>&1; then
    docker-compose build
fi

docker run -it --rm -v .:/app massdriver-massdriver:latest /bin/bash -c "mix deps.get;mix absinthe.schema.sdl"

mv schema.graphql ../mass/pkg/api/schema.graphql

cd $START
