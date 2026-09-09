.PHONY: lint, test, check, up, clean, set-hook

test:
	go test ./... --skip=Integration

test-n:
	source .env.local && go test -v -count=1 -p=1 ./... -run $(name)

lint:
	golangci-lint run -v --skip-dirs=test

check: lint test

run:
	make set-hook
	make build && source .env.local && ./mx_news_bot

up:
	docker-compose up -d db-mx;
	sleep 3
	make migrations

run-d:
	sudo bash -c 'nohup ./mx_news_bot > output.log 2>&1 &'

kill:
	sudo pkill -f mx_news_bot

migrations:
	goose -dir infra/migrations postgres "host=localhost port=6444 user=mx dbname=mx password=mxpassword sslmode=disable" up

clean:
	docker-compose down --rmi all --volumes

build:
	go build -ldflags "-s -w" -o mx_news_bot ./cmd/mx

run-ng:
	# nohup ssh -R 80:localhost:8585 serveo.net > serveo_url.txt 2>&1 &
	ngrok http 8585

set-hook-t:
	NGROK_ADDR=$$(head -n 1 ./serveo_url.txt | rev | cut -c 2- | rev | awk '{print $$5}' | tr -d '\n'); \
	awk -v addr="$$NGROK_ADDR" '{if (/^export NGROK_ADDR=/) print "export NGROK_ADDR=\"" addr "\""; else print $$0}' .env.local > .env.tmp && mv .env.tmp .env.local; \
	source .env.local; \
	echo $$NGROK_ADDR; \
	echo $$APP_BOT_TOKEN; \
	curl -F "url=$$NGROK_ADDR" https://api.telegram.org/bot$$APP_BOT_TOKEN/setWebhook

set-hook:
	# curl -F "url=$NGROK_ADDR" https://api.telegram.org/bot$APP_BOT_TOKEN/setWebhook
	curl -F "url=$$NGROK_ADDR" https://api.telegram.org/bot$$APP_BOT_TOKEN/setWebhook

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-restart:
	docker-compose down
	docker-compose up --build -d

