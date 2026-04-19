BINARY=vpn-monitor
PORT?=8080
XUI_URL?=https://akvilon.nemesh-vpn.ru:808
XUI_USER?=Akvil0n
XUI_PASS?=Perfect10nizm

.PHONY: build run docker docker-run clean

build:
	go build -ldflags="-w -s" -o $(BINARY) .

run: build
	PORT=$(PORT) XUI_URL=$(XUI_URL) XUI_USER=$(XUI_USER) XUI_PASS=$(XUI_PASS) ./$(BINARY)

docker:
	docker build -t vpn-monitor .

docker-run:
	docker run -d \
		--name vpn-monitor \
		--restart unless-stopped \
		--network host \
		-e PORT=$(PORT) \
		-e XUI_URL=$(XUI_URL) \
		-e XUI_USER=$(XUI_USER) \
		-e XUI_PASS=$(XUI_PASS) \
		vpn-monitor

docker-compose-up:
	docker-compose up -d --build

clean:
	rm -f $(BINARY)

deps:
	go mod tidy
