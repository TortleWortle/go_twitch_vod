# Twitch vod downloader written in Go
Download vods from twitch.
## usage

`tvod help <command>` for available commands.

`tvod info <video_id>` for quality options.

### Download
`tvod download <video_id>` download source.

`tvod download <video_id> <quality>` download specific quality.
#### Options
`-c, --concurrent` download parts concurrently.

`-o, --out` default is `<video_id>.ts`.

`--tmpdir` directory to store temporary video parts.

## Todo
- [ ] Allow video link instead of only video id
- [ ] Command handler cleanup.
- [ ] Better error handling.
- [ ] More source info: total length, recorded date, segment duration
- [ ] Rough range selection.
- [ ] Add tests