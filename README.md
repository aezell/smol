# smol

A CLI for publishing small, static websites to [Sprites](https://sprites.dev).

Point it at a folder of HTML, CSS, JS, and images and it handles the rest:
creating a Sprite, uploading your files, installing a web server, and serving
your site to the world.

```
smol deploy ./my-portfolio --to my-portfolio
```

That's it. Your site is live.

## Install

### From source

Requires [Go](https://go.dev/dl/) 1.21 or later.

```
git clone https://github.com/aezell/smol.git
cd smol
go build -o smol .
sudo mv smol /usr/local/bin/
```

### Pre-built binaries

Coming soon.

## Quick start

### 1. Log in

```
smol login
```

This opens your browser to authenticate with [Fly.io](https://fly.io) (which
powers Sprites). If you don't have an account, you'll be able to create one
during the flow.

If you have multiple Fly.io organizations, you can specify which one to use:

```
smol login --org my-org
```

### 2. Deploy

```
smol deploy ./my-site --to my-site
```

This will:

- Create a Sprite named `my-site` (if it doesn't exist)
- Upload all files from `./my-site`
- Install [Caddy](https://caddyserver.com) as the web server
- Make the site publicly accessible
- Print the live URL

Subsequent deploys to the same site will replace the files and restart the
server. The Sprite and Caddy installation are reused.

### 3. View your site

```
smol open my-site
```

Or just visit the URL printed after deploy.

## Commands

| Command | Description |
|---|---|
| `smol login` | Authenticate with Fly.io |
| `smol logout` | Remove saved credentials |
| `smol create <name>` | Create a new site (Sprite) without deploying |
| `smol deploy <dir> --to <name>` | Upload files and start serving |
| `smol list` | List all your sites |
| `smol status <name>` | Show service status for a site |
| `smol logs <name>` | Show web server logs |
| `smol open <name>` | Open the site URL in your browser |
| `smol destroy <name>` | Tear down a site (with confirmation) |

## How it works

Each site is a [Sprite](https://sprites.dev) -- a lightweight VM on Fly.io's
infrastructure. When you run `smol deploy`:

1. Your local files are packed into a tar archive
2. The archive is uploaded and extracted to `/srv/www` on the Sprite
3. [Caddy](https://caddyserver.com) is configured as a service to serve
   `/srv/www` on port 8080
4. The Sprite's URL is made public

Caddy provides gzip compression, correct MIME types, ETags, and directory
listings out of the box. No configuration needed.

## What makes a good smol site

smol is designed for static sites -- the kind you build with just HTML, CSS,
JavaScript, and images. No build step, no framework, no server-side language.

Some examples:

- A personal homepage or portfolio
- A blog with hand-written HTML
- An art project or interactive experiment
- A documentation site
- A small web app that talks to external APIs

If your site lives in a folder and has an `index.html`, it's a good fit.

## Configuration

Credentials are stored at `~/.config/smol/config.json`. You can override the
token and API URL with environment variables:

| Variable | Description |
|---|---|
| `SMOL_API_TOKEN` | Override the saved API token (useful for CI) |
| `SMOL_API_URL` | Override the API endpoint |

## Tips

- **Hidden files are skipped.** Files and directories starting with `.` (like
  `.git` or `.DS_Store`) are not uploaded.
- **Redeploy is fast.** Only the file upload and server restart happen on
  subsequent deploys. Caddy stays installed.
- **Destroy is permanent.** `smol destroy` deletes the Sprite entirely. There
  is a confirmation prompt unless you pass `--force`.
- **Sites wake on request.** Sprites can go to sleep when idle and wake
  automatically when someone visits the URL. Caddy is configured as a service
  so it starts automatically when the Sprite wakes.

## Project structure

```
smol/
  main.go              # entrypoint
  cmd/
    root.go            # root command and help
    login.go           # smol login (Fly.io browser auth)
    logout.go          # smol logout
    create.go          # smol create
    deploy.go          # smol deploy (tar upload, Caddy setup)
    list.go            # smol list
    status.go          # smol status
    logs.go            # smol logs
    open.go            # smol open
    destroy.go         # smol destroy
  sprite/
    client.go          # Sprite API client (HTTP)
    auth.go            # Fly.io auth flow (CLI sessions, token exchange)
    config.go          # Config file (~/.config/smol/config.json)
```

## License

MIT
