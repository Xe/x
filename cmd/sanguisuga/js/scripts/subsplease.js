const regex =
  /^.*\* (\[SubsPlease\] (.*) - ([0-9]+) \(([0-9]{3,4})p\) \[([0-9A-Fa-f]{8})\]\.mkv) \* .MSG ([^ ]+) XDCC SEND ([0-9]+)$/;

const bots = [
  "CR-ARUTHA|NEW",
  "CR-HOLLAND|NEW",
];

export const ircInfo = {
    server: "irc.rizon.net:6697",
    channel: "#subsplease",
    downloadType: "DCC",
};

export const allowLine = (nick, channel) => {
  if (channel != "#subsplease") {
    return false;
  }

  if (!bots.includes(nick)) {
    return false;
  }

  return true;
};

export const parseLine = (msg) => {
  const [
    _blank,
    fname,
    showName,
    episode,
    resolution,
    crc32,
    botName,
    packID,
  ] = msg.split(regex);

  const result = {
    fname,
    showName,
    episode,
    resolution,
    crc32,
    botName,
    packID,
  };

  return result;
};
