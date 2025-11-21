@echo off
REM Start the backend server

cd /d c:\Users\DELL\Downloads\Backend

echo.
echo Starting Stocky Backend Server...
echo MongoDB Connection: mongodb+srv://sumitkumartiwari627_db_user@cluster0.kditvi6.mongodb.net/
echo API Endpoint: http://localhost:8080
echo.

go mod tidy
go run ./cmd/server

pause
