.PHONY: first build migrate adminserver gitserver worker migratedb flushdb omnibus test \
	docker-adminserver docker-gitserver docker-migratedb docker-worker \
	clean

first: build

build: adminserver gitserver migratedb worker omnibus

adminserver:
	go build github.com/abustany/moblog-cloud/cmd/adminserver

docker-adminserver:
	go mod vendor
	docker build -t moblog-cloud/adminserver:latest -f docker/adminserver.dockerfile .

gitserver:
	go build github.com/abustany/moblog-cloud/cmd/gitserver

docker-gitserver:
	go mod vendor
	docker build -t moblog-cloud/gitserver:latest -f docker/gitserver.dockerfile .

worker:
	go build github.com/abustany/moblog-cloud/cmd/worker

docker-worker:
	go mod vendor
	docker build -t moblog-cloud/worker:latest -f docker/worker.dockerfile .

migratedb: tools/go-bindata
	cd sql && ../tools/go-bindata -pkg sql .
	go build github.com/abustany/moblog-cloud/cmd/migratedb

docker-migratedb:
	go mod vendor
	docker build -t moblog-cloud/migratedb:latest -f docker/migratedb.dockerfile .

omnibus-adminui/index.html:
	rm -fr $(dir $@)
	mkdir -p $(dir $@)
	# Pipe through tee to avoid fancy colors and terminal escape sequences
	OUTPUT_DIR=$(shell cd $(dir $@) && pwd) PUBLIC_PATH="/admin/" yarn --non-interactive --cwd=../admin-ui run build | tee

omnibus: omnibus-adminui/index.html
	echo "package adminui" > omnibus-adminui/init.go
	cd omnibus-adminui && ../tools/go-bindata -pkg adminui ./...
	go build github.com/abustany/moblog-cloud/cmd/omnibus

migrate: tools/migrate-bin
	@sh -c "[ -z \"$${DB_URL}\" ] && echo Set DB_URL environment variable to the Postgres URL before running this command && exit 1 || true"
	./tools/migrate-bin -database "${DB_URL}" -path sql up

tools/migrate-bin:
	@echo "Downloading migrate"
	mkdir -p tools && \
	cd tools && \
	curl -L https://github.com/golang-migrate/migrate/releases/download/v4.4.0/migrate.linux-amd64.tar.gz | tar xz && \
	mv migrate.linux-amd64 migrate-bin

tools/go-bindata:
	mkdir -p tools && \
	cd tools && \
	go get github.com/jteeuwen/go-bindata/go-bindata && \
	go build github.com/jteeuwen/go-bindata/go-bindata

flushdb:
	@sh -c "[ -z \"$${DB_URL}\" ] && echo Set DB_URL environment variable to the Postgres URL before running this command && exit 1 || true"
	psql -f sql/flush_db.sql ${DB_URL}

test:
	which golangci-lint >/dev/null 2>&1 && golangci-lint run
	go test -count=1 ./...

	if [ -n "${DB_URL}" ]; then DB_URL="" go test -count=1 ./...; fi

clean:
	rm -rf migrate adminserver gitserver worker migratedb omnibus omnibus-adminui
