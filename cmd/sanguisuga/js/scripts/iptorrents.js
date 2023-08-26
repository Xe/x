const regex =
  /^\[([^\]]*)]\s+(.*?(FREELEECH)?)\s+-\s+https?\:\/\/([^\/]+).*[&;\?]id=(\d+)\s*- (.*)$/;

const genURL = (torrentName, baseURL, id, passkey) =>
  `https://${baseURL}/download.php/${id}/${torrentName}.torrent?torrent_pass=${passkey}`;

export const allowLine = (nick, channel) => {
  if (channel != "#ipt.announce") {
    return false;
  }

  if (nick !== "IPT") {
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
