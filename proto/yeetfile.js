const mkGoProtoFile = (name, kinds) => {
    return {
        "input": `./${name}.proto`,
        "output": `./${name}`,
        "kinds": kinds.map((kind) => {
            return { "kind": kind, "opt": "paths=source_relative" };
        }),
    };
};

const protoFiles = [
    mkGoProtoFile("uplodr", ["go", "go-grpc"]),
    mkGoProtoFile("sanguisuga", ["go", "twirp"]),
    mkGoProtoFile("mimi/statuspage", ["go", "twirp"]),
];

protoFiles.forEach((protoFile) => {
    repoRoot = yeet.runcmd("git", "rev-parse", "--show-toplevel").trim();

    args = [
        `--proto_path=${yeet.cwd}`,
    ];

    protoFile.kinds.forEach((kind) => {
        args.push(`--${kind.kind}_out=${protoFile.output}`);
        if (kind.opt !== undefined) {
            args.push(`--${kind.kind}_opt=${kind.opt}`);
        }
    });

    yeet.runcmd(
        "protoc",
        "--proto_path=.",
        ...args,
        protoFile.input
    );
});
