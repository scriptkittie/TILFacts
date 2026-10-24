# TILFacts

Condenses TIL into soylent TIL: the newest posts from Reddit's
[r/todayilearned](https://www.reddit.com/r/todayilearned/), with the "TIL that..."
taken out, so all that's left is the fact.

## How to use it

**[Download TILFacts](https://github.com/scriptkittie/TILFacts/releases/latest)**:
on that page, under **Assets**, click **TILFacts.exe** (Windows) or the Linux file
for your computer.

1. **Double-click `TILFacts`** (the blue icon with a white label on it).
2. Your web browser opens with today's facts. That's it.

When you're done, just close the browser tab, and TILFacts closes itself a few
seconds later. Or scroll to the bottom, where it asks **"Would you like to close?"**,
and press **Yes**.

Nothing to install. It works on Windows 10 and 11 and needs an internet connection.
On Linux, see [Linux](#linux) below.

### The first time you open it

Windows may show a blue box saying **"Windows protected your PC."** This happens
with small apps that aren't from a big company. To open it anyway:

1. Click **More info**.
2. Click **Run anyway**.

Windows remembers, so you only do this once.

### Linux

1. From the [download page](https://github.com/scriptkittie/TILFacts/releases/latest),
   download **`TILFacts-linux-x64.tar.gz`** (most PCs and laptops) or
   **`TILFacts-linux-arm64.tar.gz`** (Raspberry Pi and other ARM computers).
2. Double-click it and extract it. You get a `TILFacts` folder.
3. Open a terminal in that folder and type `./install.sh`. This adds TILFacts to
   your apps menu; you only do it once.
4. Open your apps menu and click **TILFacts**.

To run it without installing, type `./TILFacts` in that folder instead. To take it
out of the apps menu later, type `./install.sh --remove`. It works on 64-bit Linux
with a desktop (Ubuntu, Fedora, Mint, Debian and so on).

### Sharing it

Send people the [download page](https://github.com/scriptkittie/TILFacts/releases/latest).
Or send the file itself: on Windows, **`TILFacts.exe`** is the only file they need
(by email, a USB stick or a shared folder); on Linux, the matching `.tar.gz`.

## What you can do

- **Save** a fact to keep it. Your saved facts are under **Saved** at the top, where
  **Copy all** copies them as a list.
- **Copy** a fact with its source link, to send to someone.
- Click the **source** (like `en.wikipedia.org`) to read where a fact came from, or
  **Thread** to see what people said about it on Reddit.
- **Search** the facts on the page.
- On the label, change **Serving size** (how many facts at a time) and **Source**
  (newest, most popular today, best of all time, and so on).
- **No, serve more** at the bottom shows the next batch.
- Pick a look in the menu at the top right: Auto (matches your computer's light or
  dark mode), Label, Night, Carton, Oat or Terminal.

Keyboard shortcuts, if you like them: J and K move between facts, S saves, O opens
the source, C opens the thread, Y copies, M serves more, R starts fresh, / searches
and T changes the look.

Hover over a fact to see the original Reddit title. Facts are what people posted on
Reddit; TILFacts doesn't check them, so follow the source before you repeat one at
dinner.

## If something goes wrong

- **"Reddit asked us to slow down"**: Reddit only lets anyone read so much without
  logging in. Wait a minute and press **Try again**.
- **"Couldn't reach Reddit"**: check the internet connection. Some work and school
  networks block Reddit; try another one.
- **"TILFacts has closed"**: it closes itself once no TILFacts tab is open.
  Double-click `TILFacts` again.

---

## About this project

TILFacts was first written on June 30, 2015 as a Python script that printed the
day's TIL posts in a terminal and asked "Would you like to close?" It's now a small
app that shows them as a web page, with a nutrition-facts label for each batch,
and it still asks that question at the end.

### How it works

On May 28, 2026 Reddit shut off the free `.json` feed the 2015 script used, for
anyone not logged in. Its RSS feed still works, but a web page opened from a file
isn't allowed to read it. So `TILFacts.exe` is a small Go program that:

- reads the RSS feed from this computer,
- serves the page (built into the program) at `http://127.0.0.1:8015`, answering
  only that address and only pages it served itself,
- opens your browser to it, and
- quits when you press **Yes**, about 10 seconds after the last TILFacts tab closes,
  or 4 minutes after a tab stops checking in (for example, if the computer sleeps).

It's careful with Reddit:

- Each request asks for 100 facts, and **No, serve more** serves from those. Reading
  100 facts costs one request.
- It reuses each answer for 90 seconds, so refreshing or switching back doesn't
  ask again.
- Requests are at least 2 seconds apart and one at a time.
- When Reddit says "slow down", it waits as long as Reddit asks (up to 10 seconds),
  tries once more, and if that's refused too, doesn't contact Reddit at all for a
  minute.
- The page's check-ins go only to the app on your computer, never to Reddit, and
  every request names itself as TILFacts with a link to this repository.

The RSS feed doesn't include upvote or comment counts, so the page doesn't show
them. Reddit has said RSS may be the next thing it closes. If that happens, the facts
will need Reddit's official API, which means registering an app with Reddit.

### Files

| File | What it is |
|---|---|
| `app/main.go` | The app's code. |
| `app/TILFacts.html` | The page, built into the app. |
| `app/TILFacts.ico`, `app/versioninfo.manifest`, `app/rsrc_windows_amd64.syso` | The icon and Windows details, built into the `.exe`. |
| `app/linux/` | What goes in the Linux download besides the app: icon, `install.sh`, `README.txt`. |
| `.github/workflows/release.yml` | Builds the Windows and Linux apps and publishes them as a release. |

The apps themselves (`TILFacts.exe` and the Linux `.tar.gz` files) aren't kept in
the repository. GitHub builds them for each release.

### Making a release

On GitHub, go to **Releases** → **Draft a new release**, type a new tag such as
`v2.0.0`, and click **Publish release**. (Or push a tag: `git tag v2.0.0` then
`git push origin v2.0.0`.)

GitHub then checks the code, builds `TILFacts.exe`, `TILFacts-linux-x64.tar.gz` and
`TILFacts-linux-arm64.tar.gz`, and adds them to the release with download
instructions. It takes about a minute; watch it under the repository's **Actions**
tab.

### Building it yourself

The release workflow does this for you. To build on your own computer, with
[Go](https://go.dev/dl/) 1.22 or newer, from the `app` folder:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H=windowsgui" -o ../TILFacts.exe
```

If the icon changes, rebuild `rsrc_windows_amd64.syso` with
[rsrc](https://github.com/akavel/rsrc):
`rsrc -ico TILFacts.ico -manifest versioninfo.manifest -arch amd64 -o rsrc_windows_amd64.syso`.

For Linux (use `GOARCH=arm64` for the ARM version), then package it with the
files in `app/linux`:

```
mkdir -p dist/TILFacts
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/TILFacts/TILFacts
cp linux/* dist/TILFacts/ && chmod +x dist/TILFacts/TILFacts dist/TILFacts/install.sh
tar -C dist -czf ../TILFacts-linux-x64.tar.gz TILFacts && rm -rf dist
```

A Mac build works the same way with `GOOS=darwin`, though macOS blocks unsigned
apps more firmly than Windows does. To test against a fake feed instead of Reddit, set
`TILFACTS_FEED_URL` (for example `http://127.0.0.1:9000/r/todayilearned/%s/.rss`).
