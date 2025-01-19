import { process } from './proof-of-work.mjs';

// from Xeact
const u = (url = "", params = {}) => {
  let result = new URL(url, window.location.href);
  Object.entries(params).forEach((kv) => {
    let [k, v] = kv;
    result.searchParams.set(k, v);
  });
  return result.toString();
};

(async () => {
  const status = document.getElementById('status');
  const image = document.getElementById('image');
  const title = document.getElementById('title');
  const spinner = document.getElementById('spinner');

  status.innerHTML = 'Calculating...';

  const { challenge, difficulty } = await fetch("/.within.website/x/cmd/anubis/api/make-challenge", { method: "POST" })
    .then(r => {
      if (!r.ok) {
        throw new Error("Failed to fetch config");
      }
      return r.json();
    })
    .catch(err => {
      status.innerHTML = `Failed to fetch config: ${err.message}`;
      image.src = "/.within.website/x/cmd/anubis/static/img/sad.webp";
      throw err;
    });

  status.innerHTML = `Calculating...<br/>Difficulty: ${difficulty}`;

  const t0 = Date.now();
  const { hash, nonce } = await process(challenge, difficulty);
  const t1 = Date.now();

  title.innerHTML = "Success!";
  status.innerHTML = `Done! Took ${t1 - t0}ms, ${nonce} iterations`;
  image.src = "/.within.website/x/cmd/anubis/static/img/happy.webp";
  spinner.innerHTML = "";
  spinner.style.display = "none";

  setTimeout(() => {
    const redir = window.location.href;
    window.location.href = u("/.within.website/x/cmd/anubis/api/pass-challenge", { response: hash, nonce, redir, elapsedTime: t1 - t0, difficulty });
  }, 2000);
})();