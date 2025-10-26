@echo off
go vet -vettool=statictest ./...
metricstest.exe -test.v -test.run=^TestIteration1$ -binary-path=cmd/server/server
metricstest.exe -test.v -test.run=^TestIteration2[AB]$ -source-path=. -agent-binary-path=cmd/agent/agent
metricstest.exe -test.v -test.run=^TestIteration3[AB]*$ -source-path=. -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server

REM Генерируем случайный порт
for /f %%a in ('powershell -Command "Get-Random -Minimum 1024 -Maximum 65535"') do set SERVER_PORT=%%a

REM Проверяем, что порт свободен (простой способ)
for /f %%b in ('netstat -ano ^| findstr "%SERVER_PORT%"') do (
    if not "%%b"=="" (
        echo Порт %SERVER_PORT% занят, попробуйте перезапустить скрипт
        exit /b 1
    )
)

REM Формируем адрес
set ADDRESS=localhost:%SERVER_PORT%

REM Создаем временный файл
set TEMP_FILE=C:\tmp\metrics-db.json

REM Запускаем тест
metricstest.exe -test.v -test.run=^TestIteration4^ ^
    -agent-binary-path=cmd\agent\agent ^
    -binary-path=cmd\server\server ^
    -server-port=%SERVER_PORT% ^
    -source-path=.
pause
REM Запускаем тест
metricstest.exe -test.v -test.run=^TestIteration5^ ^
    -agent-binary-path=cmd\agent\agent ^
    -binary-path=cmd\server\server ^
    -server-port=%SERVER_PORT% ^
    -source-path=.
pause
REM Запускаем тест
metricstest.exe -test.v -test.run=^TestIteration6^ ^
    -agent-binary-path=cmd\agent\agent ^
    -binary-path=cmd\server\server ^
    -server-port=%SERVER_PORT% ^
    -source-path=.
pause
REM Запускаем тест
metricstest.exe -test.v -test.run=^TestIteration7^ ^
    -agent-binary-path=cmd\agent\agent ^
    -binary-path=cmd\server\server ^
    -server-port=%SERVER_PORT% ^
    -source-path=.
pause
REM Запускаем тест
metricstest.exe -test.v -test.run=^TestIteration8^ ^
    -agent-binary-path=cmd\agent\agent ^
    -binary-path=cmd\server\server ^
    -server-port=%SERVER_PORT% ^
    -source-path=.
pause
REM Запускаем тест
metricstest.exe -test.v -test.run=^TestIteration9^ ^
    -agent-binary-path=cmd\agent\agent ^
    -binary-path=cmd\server\server ^
    -file-storage-path=%TEMP_FILE% ^
    -server-port=%SERVER_PORT% ^
    -source-path=.