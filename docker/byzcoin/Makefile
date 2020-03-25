CONTAINER = byzcoin
IMAGE_NAME = filesharing/$(CONTAINER)
WEBAPP_ASSET := ../../webapp/src/assets/config.toml

all: docker

.PHONY: cothority-pull conode newdb docker docker_run docker_wait docker_stop docker_start docker_clean clean

.DEFAULT_GOAL := docker_run

utils:
	@echo "Compiling binaries"; \
	go build -o build/conode; \
	go build -o build/bcadmin ../../pkg/byzcoin/bcadmin

docker_wait:
	@for port in $$( seq 7771 2 7777 ); do \
	  echo "Waiting for localhost:$$port"; \
	  until curl -s http://localhost:$$port/ok > /dev/null; do sleep .2; done; \
	done

newdb: utils
	rm -f build/conodes/*db;
	@echo "Starting nodes"; \
	( cd build; COTHORITY_ALLOW_INSECURE_ADMIN=true ./run_nodes.sh -v 2 -t -n 4 -s -d conodes > /dev/null ) & \
	make docker_wait; \
	echo "Creating new ByzCoin and printing out important variables"; echo; \
	rm -f *.cfg; \
	export DEBUG_COLOR=false; \
	( ./build/bcadmin -c . create build/conodes/public.toml --interval 500ms; \
	  ./build/bcadmin latest --bc bc-*; \
	  ./build/bcadmin key -print key-* ) | egrep "^(ByzCoin|Admin|Private|Creating Spawner)" | tee variables.txt; \
	echo; echo "Shutting down conodes"; pkill -x conode; pkill -f run_nodes.sh; \
	rm -f conodes/running

$(WEBAPP_ASSET): variables.txt build/conodes/public.toml
	 echo "ByzCoinID = \"$$( grep ByzCoinID $< | sed -e 's/.* //' )\"" > $@
	 echo >> $@
	 cat $(lastword $^) >> $@

newconfig: newdb $(WEBAPP_ASSET)

# Use this target to build from local source
docker: build/Dockerfile
	@export GO111MODULE=on GOOS=linux GOARCH=amd64; \
	go build -o build/conode
	docker build -t $(IMAGE_NAME) -f build/Dockerfile build

clean:
	rm -rf build/conode
