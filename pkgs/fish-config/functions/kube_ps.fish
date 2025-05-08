function kube_ps -a toggle
  if test "$toggle" = "on"
    set -U __kube_ps_enabled 1
    return
  end

  if test "$toggle" = "off"
    set -U __kube_ps_enabled 0
    return
  end
end
