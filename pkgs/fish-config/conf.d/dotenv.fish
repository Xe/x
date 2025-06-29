status --is-interactive || exit

# Filename of the dotenv file to look for
set -q FISH_DOTENV_FILE || set -g FISH_DOTENV_FILE .env.fish

# Path to the file containing allowed paths
set -q FISH_DOTENV_ALLOWLIST || set -g FISH_DOTENV_ALLOWLIST "$__fish_config_dir/.dotenv-allowed.list"
set -q FISH_DOTENV_BLOCKLIST || set -g FISH_DOTENV_BLOCKLIST "$__fish_config_dir/.dotenv-blocked.list"

function _fish_dotenv_source
    # First shell out to source the file in an isolated fashion. This is to
    # ensure "atomicity" where either all settings as sourced or none at all.
    if ! fish --private --no-config --command="source $FISH_DOTENV_FILE"
        echo "dotenv: Error sourcing '$FISH_DOTENV_FILE' file, bailing." >&2
        return 1
    end

    echo "dotenv: Sourcing '$FISH_DOTENV_FILE'" >&2
    source $FISH_DOTENV_FILE
end

function _fish_dotenv_hook --on-variable PWD
    [ -f "$FISH_DOTENV_FILE" ] || return
    set -f dirpath "$PWD"

    # Ensure blocklist exists
    touch "$FISH_DOTENV_BLOCKLIST"

    if command grep -Fx -q "$dirpath" "$FISH_DOTENV_BLOCKLIST" &>/dev/null
        return
    end

    # Ensure allowlist exists
    touch "$FISH_DOTENV_ALLOWLIST"

    # Check if env file is allowed or ask for confirmation
    if command grep -Fx -q "$dirpath" "$FISH_DOTENV_ALLOWLIST" &>/dev/null
        _fish_dotenv_source
    else
        read -n 1 \
            -f confirmation \
            --prompt-str="dotenv: Found '$FISH_DOTENV_FILE' file. Source it? ([N]o/[y]es/[a]lways/n[e]ver) "
        switch "$confirmation"
            case y Y
                _fish_dotenv_source
            case a A
                echo "dotenv: Adding to allowlist" >&2
                echo "$dirpath" >>"$FISH_DOTENV_ALLOWLIST"
                _fish_dotenv_source
            case e E
                echo "dotenv: Adding to blocklist" >&2
                echo "$dirpath" >>"$FISH_DOTENV_BLOCKLIST"
            case '*'
                echo "dotenv: Not sourcing." >&2
        end
    end
end

_fish_dotenv_hook
