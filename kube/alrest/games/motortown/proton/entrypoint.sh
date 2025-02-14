# add this to entrypoint so it is more generic

steamcmd=${STEAM_HOME}/steamcmd/steamcmd.sh

mkdir -p steamapps/compatdata/1829350
cp -r compatibilitytools.d/${PROTON_VERSION}/files/share/default_pfx steamapps/compatdata/1829350
export STEAM_COMPAT_DATA_PATH=${STEAM_PATH}/steamapps/compatdata/1829350

cd ${STEAM_HOME}

pwd