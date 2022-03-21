# bingus battle network 6 (bbn6)

![bingus battle network 6](logo.png)

[![murkland](https://discordapp.com/api/guilds/936475149069336596/widget.png?style=shield)](https://discord.gg/zbQngJHwSg)

netplay for mega man battle network 6 in the style of https://github.com/ssbmars/BBN3-netcode. the b in bbn6 does NOT stand for "better".

## how to use

1.  start the bundled vba-rr in `vba-rr/VBA-rr-svn480+LRC4.exe`

1.  if you are:

    -   **server:**

        1. go to _Tools > Lua Scripting > New Lua Script Window..._ and load `main_server.lua`.

    -   **client:**

        1. edit `vba-rr/lua/main_client.lua` to set `HOST` to the opponent's IP.

        2. wait for the opponent to run `main_server.lua`. they MUST start the server first.

        3. go to _Tools > Lua Scripting > New Lua Script Window..._ and load `main_client.lua`.

1.  in game, go to _Comm > LINK CBL > NetBattl > SnglBatt > Practice_. once both players have connected, you should both be in battle. have fun!

## you will need

-   a copy of MEGAMAN6_FXX (MEGAMAN6_GXX is not supported yet)

-   an indefinite amount of patience

## thank you

-   the **[National Security Agency](https://nsa.gov)**

-   **[luckytyphlosion](https://github.com/luckytyphlosion)** for https://github.com/dism-exe/bn6f and letting me bug him incessantly about dumb shit

-   **[ssbmars](https://github.com/ssbmars)** for the original BBN3 code

-   **[Playerzero_exe](https://twitter.com/Playerzero_exe)** for digging through frame data

-   **[aldelaro5](https://github.com/aldelaro5)** for help with using Ghidra with mGBA and defending me from everyone telling me to use NO$GBA

-   **[ExeDesmond](https://twitter.com/exedesmond)** for playtesting and finding horrible desync bugs
