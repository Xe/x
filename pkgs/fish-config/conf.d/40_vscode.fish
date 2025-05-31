if ! status is-interactive ||
        test "$TERM_PROGRAM" != vscode ||
        test -n "$VSCODE_SHELL_INTEGRATION"
    exit
end

set --global VSCODE_SHELL_INTEGRATION 1

function __vsc_initialize --on-event fish_prompt
    functions --erase __vsc_initialize

    function __vsc_command_output_start --on-event fish_preexec
        # Ignore commands with leading spaces or in private mode
        if string match --quiet -- " *" $argv || test -n "$fish_private_mode"
            set argv ""
        end

        printf "\e]633;C\a"
        printf "\e]633;E;%s\a" (
            string replace --all -- "\\" "\\\\" $argv |
            string replace --all ";" "\x3b" |
            string join "\x0a"
        )

        set --global _vsc_has_command
    end

    function __vsc_command_complete --on-event fish_postexec
        printf "\e]633;D;$status\a"
    end

    function __vsc_command_cancel --on-event fish_cancel
        printf "\e]633;E\a"
        printf "\e]633;D\a"
    end

    function __vsc_update_cwd --on-variable PWD
        printf "\e]633;P;Cwd=$PWD\a"
    end

    function __vsc_check_command --on-event fish_prompt
        if set --query _vsc_has_command
            set --erase _vsc_has_command
        else
            # Empty command, trigger cancel event
            __vsc_command_cancel
        end
    end

    functions --copy fish_prompt __vsc_fish_prompt

    function fish_prompt
        printf "\e]633;A\a"
        __vsc_fish_prompt
        printf "\e]633;B\a"
    end
end
