

```
# install dependencies
go mod init
go mod tidy

# execute script
go run main.go --error-message="Back-off pulling image"
go run main.go --error-message="Back-off pulling image" --namespace default
go run main.go --error-message="Back-off pulling image" --namespace default --polling-interval 30

# build docker image
task build
```