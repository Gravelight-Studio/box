@echo off
REM Build script for Box CLI on Windows
REM Reads version from root VERSION file and injects it via ldflags

setlocal enabledelayedexpansion

REM Get version from VERSION file
set VERSION_FILE=%~dp0..\VERSION
for /f "usebackq delims=" %%a in ("%VERSION_FILE%") do set VERSION=%%a

echo Building Box CLI v%VERSION%...

cd /d %~dp0cmd\box

REM Build with version injected
go build -ldflags "-X main.version=%VERSION%" -o %~dp0..\bin\box.exe

echo ✓ Built bin\box.exe
echo ✓ Version: %VERSION%
