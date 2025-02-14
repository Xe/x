FROM cm2network/steamcmd

RUN apt-get update && apt-get install -y gosu cron wget python3 libfreetype6 xvfb

ENV STEAM_HOME="/home/steam" \
  STEAM_USER="steam" \
  STEAM_PATH="/home/steam/.steam/steam"

RUN mkdir -p ${STEAM_PATH}
WORKDIR ${STEAM_PATH}

# Install Proton build from Glorious Eggroll
ENV PROTON_VERSION=GE-Proton9-24

RUN mkdir -p compatibilitytools.d/
RUN wget -O - \
  https://github.com/GloriousEggroll/proton-ge-custom/releases/download/${PROTON_VERSION}/${PROTON_VERSION}.tar.gz \
  | tar -xz -C compatibilitytools.d/

RUN chown -R ${USER}:${USER} ${STEAM_HOME}

# Export Proton paths
ENV STEAM_COMPAT_CLIENT_INSTALL_PATH=$STEAM_PATH
ENV PROTON=${STEAM_PATH}/compatibilitytools.d/${PROTON_VERSION}/proton

COPY entrypoint.sh ${STEAM_HOME}/entrypoint.sh
RUN chmod +x ${STEAM_HOME}/entrypoint.sh

ENTRYPOINT ${STEAM_HOME}/entrypoint.sh