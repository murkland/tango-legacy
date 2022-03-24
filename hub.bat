@echo off
:MENU
ECHO.
ECHO ...................
ECHO  Welcome to BBN6
ECHO ...................
ECHO.
ECHO 1 - Be Server
ECHO 2 - Be Client
ECHO 3 - Exit
ECHO.
SET /P M=Type 1, 2, or 3 then press ENTER:
IF %M%==1 GOTO SERVER
IF %M%==2 GOTO CLIENT
IF %M%==3 GOTO EOF
echo:
:SERVER
echo welcome to the bbn6 server!
set /p "SESSION_ID=matchmaking code (no spaces, pick one you and your friend agree on!): "
bbn6.exe -connect_addr=http://167.71.122.211:11223 -session_id=%SESSION_ID% 2> bbn6_server.log
pause
GOTO EOF
echo:
:CLIENT
@echo off
echo welcome to the bbn6 client!
set /p "SESSION_ID=matchmaking code (no spaces, pick one you and your friend agree on!): "
bbn6.exe -connect_addr=http://167.71.122.211:11223 -answer -session_id=%SESSION_ID% 2> bbn6_client.log
pause
GOTO EOF
echo:
:EOF
echo:
