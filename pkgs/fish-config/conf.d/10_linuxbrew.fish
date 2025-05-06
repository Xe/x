if test -d /home/linuxbrew/.linuxbrew
  set -gx HOMEBREW_PREFIX "/home/linuxbrew/.linuxbrew"
  set -gx HOMEBREW_CELLAR "/home/linuxbrew/.linuxbrew/Cellar"
  set -gx HOMEBREW_REPOSITORY "/home/linuxbrew/.linuxbrew/Homebrew"
  set -gx PATH "/home/linuxbrew/.linuxbrew/bin" "/home/linuxbrew/.linuxbrew/sbin" $PATH
  set -q MANPATH; or set MANPATH ''
  set -gx MANPATH "/home/linuxbrew/.linuxbrew/share/man" $MANPATH
  set -q INFOPATH; or set INFOPATH ''
  set -gx INFOPATH "/home/linuxbrew/.linuxbrew/share/info" $INFOPATH
else if test -d /opt/homebrew
  # Homebrew is installed on MacOS

  /opt/homebrew/bin/brew shellenv | source
end
