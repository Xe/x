const regex =
  /^New Torrent Announcement: <([^>]*)>\s+Name:'(.*)' uploaded by '.*' ?(freeleech)?\s+-\s+https:..\w+.\w+.\w+\/.\w+\/([0-9]+)$/;

const genURL = (torrentName, baseURL, id, passkey) =>
  `https://www.torrentleech.org/rss/download/${id}/${passkey}/${torrentName}`;

export const allowLine = (nick, channel) => {
  if (channel !== "#tlannounces") {
    return false;
  }

  if (nick !== "_AnnounceBot_") {
    return false;
  }

  return true;
};

export const parseLine = (msg) => {
  const [
    _blank,
    category,
    torrentName,
    freeleech,
    baseURL,
    id,
    size,
  ] = msg.split(regex);

  return {
    torrent: {
      category,
      name: torrentName,
      freeleech: freeleech !== "",
      id: id,
      url: genURL(torrentName, baseURL, id),
    },
  };
};
