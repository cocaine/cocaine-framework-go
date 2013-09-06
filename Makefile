all:
	go install cocaine
	go build cocaine
	go build testapp
	go build testcli
