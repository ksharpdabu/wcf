@echo off

echo %GOPATH%
call :build_exe linux amd64 ""
call :build_exe windows amd64 ".exe"
call :build_exe darwin amd64 ""
goto:eof

:build_exe
set CGO_ENABLED=0
set GOOS=%~1
set GOARCH=%~2
set EXT=%3

echo Build target:%~1, arch:%~2

set cur=%cd%
cd src/wcf/cmd/local/
echo build %~1 local
go build -o local_%~1_%~2%~3
cd %cur%
cd src/wcf/cmd/server
echo build %~1 server
go build -o server_%~1_%~2%~3
cd %cur%
goto:eof



pause
