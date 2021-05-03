.PHONY: start build
start:
	docker-compose pull
	docker-compose up
	open webapp/build/index.html

webapp/dist: webapp/src
	npm ci
	npm run build

build: webapp/dist
	make -C docker/byzcoin docker
	( cd webapp; ng build )
	rm -rf docker/webapp/dist
	cp -a webapp/dist/webapp docker/webapp/dist
	cp -a webapp/src/assets docker/webapp/dist
	docker build -t filesharing/webapp docker/webapp

docker_bc:
	docker run -ti -p7770-7777:7770-7777 filesharing/byzcoin
