// https://dev.to/ratmd/simple-proof-of-work-in-javascript-3kgm

export function process(data, difficulty = 5) {
  return new Promise((resolve, reject) => {
    let webWorkerURL = URL.createObjectURL(new Blob([
      '(', processTask(), ')()'
    ], { type: 'application/javascript' }));

    let worker = new Worker(webWorkerURL);

    worker.onmessage = (event) => {
      worker.terminate();
      resolve(event.data);
    };

    worker.onerror = (event) => {
      worker.terminate();
      reject();
    };

    worker.postMessage({
      data,
      difficulty
    });

    URL.revokeObjectURL(webWorkerURL);
  });
}

function processTask() {
  return function () {
    function sha256(text) {
      return new Promise((resolve, reject) => {
        let buffer = (new TextEncoder).encode(text);

        crypto.subtle.digest('SHA-256', buffer.buffer).then(result => {
          resolve(Array.from(new Uint8Array(result)).map(
            c => c.toString(16).padStart(2, '0')
          ).join(''));
        }, reject);
      });
    }

    addEventListener('message', async (event) => {
      let data = event.data.data;
      let difficulty = event.data.difficulty;

      let hash;
      let nonce = 0;
      do {
        hash = await sha256(data + nonce++);
      } while (hash.substr(0, difficulty) !== Array(difficulty + 1).join('0'));

      nonce -= 1; // last nonce was post-incremented

      postMessage({
        hash,
        data,
        difficulty,
        nonce,
      });
    });
  }.toString();
}

