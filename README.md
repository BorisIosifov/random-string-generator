# random-string-generator

Task description [here](https://gist.github.com/ciju/40afaa21a4b9be998955e84570a057c0). Done all required and optional features.

### Using:
```sh
go build -o random-string-generator generator.go
./random-string-generator -re '(1[0-2]|0[1-9])(:[0-5][0-9]){2} (A|P)M' -n 10
```

### Using with docker:
```sh
docker-compose build
docker-compose up -d
docker-compose run random-string-generator bash
./random-string-generator -re "(1[0-2]|0[1-9])(:[0-5][0-9]){2} (A|P)M" -n 10
```