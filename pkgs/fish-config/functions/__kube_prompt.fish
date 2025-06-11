# Inspired from:
# https://github.com/jonmosco/kube-ps1
# https://github.com/Ladicle/fish-kubectl-prompt

function __kube_ps_update_cache
  function __kube_ps_cache_context
    set -l ctx (kubectl config current-context 2>/dev/null)
    if test $status -eq 0
      set -g __kube_ps_context "$ctx"
    else
      set -g __kube_ps_context "n/a"
    end
  end

  function __kube_ps_cache_namespace
    set -l ns (kubectl config view --minify -o 'jsonpath={..namespace}' 2>/dev/null)
    if test -n "$ns"
      set -g __kube_ps_namespace "$ns"
    else
      set -g __kube_ps_namespace "default"
    end
  end

  function __stat_mtime
    #-- cross-platform workaround; POSIX didn't specify stat(1) and so
    #-- its interface is incompatibly different on Mac OS and Linux.
    #-- see https://unix.stackexchange.com/q/561927/3097
    python -c "print(__import__('os').stat(__import__('sys').argv[1]).st_mtime)" $argv
  end

  set -l kubeconfig "$KUBECONFIG"
  if test -z "$kubeconfig"
    set kubeconfig "$HOME/.kube/config"
  end

  if test "$kubeconfig" != "$__kube_ps_kubeconfig"
    __kube_ps_cache_context
    __kube_ps_cache_namespace
    set -g __kube_ps_kubeconfig "$kubeconfig"
    set -g __kube_ps_timestamp (date +%s)
    return
  end

  for conf in (string split ':' "$kubeconfig")
    if test -r "$conf"
      if test -z "$__kube_ps_timestamp"; or test (__stat_mtime "$conf") -gt "$__kube_ps_timestamp"
        __kube_ps_cache_context
        __kube_ps_cache_namespace
        set -g __kube_ps_kubeconfig "$kubeconfig"
        set -g __kube_ps_timestamp (date +%s)
        return
      end
    end
  end
end

function __kube_prompt
  if test -z "$__kube_ps_enabled"; or test $__kube_ps_enabled -ne 1
    return
  end

  __kube_ps_update_cache
  echo -n -s "(⎈|$__kube_ps_context)"
end
