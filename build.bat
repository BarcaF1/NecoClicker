@echo off
setlocal EnableDelayedExpansion

REM ============================================================================
REM NecoClicker — full release build
REM   - frontend (pnpm install + vite build)
REM   - wails build (production, ldflags=-s -w)
REM   - UPX compression
REM   - NSIS installer
REM Output:
REM   build\bin\NecoClicker.exe                 (~3.4 MB single-file portable)
REM   build\bin\NecoClicker-amd64-installer.exe (~6.5 MB NSIS installer)
REM ============================================================================

where wails >nul 2>nul || (echo [!] wails CLI not in PATH ^(go install github.com/wailsapp/wails/v2/cmd/wails@latest^) & exit /b 1)
where gcc   >nul 2>nul || (echo [!] gcc not in PATH ^(install MinGW-w64^) & exit /b 1)
where pnpm  >nul 2>nul || (echo [!] pnpm not in PATH ^(npm i -g pnpm^) & exit /b 1)

echo [*] Building (Wails + NSIS)...
wails build -clean -nsis -platform windows/amd64 -ldflags "-s -w" || exit /b 1

REM Compress portable exe with UPX if available
where upx >nul 2>nul
if not errorlevel 1 (
  echo [*] UPX compressing portable exe...
  upx --best --lzma build\bin\NecoClicker.exe
) else (
  echo [i] upx not found in PATH - skipping compression.
)

echo.
echo [+] Done. Artifacts:
for %%I in (build\bin\NecoClicker.exe)                  do echo     %%~fI  ^(%%~zI bytes^)
for %%I in (build\bin\NecoClicker-amd64-installer.exe)  do echo     %%~fI  ^(%%~zI bytes^)
endlocal
