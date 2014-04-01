set package=github.com/cloudius-systems/capstan/capstan
for /f %%i in ('git describe --tags ') do set version=%%i
go get %package%
go install -ldflags "-X main.VERSION %version% " %package%
