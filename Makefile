.PHONY: test clean

test:
	mkdir -p testdata
	go-acc --covermode atomic --output testdata/coverage.out ./... -- -v -race
	go tool cover -func=testdata/coverage.out
	go tool cover -html=testdata/coverage.out -o testdata/coverage.html

clean:
	git clean -ffdx
