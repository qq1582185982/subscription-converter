@echo off
echo ??  ����ת�������� - Go�汾�����ű�
echo ===============================================

REM ���Go�Ƿ�װ
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ? Go����δ��װ�����Ȱ�װGo���Ի���
    echo ? ���ص�ַ: https://golang.org/dl/
    pause
    exit /b 1
)

echo ? Go�������ͨ��

REM ��ʼ��ģ�����������
echo ? ����������...
go mod tidy

REM ����Windows�汾
echo ? ����Windows�汾...
go build -ldflags "-s -w" -o subscription-converter.exe .
if %errorlevel% neq 0 (
    echo ? Windows�汾����ʧ��
    pause
    exit /b 1
)

REM ����Linux�汾
echo ? ����Linux�汾...
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-s -w" -o subscription-converter-linux .
if %errorlevel% neq 0 (
    echo ? Linux�汾����ʧ��
    pause
    exit /b 1
)

REM ����MacOS�汾
echo ? ����MacOS�汾...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags "-s -w" -o subscription-converter-macos .
if %errorlevel% neq 0 (
    echo ? MacOS�汾����ʧ��
    pause
    exit /b 1
)

echo ? ������ɣ�
echo.
echo ? ���ɵ��ļ�:
echo    - subscription-converter.exe     (Windows�汾)
echo    - subscription-converter-linux   (Linux�汾)
echo    - subscription-converter-macos   (MacOS�汾)
echo.
echo ? ʹ��˵��:
echo    1. ˫�� subscription-converter.exe ����������
echo    2. ��������з�����ʾ�ĵ�ַ
echo    3. �ϴ�������Clash�����ļ�
echo    4. ���ɶ�������
echo.
pause 