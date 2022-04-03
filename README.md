# tango

![tango](logo.png)

[![murkland](https://discordapp.com/api/guilds/936475149069336596/widget.png?style=shield)](https://discord.gg/zbQngJHwSg)

netplay for mega man battle network games in the style of <https://github.com/ssbmars/BBN3-netcode>.

tango（タンゴ）はロックマンエグゼ６ネットプレイシステムです。

## how to use / 使用方法

-   add your legally-obtained roms in the `roms` folder and your hard-earned saves on the `saves` folder
    
    合法的に入手したROMは `roms` フォルダに、苦労して保存したものは `saves` フォルダに保存してください。

-   if you are on windows, just run `hub.bat`. if you are on not windows, you're on your own! (for now)

    Windows を使用している場合は、`hub.bat` を起動してください。そうでない場合は、現在サポートされていません。

-   you can connect to an opponent in-game by going to the menu then going to Comm > LINK CBL > NetBattl > SnglBatt / TrplBatt (do NOT pick RandBatt) > Practice. a dialog will pop up where you can enter a matchmaking code.

    ゲーム内で相手と接続するために、メニューに入り、つうしん → つうしんケーブル → ネットバトル → シングルバトル・トリプルバトル（ランダムバトルを選択しない）→ れんしゅうを選択し、リンクコードを入力してください。

-   if you run any into any issues, please let us know on our discord server: <https://discord.gg/zbQngJHwSg>

    何か問題が発生した場合は、私たちのディスコード・サーバーに連絡してください。 <https://discord.gg/zbQngJHwSg>

## remapping controls / コントロールリマッピング

-   after executing `hub.bat` once, a new file named `tango.toml` will be created on your folder

    `hub.bat`を一回実行すると、あなたのフォルダに `tango.toml` という名前のファイルが作成されます。    

-   you can edit the `[Keymapping]` section to change your enabled keys. The list of valid keys is included in the `keys.txt` file

    は、`[Keymapping]`セクションを編集して、有効なキーを変更することができます。有効なキーの一覧は `keys.txt` ファイルに含まれています。

## supported games / 対応ゲーム

-   MEGAMAN6_FXX: Mega Man Battle Network 6: Cybeast Falzar
-   MEGAMAN6_GXX: Mega Man Battle Network 6: Cybeast Gregar
-   ROCKEXE6_RXX: ロックマンエグゼ 6 電脳獣ファルザー
-   ROCKEXE6_GXX: ロックマンエグゼ 6 電脳獣グレイガ

## thank you / 感謝

too many to count! a full list of credits is available with each release on the [releases page](https://github.com/murkland/tango/releases)!

[リリースのページ](https://github.com/murkland/tango/releases)で各リリースのクレジットを見ることができます。
