
save-godeps:
	godep save github.com/crackcomm/cloudflare/cf

cloudflare-build:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o ./dist/cf ./cf/main.go

install:
	go install github.com/crackcomm/cloudflare/cf

dist: cloudflare-build

clean:
	rm -rf dist
