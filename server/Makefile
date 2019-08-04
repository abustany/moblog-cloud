.PHONY: first build migrate adminserver gitserver worker migratedb flushdb test

first: build

build: adminserver gitserver migratedb worker

adminserver:
	go build github.com/abustany/moblog-cloud/cmd/adminserver

gitserver:
	go build github.com/abustany/moblog-cloud/cmd/gitserver

worker:
	go build github.com/abustany/moblog-cloud/cmd/worker

migratedb: tools/go-bindata
	cd sql && ../tools/go-bindata -pkg sql .
	go build github.com/abustany/moblog-cloud/cmd/migratedb

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
	go test -count=1 ./...

	@echo "Previous test run was against a real database, running again with MemoryUserStore"
	[ -n "${DB_URL}" ] && DB_URL="" go test -count=1 ./...
