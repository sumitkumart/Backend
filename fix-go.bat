@echo off
REM Fix Go installation and run the server

echo Fixing Go GOROOT environment variable...
setx GOROOT C:\Go
echo.
echo IMPORTANT: Close all terminal windows and open a NEW one to apply the change.
echo After restarting, run this command:
echo.
echo   cd c:\Users\DELL\Downloads\Backend
echo   go mod tidy
echo   go run ./cmd/server
echo.
echo Alternatively, use Docker (recommended):
echo.
echo   docker-compose up --build
echo.
pause
