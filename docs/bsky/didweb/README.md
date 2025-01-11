# Setting up a Bluesky did:web account

- PDS provisioning
  - `civo sshkey create`
  - Terraform to create instance
  - Find ubuntu diskimage
  - Create instance
  - Install Docker and Docker Compose
  - Setup rclone for backups
  - Set AWS route53 zone
  - `engram.within.website`
- Install PDS
  - curl2bash
  - manually patched script to support ubuntu 24.04
  - root is a reserved username, okay
- Test login
  - Can't verify email address
- Making did:web account hosting stuff
  - Point cetacean.club to Tigris
  - tigris bucket
  - Route 53 doesn't allow CNAME at the apex domain
  - Had to use the DNS provider to get the IP addresses of Tigris
  - Nope, tigris wants a CNAME, failing to the.cetacean.club
- did:web account
  - generate privkey / pubkey
    - put in 1password
  - generate did.json
  - upload to tigris
  - I put the DID document in the wrong place
    - fuck I needed to do this:
      ```sh
      aws s3 cp did.json s3://the.cetacean.club/.well-known/did.json
      ```
  - Create invite code with pdsadmin
  - Sign up
    ```json
    {
      "level": 50,
      "time": 1732561457309,
      "pid": 7,
      "hostname": "engram",
      "name": "xrpc-server",
      "status": 400,
      "message": "External handle did not resolve to DID",
      "msg": "error in xrpc method com.atproto.server.createAccount"
    }
    ```
  - add DNS and HTTP verification
    ```hcl
    resource "aws_route53_record" "_atproto_the_cetacean_club" {
      zone_id = data.aws_route53_zone.cetacean_club.zone_id
      name = "_atproto.${tigris_bucket.the-cetacean.bucket}"
      type = "TXT"
      ttl = "3600"
      records = ["did=did:web:the.cetacean.club"]
    }
    ```
    HTTP:
    ```
    did:web:the.cetacean.club
    ```
    Then:
    ```
    aws s3 cp atproto-did s3://the.cetacean.club/.well-known/atproto-did
    ```
  - how to verify/activate your account
    - register account
    - set token in environment
    - Get reccomended did credentials .verificationMethods.atproto
    - s/did:key://
    - Put in did.json .verificationMethod[0].publicKeyMultibase
    - activate account
    - skeet: https://bsky.app/profile/the.cetacean.club/post/3lbsasfpb2s2m
