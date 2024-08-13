# sefud: simple, encrypted, file upload & download

## why?

i just wanted to learn how to build and host my own file uploading service.
inspired by 0x0.st and waifuvault.moe and other similar services.
will try to make this project adhere mostly to what 0x0.st offers
so i can use it as an alternative.

## how?

~for devs: run and watch

```sh
air --build.cmd "go build -o tmp/sefud cmd/sefud/main.go" --build.bin "tmp/sefud"
```

## todo?

will be implementing this features one by one:

- [ ] basic:
  - [x] hello world
  - [ ] bootstrap db
- [ ] api:
  - [ ] simple file upload & download
  - [ ] encrypt files with a password
  - [ ] get remaining file age
  - [ ] delete files with a certain token provided after upload
  - [ ] automatically delete files after a certain time (make this specifiable?)
- [ ] webapp: simple react.js or next.js webapp to expose the api to the web
- [ ] database:
  - [ ] sqlite (for now?)
  - [ ] redis (for caching?)
  - [ ] use ULIDs + some URL shortening method?
- [ ] background jobs:
  - [ ] antivirus scanning
  - [ ] nsfw scanning
  - [ ] automate file deletion
- [ ] cli tool
  - ( curl might be just enough + i already wrote [pb.fish](https://github.com/Sardonyx001/pb.fish) for 0x0.st and co.)

## project structure

sefud/
├── cmd/
│ └── sefud/
│ └── main.go
├── server/
│ └── server.go
├── handlers/
│ ├── upload.go
│ ├── download.go
│ └── delete.go
