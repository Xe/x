# sapientwindex

sapientwindex is a Reddit -> Discord bot. It will monitor a subreddit
(or group of subreddits) and then post any new posts to a given
channel by webhook.

<details>
  <summary>If you know what "Kubernetes" means and you have your own cluster</summary>

If you have a Kubernetes cluster, create a generic secret called
`sapientwindex` in the default namespace with the following fields:

* `DISCORD_WEBHOOK_URL`: The webhook URL to use for Discord
* `REDDIT_USERNAME`: Your reddit username
* `SUBREDDITS`: The subreddits you want to scrape, separated by commas

Run `kubectl apply -f manifest.yaml` and you should be good.

Updating the service is done by restarting the deployment:

```
kubectl rollout restart deployments/sapientwindex
```

Change the namespace of the manifest if running this in a separate
namespace is desired.

</details>

## Prerequisites for self-hosting

In order to host this yourself, you need the following things:

* A linux system that is on 24/7 to run this on (WSL on a gaming tower
  is fine)
* An x86-64 CPU (any computer sold in the last decade is fine)
* A discord webhook for the channel in question
* A reddit account for attributing the bot to yourself
* A list of subreddits to monitor

1. Install [Docker
   Desktop](https://docs.docker.com/desktop/install/windows-install/)
1. Run the following command to start the sapientwindex service:
   ```
   docker run --name sapientwindex -e DISCORD_WEBHOOK_URL=<paste webhook here> -e REDDIT_USERNAME=<your reddit username> -e SUBREDDITS=<list,of,subreddits> -dit ghcr.io/xe/x/sapientwindex:latest
   ```
1. Run the following command to verify that the bot has started:
   ```
   docker logs sapientwindex
   ```
   If you see a message like this:
   ```
   {"time":"2024-05-09T16:39:08.546206894Z","level":"INFO","source":{"function":"main.main","file":"within.website/x/cmd/sapientwindex/main.go","line":28},"msg":"starting up","subreddit":"tulpas","scan_duration":"30s"}
   ```
   then everything is good to go.
   
### Updating the bot

To update the bot, run these commands:

1. Pull the latest version of the sapientwindex container:
   ```
   docker pull ghcr.io/xe/x/sapientwindex:latest
   ```
1. Delete the old version of your sapientwindex container:
   ```
   docker rm -f sapientwindex
   ```
1. Run the start command again:
   ```
   docker run --name sapientwindex -e DISCORD_WEBHOOK_URL=<paste webhook here> -e REDDIT_USERNAME=<your reddit username> -e SUBREDDITS=<list,of,subreddits> -dit ghcr.io/xe/x/sapientwindex:latest
   ```
   
Updates to the bot will be done very infrequently.

## Hosted option

For a nominal fee, I can host a copy of this bot for you on my
homelab. Please [contact me](sapientwindexsales@xeserv.us) to arrange
terms for this hosted option.

## Support

Support is done by [GitHub issues](https://github.com/Xe/x/issues) on
a best-effort basis, with priority to [people subscribed to me on
Patreon](https://patreon.com/cadey). If you deploy this bot in your
community, a subscription would be greatly appreciated.

Support can also be done by email at `sapientwindex@xeserv.us`. Again,
priority support will be given to [my
patrons](https://patreon.com/cadey) with all other support being done
on a best-effort basis.
