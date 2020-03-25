.PHONY: start build
start:
	docker-compose pull
	docker-compose up
	open webapp/build/index.html

webapp/dist: webapp/src
	npm build

build: webapp/dist
	make -C docker/byzcoin docker
	( cd webapp; ng build )
	rm -rf docker/webapp/dist
	cp -a webapp/dist/webapp docker/webapp/dist
	cp -a webapp/src/assets docker/webapp/dist
	docker build -t filesharing/webapp docker/webapp
