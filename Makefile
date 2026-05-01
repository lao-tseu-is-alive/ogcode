.PHONY: dev build clean

dev:
	@echo "Starting Go server on :8080..."
	go run . serve --port 8080

dev-web:
	@echo "Starting Vite dev server on :5173..."
	cd web && npm run dev

build-web:
	cd web && npm install --legacy-peer-deps --cache /tmp/npm-cache && npm run build

build-server:
	CGO_ENABLED=0 go build -o ogcode .

build: build-web build-server
	@echo "Build complete: ./ogcode"

install: build
	cp ogcode /Users/admin/.local/bin/ogcode
	@echo "Installed to /Users/admin/.local/bin/ogcode"

clean:
	rm -f ogcode
	rm -rf web/dist web/node_modules web/.solid
	rm -rf .ogcode

test:
	go test ./...