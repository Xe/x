function fish_greeting
  echo "Hostname:  $(hostname)"
  echo "    User:  $(whoami)"
  echo " OS/arch:  $(uname -s)/$(uname -m)"
end