# Golang Developer Test

## Description

You decided to create a competitor for Amazon S3, and you know how to create a better file storage service.

Server A receives a file, you need to cut it to ~6 equal parts and save to storage servers Bn (n ≥ 6).

On REST query you need to get file parts from Bn servers, glue it back and return it.

## What we have

  - One REST server
  - n storage servers

## Restrictions

  - Create a service and a test module, that checks its work
  - Storage servers can be added at any time, but can't be removed
  - Storage servers should be filled equally

## Good to have

  - Clean and readable code
  - Comments

With this test we want to understand your way of thinking and your ability to find an approach to solving problems.