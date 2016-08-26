.PHONY: build clean push

build:
	CGO_ENABLED=0 go build -installsuffix nocgo -o autoscale
	docker build -t 192.168.122.110/csphere/autoscale .

push: build
	docker push 192.168.122.110/csphere/autoscale

clean:
	rm -f autoscale
