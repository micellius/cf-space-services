# Build the project
all: clean build install

clean:
	rm -f cf-space-services

build:
	go build  

install:
	cf install-plugin -f cf-space-services	

release:
	rm -rf dist 
	mkdir dist
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -gcflags "all=-trimpath=$GOPATH" -o dist/cf-space-services-darwin-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -gcflags "all=-trimpath=$GOPATH" -o dist/cf-space-services-linux-amd64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -gcflags "all=-trimpath=$GOPATH" -o dist/cf-space-services-amd64.exe 
