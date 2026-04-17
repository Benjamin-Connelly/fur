#!/usr/bin/env bash
# Helpers used by demo.tape to show chapter captions between fur sessions.
# banner TITLE [KEYS]
banner() {
    local title="$1"
    local keys="$2"
    clear
    printf '\n\n\n'
    printf '   \033[1;35m┃\033[0m \033[1;37m%s\033[0m\n' "$title"
    if [ -n "$keys" ]; then
        printf '   \033[1;35m┃\033[0m \033[0;36mkeys:\033[0m \033[0;33m%s\033[0m\n' "$keys"
    fi
    printf '\n'
}

# title_card
title_card() {
    clear
    printf '\n\n\n\n'
    printf '   \033[1;35m▌fur\033[0m  \033[0;37m— dual-mode markdown navigator\033[0m\n\n'
    printf '   \033[0;90m   split-pane TUI · web server · SSH remote · MCP · link graph\033[0m\n\n'
    printf '   \033[0;36m   github.com/Benjamin-Connelly/fur\033[0m\n'
}

# end_card
end_card() {
    clear
    printf '\n\n\n\n'
    printf '   \033[1;35m▌end of demo\033[0m\n\n'
    printf '   \033[0;37m   install:\033[0m  \033[0;33mbrew install Benjamin-Connelly/fur/fur\033[0m\n'
    printf '   \033[0;37m   source:\033[0m   \033[0;33mgo install github.com/Benjamin-Connelly/fur/cmd/fur@latest\033[0m\n\n'
    printf '   \033[0;36m   github.com/Benjamin-Connelly/fur\033[0m\n'
}
