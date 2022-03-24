@echo off
cls
:MENU
ECHO.   .  .-O*.  ..-...--..-*-*O#Oo*--..-*.----     .#o     
ECHO.   -  .-O*.  ..*-.-*-.-***oO##o***-.-*.-*--     .#O     
ECHO.  .-.   O-.   -**.oo-.-oo*oO##Ooo*--**.*O.-.   ..#o     
ECHO. ..*.   O. ...-**.*o---ooooO##Ooo***o*-oo.o-.  ..O.     
ECHO. .-*.  .-. ...*OO-oO**-oooOO##Ooooo*oooOO*O--....O.     
ECHO. .-O.  ......-OOO**ooooOOOO#O**-.-----oOooo*----.-.     
ECHO. .*#......---*oOOooOOooo****----.--*-**oOoooo***----  . 
ECHO. .*#. ..----**oOOoOo*-------------**ooo*...**Oo*----  - 
ECHO. .*O....---**ooO#Oo***********oooooo**o* ..---*o-.... O 
ECHO. .-O...--*o*ooOO**oooo*o*oo*o**o***-***..-*-*--*o*--- # 
ECHO. .-*....-*o*oO*---*****--*---..-----*oo.....--..-o*--.* 
ECHO.  ..-.--*oOooO********---***-.-****-*##O*--.-**o..o-..- 
ECHO.  .....**oOoO-.------..----...-----*O#OOOo ...-*# -*... 
ECHO. ...--***O#oO-.-..--.-**-----*****o--O#O#O--.-.-*O.O--- 
ECHO.  ----*oOO#O#-...*-..--.---*---*--**.-oOO##-..-.*OO*..- 
ECHO. .---o***oooo*- ***o-.-******oO*oo.Oo*O##o**...--Oo*-*- 
ECHO.  ...-o-*....-. *oooo*-..-ooOO#O#o.*#OOo-.o .* .Oo*-.O. 
ECHO. .----*##oo-.....*oo***..oOOoOOOOOoOOOO* -*.*O oOo*.-#- 
ECHO.  .----Oo--.......-*ooo*-*ooo#OOO###OO#Oo..o#Oo#Ooo-*#* 
ECHO. ..---*OOOOooo*-..---***-..*OOOOOOOOOOO#OO#####Oooo-*#* 
ECHO.  o-.-*ooOO#o*. ... .      .o#O#OOOOO##**###OOOOooo--#- 
ECHO.  o--**OOOOOoooo#-.*#-      .o#OO#OOOo*o###OOOOoo**..#  
ECHO.  O..*.oOoO##OOo##O#@-        -oOOOooO####OOO#Ooo** .O  
ECHO.  O-.O-*ooOO*..--o#O-.. .      -OOO####O#O##OOooo*-  -  
ECHO.  O-.O--oo#-.-**--*-----.....  .*O####O##O##oOooO*-  .  
ECHO.  O-.#-.oo#o..---**oOOOO*........*Oo##OO#OO#oOooO*.  .  
ECHO.  O-*#*.ooO#O-.***oOO##OO-  ......--*OOO#Oo#oOooO*.     
ECHO.  O.O#*.*o*O#*-**oOooOOoO*..    ....--oO#oo#oOooO-.    -
ECHO .........................................................
ECHO    Welcome to BBN6 - Hub Batch is Prettier Now Edition
ECHO .........................................................
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
echo Welcome to the BBN6 Server Setup!
set /p "SESSION_ID=matchmaking code (no spaces, pick one you and your friend agree on!): "
bbn6.exe -connect_addr=http://167.71.122.211:11223 -session_id=%SESSION_ID% 2> hub_server.log
pause
GOTO EOF
echo:
:CLIENT
@echo off
echo Welcome to the BBN6 Client Setup!
set /p "SESSION_ID=matchmaking code (no spaces, pick one you and your friend agree on!): "
bbn6.exe -connect_addr=http://167.71.122.211:11223 -answer -session_id=%SESSION_ID% 2> hub_client.log
pause
GOTO EOF
echo:

