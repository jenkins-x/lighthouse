CHART_REPO := http://jenkins-x-chartmuseum:8080
CURRENT=$(pwd)
NAME := lighthouse
OS := $(shell uname)

build: clean
	rm -rf requirements.lock
	helm dependency build
	helm lint

install: clean build
	helm install . --name ${NAME}

upgrade: clean build
	helm upgrade ${NAME} .

delete:
	helm delete --purge ${NAME}

clean:
	rm -rf charts
	rm -rf ${NAME}*.tgz

.PHONY: snapshot
snapshot: update-version release

release: clean
	helm dependency build
	helm lint
	helm init --client-only
	helm package .
	curl --fail -u $(CHARTMUSEUM_USER):$(CHARTMUSEUM_PASS) --data-binary "@$(NAME)-$(shell sed -n 's/^version: //p' Chart.yaml).tgz" $(CHART_REPO)/api/charts
	rm -rf ${NAME}*.tgz%

update-version:
ifeq ($(OS),Darwin)
	sed -i "" -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i "" -e "s/tag:.*/tag: $(VERSION)/" values.yaml
else ifeq ($(OS),Linux)
	sed -i -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i -e "s/tag:.*/tag: $(VERSION)/" values.yaml
else
	echo "platfrom $(OS) not supported to tag with"
	exit -1
endif

tag: update-version
	git add --all
	git commit -m "release $(VERSION)" --allow-empty # if first release then no verion update is performed
	git tag -fa v$(VERSION) -m "Release version $(VERSION)"
	git push origin v$(VERSION)

delete-from-chartmuseum:
	curl --fail -u $(CHARTMUSEUM_USER):$(CHARTMUSEUM_PASS) -X DELETE $(CHART_REPO)/api/charts/$(NAME)/$(VERSION)
