@echo off
setlocal

set _EXITCODE=0

REM If no target is provided, default to test.
if [%1]==[] goto test

set _TARGETS=generate,test,testacc,testrace,updatedeps,vet

REM Run target.
for %%a in (%_TARGETS%) do (if x%1==x%%a goto %%a)
goto usage

REM generate runs `go generate` to build the dynamically generated
REM source files.
:generate
	go generate ./...
	goto :eof

REM test runs the unit tests and vets the code.
:test
	call :testsetup
	go test %_TEST% %TESTARGS% -timeout=30s -parallel=4
	call :setMaxExitCode %ERRORLEVEL%
	echo.
	goto vet

REM testacc runs acceptance tests.
:testacc
	call :testsetup
	if x%_TEST% == x./... goto testacc_fail
	if x%_TEST% == x.\... goto testacc_fail
	set TF_ACC=1
	go test %_TEST% -v %TESTARGS% -timeout 45m
	exit /b %ERRORLEVEL%
:testacc_fail
	echo ERROR: Set TEST to a specific package.
	exit /b 1

REM testrace runs the race checker.
:testrace
	call :testsetup
	go test -race %_TEST% %TESTARGS%
	exit /b %ERRORLEVEL%

REM testsetup calls `go generate` and defines the variables TF_ACC
REM and _TEST. TF_ACC is always cleared. _TEST defaults to the value
REM of the TEST environment variable, provided that TEST is defined,
REM otherwise _TEST it is set to "./...".
:testsetup
	call :generate
	set TF_ACC=
	set _TEST=./...
	if defined TEST set _TEST=%TEST%
	goto :eof

REM updatedeps installs all the dependencies that Terraform needs to
REM run and build.
:updatedeps
	for /f "tokens=*" %%c in ('git symbolic-ref --short HEAD 2^>nul ^|^| git rev-parse HEAD') do set _REF=%%c
	go get -u github.com/mitchellh/gox
	go get -u golang.org/x/tools/cmd/stringer
	go get -f -u -v ./...
	call :setMaxExitCode %ERRORLEVEL%
	git checkout %_REF% 1>nul 2>&1
	exit /b %_EXITCODE%

REM vet runs the Go source code static analysis tool `vet` to find
REM any common errors.
:vet
	set _VETARGS=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr
	if defined VETARGS set _VETARGS=%VETARGS%

	go tool vet 2>nul
	if %ERRORLEVEL% equ 3 go get golang.org/x/tools/cmd/vet
	
	echo go tool vet %_VETARGS% .
	go tool vet %_VETARGS% .
	set _vetExitCode=%ERRORLEVEL%
	if %_vetExitCode% equ 0 exit /b %_EXITCODE%
	call :setMaxExitCode %_vetExitCode%
	echo.
	echo Vet found suspicious constructs. Please check the reported constructs
	echo and fix them if necessary before submitting the code for reviewal.
	exit /b %_EXITCODE%

:setMaxExitCode
	if %1 gtr %_EXITCODE% set _EXITCODE=%1
	goto :eof

:usage
	echo usage: make [target]
	echo.
	echo target is in {%_TARGETS%}.
	echo target defaults to test if none is provided.
	exit /b 2
	goto :eof
