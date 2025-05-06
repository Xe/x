set -gx PATH $PATH $HOME/go/bin $HOME/bin
set -gx PATH $PATH $HOME/.local/bin
set -gx PATH $PATH $HOME/.luarocks/bin
set -gx PATH $PATH $HOME/.cargo/bin

set -gx WASMER_DIR $HOME/.wasmer
set -gx WASMER_CACHE_DIR $WASMER_DIR/cache
set -gx PATH $PATH $WASMER_DIR/bin $WASMER_DIR/globals/wapm_packages/.bin

set -gx EDITOR vim