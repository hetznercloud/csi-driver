# Changes

## master

- Fix usage of `Aborted` error code, which leads to an increasing CPU usage

## v1.2.1

- Add missing RBAC rules required for newer k8s version

## v1.2.0

- Implement volume resizing
- Implement volume statistics

## v1.1.5

- Revert fix from v1.1.2 to retry attach/detach when server is locked

## v1.1.4

- Respect minimum volume size of 10 GB

## v1.1.3

- Detach volumes before deleting them

## v1.1.2

- Fix error handling for attaching/detaching volumes in case server is locked

## v1.1.1

- Improve logging

## v1.1.0

- Implement topology awareness (supporting nodes and volumes in different locations)

## v1.0.0

- Initial release
