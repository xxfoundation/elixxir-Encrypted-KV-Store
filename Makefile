.PHONY: test clean

test:
	mkdir -p testdata
	go-acc --covermode atomic --output testdata/coverage.out ./... -- -v
	go tool cover -func=testdata/coverage.out
	go tool cover -html=testdata/coverage.out -o testdata/coverage.html
	go tool cover -func=testdata/coverage.out | grep "total:" | awk '{print $3}' | sed 's/\%//g' > testdata/coverage-percentage.txt

clean:
	git clean -ffdx
