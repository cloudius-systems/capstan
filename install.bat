set package=github.com/cloudius-systems/capstan

:: Clone github repo into $GOPATH/src/$package, but don't install yet
go get -d %package%

:: Calculate version
cd /d %GOPATH%/src/%package%
for /f %%i in ('git describe --tags') do set version=%%i

:: Clean
rm %GOPATH%\bin\capstan 2> nul

:: Install with VERSION string properly set
go install -ldflags "-X main.VERSION=%version% " %package%
