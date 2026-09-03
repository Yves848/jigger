Output out/omarchy-04-regex.gif
Output out/omarchy-04-regex.mp4

# --- décor figé : identique sur macOS et Omarchy (et repris par capturer.ps1) ---
Set FontFamily "MesloLGL Nerd Font"
Set FontSize 22
Set Width 1000
Set Height 530
Set Padding 24
Set TypingSpeed 90ms
Set Framerate 24
Set Theme { "background": "#1e1e2e", "foreground": "#cdd6f4", "cursor": "#f5e0dc", "selection": "#585b70", "black": "#45475a", "red": "#f38ba8", "green": "#a6e3a1", "yellow": "#f9e2af", "blue": "#89b4fa", "magenta": "#f5c2e7", "cyan": "#94e2d5", "white": "#bac2de", "brightBlack": "#585b70", "brightRed": "#f38ba8", "brightGreen": "#a6e3a1", "brightYellow": "#f9e2af", "brightBlue": "#89b4fa", "brightMagenta": "#f5c2e7", "brightCyan": "#94e2d5", "brightWhite": "#a6adc8" }

Set Shell "zsh"
Hide
Type "tmux -L jiggercap -f $JIGGER_MEDIA/fixtures/tmux.conf new-session -A -s capture zsh" Enter
Sleep 2s
Type "clear" Enter
Sleep 1s
Show
Sleep 800ms

Type "yay -S fire"
Sleep 3s
Ctrl+R
Sleep 2s
Type "(bird|fly)"
Sleep 3s
Tab
Sleep 2500ms
