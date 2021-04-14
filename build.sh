#format program
gofmt -s -w ./goudp
#fix code
go tool fix ./goudp
#reports suspicious constructs
go vet ./goudp

#test
go test ./goudp
#build
go install ./goudp #remember to copy the goudp.exe to GO_PATH
#to see help
goudp -help