FROM test-task-s3-build:latest as build

WORKDIR /usr/local/go/src/test_task_s3

COPY cmd/storage-service /usr/local/go/src/test_task_s3/cmd/storage-service
COPY internal /usr/local/go/src/test_task_s3/internal

RUN CGO_ENABLED=0 go build -o bin/storage-service ./cmd/storage-service/.

FROM test-task-s3-ubuntu:latest as deployable

COPY --from=build /usr/local/go/src/test_task_s3/bin/storage-service /local/bin/storage-service
