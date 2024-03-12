FROM test-task-s3-build:latest as build

WORKDIR /usr/local/go/src/test_task_s3

COPY cmd/rest-service /usr/local/go/src/test_task_s3/cmd/rest-service
COPY internal /usr/local/go/src/test_task_s3/internal

RUN go build -o bin/rest-service ./cmd/rest-service/.

FROM test-task-s3-ubuntu:latest as deployable

COPY --from=build /usr/local/go/src/test_task_s3/bin/rest-service /local/bin/rest-service

ENTRYPOINT /local/bin/rest-service




