build:
	mkdir ./tmp
	go build -o ./tmp/truck

run: build
	./tmp/truck

release:
	mkdir -p ./tmp/release
	cp Dockerfile ./tmp/release/Dockerfile
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s" -installsuffix "release" -a -o ./tmp/release/truck_linux_amd64
	docker build -t jcoene/truck:latest ./tmp/release
	docker push jcoene/truck:latest
