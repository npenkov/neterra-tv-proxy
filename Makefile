REPO=npenkov/neterra-tv-proxy
VER=1.0
ARCH=amd64
PLATFORM=linux

all : build docker push clear
.PHONY: push docker build all clean

build:
	GOOS=${PLATFORM} GOARCH=${ARCH} GOARM=7 go build -o ntvp main.go

docker:
	@echo "Building docker image"
	docker build -t ${REPO}:${VER} -t ${REPO}:latest .

push: 
	@echo "Pushing to dockerhub"
	docker push ${REPO}:${VER} 
	docker push ${REPO}:latest

clean:
	rm -rf ntvp