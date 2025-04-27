export type IRC = {
  server: string;
  password: string;
  channel: string;
  nick: string;
  user: string;
  real: string;
};

export type Show = {
  title: string;
  diskPath: string;
  quality: string;
};

export type Transmission = {
  host: string;
  user: string;
  password: string;
  https: boolean;
  rpcURI: string;
};

export type Tailscale = {
  hostname: string;
  authkey: string;
  dataDir?: string;
};

export type Telegram = {
  token: string;
  mentionUser: number;
};

export type WireGuardPeer = {
  publicKey: string;
  endpoint: string;
  allowedIPs: string[];
};

export type WireGuard = {
  privateKey: string;
  address: string[];
  dns: string;
  peers: WireGuardPeer[];
};

export type Config = {
  irc: IRC;
  xdcc: IRC;
  transmission: Transmission;
  shows: Show[];
  rssKey: string;
  tailscale: Tailscale;
  baseDiskPath: string;
  telegram: Telegram;
  wireguard: WireGuard;
};

export default {
  irc: {
    server: "",
    password: "",
    channel: "",
    nick: "",
    user: "",
    real: "",
  },
  xdcc: {
    server: "",
    password: "",
    channel: "",
    nick: "",
    user: "",
    real: "",
  },
  transmission: {
    host: "",
    user: "",
    password: "",
    https: true,
    rpcURI: "/transmission/rpc",
  },
  shows: [
    {
      title: "Show Name",
      diskPath: "/data/TV/Show Name",
      quality: "1080p",
    },
  ],
  rssKey: "",
  tailscale: {
    hostname: "sanguisuga-dev",
    authkey: "",
    dataDir: undefined,
  },
  baseDiskPath: "/data/TV/",
  telegram: {
    token: "",
    mentionUser: 0,
  },
  wireguard: {
    // for downloading files over DCC (XDCC)
    privateKey: "",
    address: [],
    dns: "",
    peers: [
      {
        publicKey: "",
        allowedIPs: [],
        endpoint: "",
      },
    ],
  },
} satisfies Config;
