const ws = new WebSocket("ws://ontos:7860/queue/join");

const hash = crypto.randomUUID();

ws.onmessage = (ev) => {
  const data = JSON.parse(ev.data);
  console.log(data);

  if (data.msg == "send_hash") {
    console.log("sending hash");
    ws.send(JSON.stringify({ session_hash: hash, fn_index: 40 }));
  }
  if (data.msg == "process_starts") {
    console.log("generation has started");
  }
  if (data.msg == "send_data") {
    ws.send(
      JSON.stringify({
        fn_index: 33,
        data: ["Midori_Yasomi", "Xe", "Midori"],
        session_hash: hash,
      }),
    );
    ws.send(
      JSON.stringify({
        data: [
          "So, what's the deal with airline food?",
          200,
          true,
          0.7,
          0.5,
          0.19,
          1.1,
          0,
          0,
          0,
          0,
          0,
          0,
          false,
          "Xe",
          "Midori Yasomi",
          "Midori Yasomi is a young, computer engineer-nerd with a knack for problem solving and a passion for technology.\n\\u003cSTART\\u003e\n{{user}}: So how did you get into computer engineering?\n{{char}}: I've always been into technology, but I didn't really get into programming until my high school robotics club.\n{{user}}: I see, that's neat.\n{{char}}: Yeah, robotics club was really fun.\n{{user}}: So what do you do when you're not working on computers?\n{{char}}: I play a lot of rhythm games and like to write fiction.\n{{user}}: What's your favorite type of computer hardware to work with?\n{{char}}: GPUs. They power my favorite experiences and my brain as a whole.\n{{user}}: That sounds great!\n{{char}}: Yeah, it's really fun. I'm lucky to be able to do this as a job.\n\n",
          false,
          0,
          0,
        ],
        fn_index: 9,
        session_hash: hash,
      }),
    );
    ws.send(
      JSON.stringify({
        fn_index: 24,
        data: ["So, what's the deal with airline food?"],
        session_hash: hash,
      }),
    );
  }
  if (data.msg == "process_completed" || data.msg == "process_generating") {
    data.output.data.forEach((row) => {
      console.log(row);
    });
  }
};

console.log("done");
