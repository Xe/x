# status is-interactive; or return 0

function __autols.fish::on::install --on-event autols_install
    set --query AUTOLS_FISH_IGNORED_DIRS; or set --universal AUTOLS_FISH_IGNORED_DIRS
    # set --append AUTOLS_FISH_IGNORED_DIRS "*node_modules"
    set --append AUTOLS_FISH_IGNORED_DIRS "*build"
end

function __autols.fish::on::uninstall --on-event autols_uninstall
    for v in (set --universal --names)
        string match --quiet "AUTOLS_FISH_*" $v; and set --erase $v
    end
    for fn in (functions --names)
        string match --quiet "__autols.fish::*" $fn; and functions --erase $fn
    end
end

function __autols.fish::on::postexec --on-event fish_postexec
    # Used to control if autols should be enabled or not
    set --query AUTOLS_FISH_DISABLED; and return 0

    set --query __autols_last_dir; or set --global __autols_last_dir $PWD
    # Do not want to `ls` if the user did `cd .`
    test $PWD = $__autols_last_dir; and return 0
    set __autols_last_dir $PWD

    # Check if the current directory is ignored
    for dir in $AUTOLS_FISH_IGNORED_DIRS
        string match --quiet --ignore-case "$dir" $PWD; and return 0
    end

    set -l cmd ls
    # for f in * .*
    # for f in *
    #     # Use the long option to show what the symlink points to
    #     test -L $f; and set cmd ll; and break
    # end

    $cmd
end
