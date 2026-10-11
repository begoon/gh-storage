set dotenv-load := true

dev:
    go run .

test:
    go test -race ./...

build:
    go build -o exe .

clean:
    rm -rf exe tmp

deploy:
    vercel deploy --prod
