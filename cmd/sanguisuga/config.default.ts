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
  https: bool;
  rpcURI: string;
};

export type Tailscale = {
  hostname: string;
  authkey: string;
  dataDir?: string;
};

export type Config = {
  irc: IRC;
  xdcc: IRC;
  transmission: Transmission;
  shows: Show[];
  rssKey: string;
  tailscale: Tailscale;
  baseDiskPath: string;
};

export default {
  irc: {
    server: "",
    password: "",
    channel: "",
    nick: "",
    user: "",
    real: ""
  },
  transmission: {
    host: "",
    user: "",
    password: "",
    https: true,
    rpcURI: "/transmission/rpc"
  },
  shows: [
    {
      title: "Show Name",
      diskPath: "/data/TV/Show Name",
      quality: "1080p"
    }
  ],
  rssKey: "",
  tailscale: {
    hostname: "sanguisuga-dev",
    authkey: "",
    dataDir: undefined,
  },
} satisfies Config;
