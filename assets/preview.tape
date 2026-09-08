Output ./assets/preview.gif

Require go

Set Shell "fish"
Set FontFamily "ComicShannsMono Nerd Font Mono"
Set FontSize 20
Set Width 1400
Set Height 850
Set Padding 28
Set Margin 60
Set MarginFill "./assets/gradient.png"
Set BorderRadius 20
Set CursorBlink true

Hide
Type "set -g fish_autosuggestion_enabled 0"
Enter
Type "clear"
Enter
Show

Type "sitch -c examples/minimal.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/custom.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/all-rows.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/three-columns.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/tty.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/1-2-3-1.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/1-3-2-1.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/2-3-1-2.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/3-1-2-3.toml"
Enter
Sleep 2s
Type "clear"
Enter
Type "sitch -c examples/3-2-1-3.toml"
Enter
Sleep 2s
